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

TOOLS_BUCKET="apiproxy_tools"
PLATFORM="GCE"
ESP_RELEASE_PROJECT="endpoints-release"
GCSRUNNER_RELEASE_PROJECT="espv2-gcsrunner-release"

# Setting SUDO if not running as root.
if [[ $UID -ne 0 ]]; then
  SUDO=sudo
fi

# Library of useful utilities.
function set_gcloud() {
  export GCLOUD="$(which gcloud)" || export GCLOUD='/usr/lib/google-cloud-sdk/bin/gcloud'
  export GSUTIL="$(which gsutil)" || export GSUTIL='/usr/lib/google-cloud-sdk/bin/gsutil'
}

function set_bazel() {
  export BAZEL="$(which bazel)" || export BAZEL='/usr/local/bin/bazel'
}

function set_wrk() {
  export WRK="$(which wrk)" || export WRK='/usr/local/bin/wrk'
}

set_bazel
set_gcloud
set_wrk

# Exit with a message and an exit code.
# Arguments:
#   $1 - string with an error message
#   $2 - exit code, defaults to 1
function error_exit() {
  # ${BASH_SOURCE[1]} is the file name of the caller.
  echo "${BASH_SOURCE[1]}: line ${BASH_LINENO[0]}: ${1:-Unknown Error.} (exit ${2:-1})" 1>&2
  exit ${2:-1}
}

# Tag a source image with a target image which shouldn't exist.
# Arguments:
#   $1 - the source image.
#   $2 - the target image.
function docker_tag() {
  ${GCLOUD} docker -- pull "${1}" 2>/dev/null || error_exit "Cannot pull image: ${1}"
  ${GCLOUD} docker -- pull "${2}" 2>/dev/null && error_exit "Trying to override an existing image: ${2}"
  docker tag "${1}" "${2}" || error_exit "Failed to tag ${1} with ${2}"
}

# Tag -f a source image with a target image which may exist already.
# Arguments:
#   $1 - the source image.
#   $2 - the target image.
function docker_tag_f() {
  ${GCLOUD} docker -- pull "${1}" || error_exit "Cannot pull image: ${1}"
  docker tag -f "${1}" "${2}" || error_exit "Failed to tag ${1} with ${2}"
}

# Retries a command with an exponential back-off.
# The back-off base is a constant 3/2
# Options:
#   -n Maximum total attempts (0 for infinite, default 10)
#   -t Maximum time to sleep between retries (default 60)
#   -s Initial time to sleep between retries. Subsequent retries
#      subject to exponential back-off up-to the maximum time.
#      (default 5)
function retry() {
  local OPTIND OPTARG ARG
  local COUNT=10
  local SLEEP=5 MAX_SLEEP=60
  local MUL=3 DIV=2 # Exponent base multiplier and divisor
  # (Bash doesn't do floats)

  while getopts ":n:s:t:" ARG; do
    case ${ARG} in
      n) COUNT=${OPTARG} ;;
      s) SLEEP=${OPTARG} ;;
      t) MAX_SLEEP=${OPTARG} ;;
      *) echo "Unrecognized argument: -${OPTARG}" ;;
    esac
  done

  shift $((OPTIND - 1))

  # If there is no command, abort early.
  [[ ${#} -le 0 ]] && { echo "No command specified, aborting.";
  return 1; }

  local N=1 S=${SLEEP} # S is the current length of sleep.
  while :; do
    echo "${N}. Executing ${@}"
    "${@}" && { echo "Command succeeded.";
    return 0; }

    [[ ( ( COUNT -le 0 || N -lt COUNT ) ) ]]  \
      || { echo "Command '${@}' failed ${N} times, aborting.";
    return 1; }

    if [[ ( ( S -lt MAX_SLEEP ) ) ]]; then
      # Must always count full exponent due to integer rounding.
      ((S = SLEEP * ( MUL ** ( N - 1 ) ) / ( DIV ** ( N - 1 ) )))
    fi

    ((S = ( S < MAX_SLEEP ) ? S : MAX_SLEEP))

    echo "Command failed. Will retry in ${S} seconds."
    sleep ${S}

    ((N ++))
  done
}

# Download api Keys from Cloud storage and source the file.
function set_api_keys() {
  local api_key_directory="$(mktemp -d)"
  $GSUTIL cp gs://apiproxy-testing-client-secret-files/api_keys  \
    "${api_key_directory}/api_keys"  \
    || error_exit "Failed to download API key file."

  source "${api_key_directory}/api_keys"
}

# Download test-client keys from Cloud storage
function get_test_client_key() {
  local remote_file_name=$1
  local key_path=$2
  $GSUTIL  cp "gs://apiproxy-testing-client-secret-files/$remote_file_name" "$key_path"
  echo -n "$key_path"
  return 0
}

# Creates a simple Json Status file
function create_status_file() {
  local OPTIND OPTARG ARG
  local file_path=''
  local test_status=''
  local test_id=''
  local run_id=''

  while getopts :f:s:t:r: ARG; do
    case ${ARG} in
      f) file_path="${OPTARG}" ;;
      s) test_status=${OPTARG} ;;
      t) test_id="${OPTARG}" ;;
      r) run_id="${OPTARG}" ;;
      *) echo "Unrecognized argument: -${OPTARG}" ;;
    esac
  done

  [[ -n "${file_path}" ]] || { echo 'File path is not set.';
  return 1; }
  [[ -n "${test_status}" ]] || { echo 'Status is not set.';
  return 1; }
  [[ -n "${test_id}" ]] || { echo 'Test id is not set.';
  return 1; }
  [[ -n "${run_id}" ]] || { echo 'Run id is not set.';
  return 1; }

  mkdir -p "$(dirname "${file_path}")"

  cat >"${file_path}" << __EOF__
{
  "scriptStatus": ${test_status},
  "testId": "${test_id}",
  "date": "$(date +%s)",
  "runId": "${run_id}",
  "headCommitHash": "$(get_sha)"
}
__EOF__
  return 0
}

