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

set -eo pipefail

if [[ "$(uname)" != "Linux" ]]; then
  echo "Run on Linux only."
  #  exit 1
fi

SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_PATH}/../../.." && pwd)"
. ${ROOT}/scripts/all-utilities.sh ||
{
  echo "Cannot load Bash utilities"
  exit 1
}

API_KEY=''
SERVICE_NAME="echo.endpoints.cloudesf-testing.cloud.goog"
SCHEME=''
HOST=''
PORT=''
PLATFORM=''
DURATION_IN_HOUR=0

REQUEST_COUNT=10000
USE_SSL=false

while getopts :a:m:h:p:l:s:t: arg; do
  case ${arg} in
    a) API_KEY="${OPTARG}" ;;
    m) SCHEME="${OPTARG}" ;;
    h) HOST="${OPTARG}" ;;
    p) PORT="${OPTARG}" ;;
    l) DURATION_IN_HOUR="${OPTARG}" ;;
    s) SERVICE_NAME="${OPTARG}" ;;
    t) PLATFORM="${OPTARG}" ;;
    *) echo "Invalid option: -${OPTARG}" ;;
  esac
done

if ! [[ -n "${API_KEY}" ]]; then
  set_api_keys
  API_KEY="${ENDPOINTS_JENKINS_API_KEY}"
fi

[[ -n "${HOST}" ]] || error_exit 'Please specify a host with -h option.'

