#!/bin/bash

# Copyright 2018 Google LLC

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Fail on any error.
set -eo pipefail

. ${ROOT}/scripts/all-utilities.sh || {
  echo "Cannot load Bash utilities"
  exit 1
}

# End to End tests common options
function e2e_options() {
  local OPTIND OPTARG arg
  while getopts :a:b:B:m:g:i:k:l:r:R:s:t:v:V: arg; do
    case ${arg} in
      a) APIPROXY_SERVICE="${OPTARG}" ;;
      b) BOOKSTORE_IMAGE="${OPTARG}" ;;
      B) BUCKET="${OPTARG}" ;;
      m) APIPROXY_IMAGE="${OPTARG}" ;;
      g) BACKEND="${OPTARG}" ;;
      i) UNIQUE_ID="${OPTARG}" ;;
      k) API_KEY="${OPTARG}" ;;
      l) DURATION_IN_HOUR="${OPTARG}" ;;
      R) ROLLOUT_STRATEGY="${OPTARG}" ;;
      s) SKIP_CLEANUP='true' ;;
      t) TEST_TYPE="$(echo ${OPTARG} | tr '[A-Z]' '[a-z]')" ;;
      v) VM_IMAGE="${OPTARG}" ;;
      V) ENDPOINTS_RUNTIME_VERSION="${OPTARG}" ;;
      *) e2e_usage "Invalid option: -${OPTARG}" ;;
    esac
  done
  if [[ -z ${API_KEY} ]]; then
    # Setting APY_KEY
    set_api_keys
    API_KEY="${ENDPOINTS_JENKINS_API_KEY}"
    [[ -n "${API_KEY}" ]] || error_exit 'Could not set api key.'
  fi
  if [[ -n "${BUCKET}" ]]; then
    REMOTE_LOG_DIR="gs://${BUCKET}/$(get_sha)/logs/${UNIQUE_ID}"
  fi
}

# Echo and run command, exit on failure
function run_nonfatal() {
  echo ""
  echo "[$(date)] $@"
  "${@}"
  local status=${?}
  if [[ "${status}" != "0" ]]; then
    echo "Command failed with exit status ${status}: ${@}" >&2
  fi
  return ${status}
}

# Echo and run a shell command, exit on failure
function run() {
  run_nonfatal "${@}" || error_exit "command failed"
}

# Run and upload logs
function long_running_test() {
  local host="${1}"
  local duration_in_hour=${2}
  local api_key="${3}"
  local apiproxy_service="${4}"
  local log_dir="${5}"
  local test_id="${6}"
  local run_id="${7}"
  local test_type=''
  [[ ${duration_in_hour} -gt 0 ]] && test_type='long-run-test_'
  local final_test_id="${test_type}${test_id}"
  local log_file="${log_dir}/${final_test_id}.log"
  local json_file="${log_dir}/${final_test_id}.json"
  local status
  local http_code=200
  echo "Running ${BACKEND} long running test on ${host}"
  echo ${host}
  echo ${api_key}
  echo ${apiproxy_service}
  # TODO(jilinxia): add tests for other backends.
  if [[ "${BACKEND}" == 'bookstore' ]]; then
    retry -n 20 check_http_service "${host}:80/shelves" ${http_code}
    # TODO(jilinxia): add tests
    status=${?}
    if [[ ${status} -eq 0 ]]; then
      echo 'Running long running test.'
      run_nonfatal "${SCRIPT_PATH}/linux-test-kb-long-run.sh" \
        -h "${host}" \
        -l "${duration_in_hour}" \
        -a "${api_key}" \
        -s "${apiproxy_service}" 2>&1 | tee "${log_file}"
      status=${PIPESTATUS[0]}
    fi
  elif [[ "${BACKEND}" == 'echo' ]]; then
    retry -n 20 check_grpc_service "${host}:80"
    status=${?}
    if [[ ${status} -eq 0 ]]; then
      run_nonfatal "${SCRIPT_PATH}"/linux-grpc-test-long-run.sh"" \
        -g "${host}" \
        -l "${duration_in_hour}" \
        -a "${api_key}" \
        -s "${apiproxy_service}" 2>&1 | tee "${log_file}"
      status=${PIPESTATUS[0]}
    fi
  elif [[ "${BACKEND}" == 'interop' ]]; then
    run_nonfatal "${SCRIPT_PATH}"/test-grpc-interop.sh \
      -h "${host}:80" \
      -l "${duration_in_hour}" 2>&1 | tee "${log_file}"
    status=${PIPESTATUS[0]}
  fi
  return ${status}
}

# Check for host http return code.
function check_http_service() {
  local host=${1}
  echo $host
  local http_code="${2}"
  local errors="$(mktemp /tmp/curl.XXXXX)"
  local http_response="$(curl -k -m 20 --write-out %{http_code} --silent --output ${errors} ${host})"
  echo "Pinging host: ${host}, response: ${http_response}"
  if [[ "${http_response}" == "${http_code}" ]]; then
    echo "Service is available at: ${host}"
    return 0
  else
    echo "Response body:"
    cat $errors
    echo "Service ${host} is not ready"
    return 1
  fi
}
function check_grpc_service() {
  local host=${1}
  cat <<EOF | "${ROOT}/bin/grpc_echo_client"
server_addr: "${host}"
plans {
  echo {
    request {
      text: "Hello, world!"
    }
  }
}
EOF
  local status=${?}
  if [[ ${status} -eq 0 ]]; then
    echo "Service is available at: ${host}"
  else
    echo "Service ${host} is not ready"
  fi
  return ${status}
}

function get_cluster_host() {
  local COUNT=10
  local SLEEP=15
  for i in $(seq 1 ${COUNT}); do
    local host=$(kubectl get service app -n ${1} | awk '{print $4}' | grep -v EXTERNAL-IP)
    [ '<pending>' != $host ] && break
    echo "Waiting for server external ip. Attempt  #$i/${COUNT}... will try again in ${SLEEP} seconds" >&2
    sleep ${SLEEP}
  done
  if [[ '<pending>' == $host ]]; then
    echo 'Failed to get the GKE cluster host.'
    return 1
  else
    echo "$host"
    return 0
  fi
}

# Convenience method to sed files, works on both linux and mac
function sed_i() {
  # Incompatible sed parameter parsing.
  if sed -i 2>&1 | grep -q 'requires an argument'; then
    sed -i '' "${@}"
  else
    sed -i "${@}"
  fi
}

# Creating and activating a service
function create_service() {
  echo 'Deploying service'
  case "$#" in
    '1')
      local swagger_json="${1}"
      retry -n 3 run ${GCLOUD} endpoints services deploy "${swagger_json}"
      ;;
    '2')
      retry -n 3 run ${GCLOUD} endpoints services deploy ${@:1}
      ;;
    *)
      echo "Invalid arguments ${@} provided for create service"
      return 1;
      ;;
  esac
}