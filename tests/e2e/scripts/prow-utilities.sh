#!/bin/bash

# Copyright 2019 Google LLC

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
  while getopts :a:b:B:m:g:i:k:l:r:p:R:s:t:v:V:f: arg; do
    case ${arg} in
      a) APIPROXY_SERVICE="${OPTARG}" ;;
      b) BOOKSTORE_IMAGE="${OPTARG}" ;;
      B) BUCKET="${OPTARG}" ;;
      m) APIPROXY_IMAGE="${OPTARG}" ;;
      g) BACKEND="${OPTARG}" ;;
      i) UNIQUE_ID="${OPTARG}" ;;
      k) API_KEY="${OPTARG}" ;;
      f) BACKEND_PLATFORM="${OPTARG}" ;;
      l) DURATION_IN_HOUR="${OPTARG}" ;;
      p) PROXY_PLATFORM="${OPTARG}" ;;
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
  local address="${1}"
  local scheme="${2}"
  local port="${3}"
  local duration_in_hour=${4}
  local api_key="${5}"
  local apiproxy_service="${6}"
  local log_dir="${7}"
  local test_id="${8}"
  local run_id="${9}"
  local platform="${10}"
  local host_header="${11}"

  local test_type=''
  [[ ${duration_in_hour} -gt 0 ]] && test_type='long-run-test_'
  local final_test_id="${test_type}${test_id}"
  local log_file="${log_dir}/${final_test_id}.log"
  local json_file="${log_dir}/${final_test_id}.json"
  local status
  local http_code=200
  echo "Running ${BACKEND} long running test on ${address}"
  echo "ESPv2 listening at ${scheme}://${address}:${port}"
  echo ${api_key}
  echo ${apiproxy_service}
  case "${BACKEND}" in
    'bookstore')
      retry -n 20 check_http_service "${scheme}://${address}:${port}/shelves" ${http_code} "${host_header}"
      status=${?}
      if [[ ${status} -eq 0 ]]; then
        echo 'Running long running test.'
        run_nonfatal "${SCRIPT_PATH}/linux-test-kb-long-run.sh"  \
          -m "${scheme}" \
          -h "${address}"  \
          -p "${port}"  \
          -l "${duration_in_hour}"  \
          -a "${api_key}"  \
          -s "${apiproxy_service}" \
          -t "${platform}" \
          -r "${host_header}"   \
          2>&1 | tee "${log_file}" \
          || status=${?}
      fi
      ;;
    'echo')
      retry -n 20 check_grpc_service "${address}:${port}"
      status=${?}
      if [[ ${status} -eq 0 ]]; then
        run_nonfatal "${SCRIPT_PATH}"/linux-grpc-test-long-run.sh""  \
          -g "${address}"  \
          -l "${duration_in_hour}"  \
          -a "${api_key}"  \
          -s "${apiproxy_service}" 2>&1 | tee "${log_file}" \
          || status=${?}
      fi
      ;;
    'interop')
      status=0
      run_nonfatal "${SCRIPT_PATH}"/test-grpc-interop.sh  \
        -h "${address}:${port}"  \
        -l "${duration_in_hour}" 2>&1 | tee "${log_file}" \
        || status=${?}
      ;;
    *)
      echo "Invalid backend ${BACKEND}"
      return 1 ;;
  esac

  create_status_file  \
    -f "${json_file}"  \
    -s ${status}  \
    -t "${final_test_id}"  \
    -r "${run_id}"  \
    || { echo "Could not create ${json_file}.";
  return 1; }

  return ${status}
}

# Check for host http return code.
function check_http_service() {
  local host=${1}
  echo $host
  local http_code="${2}"
  local host_header="${3}"
  local errors="$(mktemp /tmp/curl.XXXXX)"
  if [[ -n ${host_header} ]];
  then
   local http_response="$(curl -k -m 20 --write-out %{http_code} --silent --output ${errors} ${host} -H "HOST:${host_header}")"
  else
    local http_response="$(curl -k -m 20 --write-out %{http_code} --silent --output ${errors} ${host})"
  fi
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
  cat << EOF | "${ROOT}/bin/grpc_echo_client"
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

# Upload logs remote directory
function upload_logs() {
  local remote_dir="${1}"
  local log_dir="${2}"

  echo "Uploading content of ${log_dir} to ${remote_dir}"
  retry -n 3 ${GSUTIL} -h 'Content-Type:text/plain' -m cp -r  \
    "${log_dir}" "${remote_dir}"  \
    || echo "Failed to upload ${log_dir}"
}

function wait_apiproxy_image() {
  local ENVOY_IMAGE_SHA_NAME=$(get_envoy_image_name_with_sha)
  local PROXY_IMAGE_SHA_NAME=$(get_proxy_image_name_with_sha)
  local SERVERLESS_IMAGE_SHA_NAME=$(get_serverless_image_name_with_sha)

  printf "Waiting for the following images:\n  %s\n  %s\n  %s\n" \
    "${ENVOY_IMAGE_SHA_NAME}" \
    "${PROXY_IMAGE_SHA_NAME}" \
    "${SERVERLESS_IMAGE_SHA_NAME}"

  # Wait 60 minutes (in case of cold cache). Generally only takes 15 minutes.
  local WAIT_IMAGE_TIMEOUT=3600
  local SLEEP_UNIT=60

  while true; do

    gcloud container images describe "${ENVOY_IMAGE_SHA_NAME}"  \
      && gcloud container images describe "${PROXY_IMAGE_SHA_NAME}"  \
      && gcloud container images describe "${SERVERLESS_IMAGE_SHA_NAME}"  \
      && { echo "Found the images";
    break; }

    if [ ${WAIT_IMAGE_TIMEOUT} -gt 0 ]; then
      echo "Waiting images with ${WAIT_IMAGE_TIMEOUT}s left"
      sleep ${SLEEP_UNIT}
      WAIT_IMAGE_TIMEOUT=$((WAIT_IMAGE_TIMEOUT - SLEEP_UNIT))
    else
      return 1;
    fi
  done
  return 0;
}

function download_client_binaries() {
  gsutil -m cp "gs://apiproxy-testing-presubmit-binaries/*" ${ROOT}/bin/
  mv ${ROOT}/bin/api_descriptor.pb ${ROOT}/tests/endpoints/grpc_echo/proto/api_descriptor.pb
  chmod +x ${ROOT}/bin/*
}

function get_apiproxy_service() {
  if [[ "${1}" == "bookstore" ]]; then
    echo "bookstore.endpoints.cloudesf-testing.cloud.goog"
  elif [[ "${1}" == "echo" ]]; then
    echo "echo.endpoints.cloudesf-testing.cloud.goog"
  elif [[ "${1}" == "interop" ]]; then
    echo "interop.endpoints.cloudesf-testing.cloud.goog"
  else
    echo "Service ${1} is not supported."
    return 1
  fi
}

function install_e2e_dependencies() {
  pip install python-gflags
}