# Nginx default max_concurrent_streams is 128.
# If CONCURRENT > 128, some requests will fail with RST_STREAM.
CONCURRENT_TYPES=(50 128)
CONCURRENT_TYPES_SIZE=${#CONCURRENT_TYPES[@]}
STREAM_COUNT_TYPES=(10 50 100)
STREAM_COUNT_TYPES_SIZE=${#STREAM_COUNT_TYPES[@]}
# For now, total transfer size per stream direction should not pass 1 MB
RANDOM_PAYLOAD_SIZE_TYPES=(1024 20000)
RANDOM_PAYLOAD_SIZE_TYPES_SIZE=${#RANDOM_PAYLOAD_SIZE_TYPES[@]}
ALL_CONFIG_TYPES=$((CONCURRENT_TYPES_SIZE * STREAM_COUNT_TYPES_SIZE * RANDOM_PAYLOAD_SIZE_TYPES_SIZE))

function generate_run_config() {
  local run=${1}
  CONCURRENT=${CONCURRENT_TYPES[$((run % CONCURRENT_TYPES_SIZE))]}
  echo concurrent="${CONCURRENT}"

  STREAM_COUNT_RUN=$((run / CONCURRENT_TYPES_SIZE))
  STREAM_COUNT=${STREAM_COUNT_TYPES[$((STREAM_COUNT_RUN % STREAM_COUNT_TYPES_SIZE))]}
  echo stream_count="${STREAM_COUNT}"

  RANDOM_PAYLOAD_SIZE_RUN=$((STREAM_COUNT_RUN / STREAM_COUNT_TYPES_SIZE))
  RANDOM_PAYLOAD_SIZE=${RANDOM_PAYLOAD_SIZE_TYPES[$((RANDOM_PAYLOAD_SIZE_RUN % RANDOM_PAYLOAD_SIZE_TYPES_SIZE))]}
  echo random_payload_size="${RANDOM_PAYLOAD_SIZE}"
}

function grpc_test_pass_through() {
  echo "Starting grpc pass through stress test at $(date)."

  # Determine if we need SSL or not
  if [ "${SCHEME}" == "https" ]; then
    USE_SSL='true'
  fi

  local tmp_file="$(mktemp)"
  local failures=0
  for run in $(seq 1 ${ALL_CONFIG_TYPES}); do
    generate_run_config $((run - 1))

    # Generating token for each run, that they expire in 1 hour.
    local AUTH_TOKEN=$("${ROOT}/tests/e2e/scripts/gen-auth-token.sh" -a "${SERVICE_NAME}")

    (set -x; python "${ROOT}/tests/e2e/client/grpc/grpc_stress_input.py" \
        --server="${HOST}:${PORT}" \
        --allowed_failure_rate=0.3 \
        --api_key="${API_KEY}" \
        --auth_token="${AUTH_TOKEN}" \
        --request_count="${REQUEST_COUNT}" \
        --concurrent="${CONCURRENT}" \
        --requests_per_stream="${STREAM_COUNT}" \
        --random_payload_max_size="${RANDOM_PAYLOAD_SIZE}" \
        --use_ssl="${USE_SSL}" \
      --random_payload_max_size="${RANDOM_PAYLOAD_SIZE}" >"${tmp_file}")
    # gRPC test client occasionally aborted. Retry up to 5 times.

    local count=0
    while :; do
      cat "${tmp_file}" | "${ROOT}/bin/grpc_echo_client"
      local status=$?
      if [[ "$status" == "0" ]]; then
        break
      fi
      if [[ "$status" != "134" ]] || [[ ${count} -gt 5 ]]; then
        ((failures++))
        break
      fi
      ((count++))
      echo "Test client crashed, Retry the test: ${count}"
    done
  done
  return $failures
}


function grpc_test_transcode() {
  echo "Starting grpc transcode stress test at $(date)."

  # Generating token for each run, that they expire in 1 hour.
  local AUTH_TOKEN=$("${ROOT}/tests/e2e/scripts/gen-auth-token.sh" -a "${SERVICE_NAME}")

  (set -x; python ${ROOT}/tests/e2e/client/apiproxy_client.py \
      --test=stress \
      --host="${SCHEME}://${HOST}:${PORT}" \
      --api_key="${API_KEY}" \
      --auth_token="${AUTH_TOKEN}" \
      --test_data="${ROOT}/tests/e2e/testdata/grpc_echo/grpc_test_data.json" \
    --root="${ROOT}")
}

function grpc_test_transcode_fuzzing() {
  STATUS_HOST="http://${HOST}:8001"

  # Generating token for each run, that they expire in 1 hour.
  local AUTH_TOKEN=$("${ROOT}/tests/e2e/scripts/gen-auth-token.sh" -a "${SERVICE_NAME}")

  echo "Starting grpc transcode fuzz test at $(date)."
  (set -x; python ${ROOT}/tests/e2e/client/apiproxy_transcoding_fuzz_test.py \
      --address="${SCHEME}://${HOST}:${PORT}" \
      --status_address="${STATUS_HOST}" \
      --api_key="${API_KEY}" \
      --auth_token="${AUTH_TOKEN}" \
    --runs=1)
}

END_TIME=$(date +"%s")
END_TIME=$((END_TIME + DURATION_IN_HOUR * 60 * 60))
RUN_COUNT=0
GRPC_STRESS_FAILURES=0
HTTP_STRESS_FAILURES=0

if [ "$PLATFORM" = "gke" ]; then
  status_server_init ${HOST}
fi

# ${ROOT}/tests/client/esp_client.py needs to run at that folder.
pushd ${ROOT}/tests/e2e/client > /dev/null

while true; do
  RUN_COUNT=$((RUN_COUNT + 1))
  echo "Starting test run ${RUN_COUNT} at $(date)."
  echo "Failures so far: pass-through: ${GRPC_STRESS_FAILURES}, transcode: ${HTTP_STRESS_FAILURES}."
  #######################
  # Insert tests here
  #######################
  RUN_COUNT=$((RUN_COUNT++))

  grpc_test_pass_through || ((GRPC_STRESS_FAILURES++))

  grpc_test_transcode || ((HTTP_STRESS_FAILURES++))

  if [ "$PLATFORM" = "gke" ]; then
    grpc_test_transcode_fuzzing|| ((HTTP_STRESS_FAILURES++))
    detect_memory_leak_check ${RUN_COUNT}
  fi

  # Break if test has run long enough.
  [[ $(date +"%s") -lt ${END_TIME} ]] || break
done
popd > /dev/null

echo "Finished ${RUN_COUNT} test runs."
echo "Failures: pass-through: ${GRPC_STRESS_FAILURES}, transcode: ${HTTP_STRESS_FAILURES}."

MEMORY_LEAK=0
if [ "$PLATFORM" = "gke" ]; then
  # We fail the test if memory increase is large.
  detect_memory_leak_final ${RUN_COUNT} && MEMORY_LEAK=0 || MEMORY_LEAK=1
fi
# Only mark test as failed if any pass-through tests failed.
# This is to be consistent with other http stress tests.
# All failure will be analyzed by release-engineers.
exit $((${GRPC_STRESS_FAILURES} + ${HTTP_STRESS_FAILURES} + ${MEMORY_LEAK}))