# Uses 3 functions to detect memory leak for stress tests.
# 1) call detect_memory_leak_init() before your loop
# 2) call detect_memory_leak_check() for each iteration
# 3) call detect_memory_leak_final() at the end.
function detect_memory_leak_init() {
  local host=${1}
  # host format has to be: proto://host:port.
  STATUS_SERVER="http://${host}:8001"
  echo "STATUS_SERVER: ${STATUS_SERVER}"
}

function detect_memory_leak_check() {
  local run_count=${1}
  local local_json="$(mktemp /tmp/XXXXXX.json)"

  curl "${STATUS_SERVER}/memory" >"${local_json}"

  python -m json.tool "${local_json}"
  local curr_usage=$(python -c  \
    "import json, sys;obj = json.load(open(\"${local_json}\"));print obj['allocated']")
  rm "${local_json}"
  [[ -n "${curr_usage}" ]] || { echo "Could not extract memory usage";
  return 1; }

  if [[ ${run_count} -eq 1 ]]; then
    LAST_MEMORY_USAGE=${curr_usage}
    echo "Start Memory Usage (Bytes): ${curr_usage}."
    return 0;
  fi
  local delta=$((curr_usage - LAST_MEMORY_USAGE))
  LAST_MEMORY_USAGE=${curr_usage}
  echo "Memory increased at test run ${run_count}: ${delta}."

  # Each run of test sends about 100K requests.
  # It is 1MB leak if there is a 10 bytes leak per request
  local threshold=$((2**20))
  if [[ ${delta} -gt ${threshold} ]]; then
    MEMLEAK_COUNT=$((MEMLEAK_COUNT+1))
    echo "************ Memory usage increased (>${threshold} Bytes) *************"
  fi
}

function detect_memory_leak_final() {
  local run_count=${1}
  [[ ${run_count} -gt 1 ]]  \
    || { echo "Only run test once.";
  return 0; }

  # It is a leak if 70% of rounds has increased memory usage.
  if [[ ${MEMLEAK_COUNT} -gt $((run_count * 7 / 10)) ]]; then
    echo "************ Memory leak is found!!! *************"
    return 1
  fi
}

# Extract key from test env json created from
# script/create-test-env-json script.
# As an example, from this json
# {
# "test": "test-id",
# "run_id": "test-id-1902",
# "run_description": "Commit message",
# "owner": "John Doe"
# }
function extract_key_from_test_env_file() {
  local key="${1}"
  local json_path="${2}"
  cat "${json_path}"  \
    | python -c "import json,sys;obj=json.load(sys.stdin);print obj['${key}']"  \
    || { echo "Could not extract ${key} from ${json_path}";
  return 1; }
}

function update_tool() {
  local tool_name="${1}"
  local tool_version="${2}"
  local local_path="${3}"
  local remote_path="gs://${TOOLS_BUCKET}/${tool_name}/${tool_version}/${PLATFORM}/${tool_name}"

  [[ -z "${TOOLS_BUCKET}" ]] && return 1
  echo "Uploading ${local_path} to ${remote_path}."
  ${GSUTIL} cp "${local_path}" "${remote_path}"  \
    || { echo "Failed to upload ${tool_name} to ${TOOLS_BUCKET}";
  return 1; }
  return 0
}

function get_tool() {
  local tool_name="${1}"
  local tool_version="${2}"
  local local_path="${3}"
  local remote_path="gs://${TOOLS_BUCKET}/${tool_name}/${tool_version}/${PLATFORM}/${tool_name}"

  [[ -z "${TOOLS_BUCKET}" ]] && return 1
  echo "Downloading ${remote_path} to ${local_path}."
  ${GSUTIL} cp "${remote_path}" "${local_path}"  \
    || { echo "Failed to upload ${tool_name} to ${TOOLS_BUCKET}";
  return 1; }
  return 0
}

function get_sha() {
  echo "$(git rev-parse --verify HEAD)"
}

function get_envoy_image_name() {
  echo -n 'gcr.io/cloudesf-testing/envoy-binary'
}

function get_gcsrunner_image_name() {
  echo -n 'gcr.io/cloudesf-testing/gcsrunner'
}

function get_proxy_image_name() {
  echo -n 'gcr.io/cloudesf-testing/apiproxy'
}

function get_serverless_image_name() {
  echo -n 'gcr.io/cloudesf-testing/apiproxy-serverless'
}

function get_proxy_image_release_name() {
  echo -n "gcr.io/${ESP_RELEASE_PROJECT}/endpoints-runtime"
}

function get_serverless_image_release_name() {
  echo -n "gcr.io/${ESP_RELEASE_PROJECT}/endpoints-runtime-serverless"
}

function get_gcsrunner_image_release_name() {
  echo -n "gcr.io/${GCSRUNNER_RELEASE_PROJECT}/gcsrunner"
}


function get_tag_name() {
  local tag_format="%H"
  tag_name="$(git show -q HEAD --pretty=format:"${tag_format}")"
  echo -n "${tag_name}"
}

function get_envoy_image_name_with_sha() {
  # Generic docker image format. https://git-scm.com/docs/git-show.
  echo -n "$(get_envoy_image_name):$(get_tag_name)"
  return 0
}

function get_gcsrunner_image_name_with_sha() {
  # Generic docker image format. https://git-scm.com/docs/git-show.
  echo -n "$(get_gcsrunner_image_name):$(get_tag_name)"
  return 0
}

function get_proxy_image_name_with_sha() {
  # Generic docker image format. https://git-scm.com/docs/git-show.
  echo -n  "$(get_proxy_image_name):$(get_tag_name)"
  return 0
}

function get_serverless_image_name_with_sha() {
  # Generic docker image format. https://git-scm.com/docs/git-show.
  echo -n  "$(get_serverless_image_name):$(get_tag_name)"
  return 0
}


function get_unique_id() {
  local uuid=$(cat /dev/urandom | tr -dc 'a-z0-9' | fold -w 32 | head -n 1)
  echo "${1}-${uuid}"
  return 0
}

# Attempts to setup bazel to use a remote cache
# On non-Prow hosts, the remote cache will not be used
function try_setup_bazel_remote_cache() {

  local prow_job_id=$1
  local docker_image_name=$2
  local root_dir=$3
  local presubmit_test_case=$4
  local gcp_project_id="cloudesf-testing"
  local silo_uuid="v5"

  # Determine if this job is running on a non-Prow host. All Prow jobs must have this env var
  # https://github.com/kubernetes/test-infra/blob/master/prow/jobs.md#job-environment-variables
  if [[ -z "${prow_job_id}" ]]; then
    echo "PROW_JOB_ID not set. Script continuing without bazel remote cache on non-Prow host.";
    return 0;
  fi
  echo "Setting up remote bazel cache on Prow host. Prow Job ID: ${prow_job_id}"

  # Variables must be set to determine cache location
  if [[ -z "${gcp_project_id}" ]]; then
    echo "PROJECT_ID not set, cannot determine remote cache location.";
    exit 2;
  fi
  echo "Cache Project ID: ${gcp_project_id}"
  if [[ -z "${docker_image_name}" ]]; then
    echo "IMAGE not set, cannot determine cache silo.";
    exit 2;
  fi

  # Cache silo name is determined by image_name-UUID-[empty|asan|tsan].
  # This works because the environment is consistent in any containers of this docker image.
  # Also, replace special characters that RBE does not accept with a '/'
  local cache_silo
  cache_silo="$(echo "${docker_image_name}-uuid-${silo_uuid}-${presubmit_test_case}" | tr @: /)"
  echo "Original Image Name: ${docker_image_name}"
  echo "Cache Silo Name: ${cache_silo}"

  # Append Prow bazelrc to workspace's bazelrc so that all commands will default to using it
  cat "${root_dir}/prow/.bazelrc" >>"${root_dir}/.bazelrc"

  # Replace templates with real environment variables
  # Use @ as delimiter because docker image name may have '/'
  sed -i -e "s@CACHE_SILO_NAME@${cache_silo}@g" ${root_dir}/.bazelrc
  sed -i -e "s@CACHE_PROJECT_ID@${gcp_project_id}@g" ${root_dir}/.bazelrc
}

function envoy_binary_gcs_path() {
  echo -n "gs://apiproxy-testing-envoy-binaries/$(get_sha)"
}

function upload_envoy_binary() {
  gsutil -m cp ${ROOT}/bin/envoy $(envoy_binary_gcs_path)
}

function download_envoy_binary() {
  gsutil -m cp $(envoy_binary_gcs_path)  ${ROOT}/bin/envoy
}
