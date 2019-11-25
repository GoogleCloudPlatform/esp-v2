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
# This script runs a grpc interop long-running test.
# It requires bazel build following targets:
#   make build-grpc-interop
#   @@com_github_grpc_grpc//test/cpp/interop:interop_client
#   @@com_github_grpc_grpc//test/cpp/interop:stress_test
#   @@com_github_grpc_grpc//test/cpp/interop:metrics_client

SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_PATH}/../../.." && pwd)"
. ${ROOT}/scripts/all-utilities.sh ||
{
  echo "Cannot load Bash utilities"
  exit 1
}

HOST=''
DURATION_IN_HOUR=0
TEST_CASES='empty_unary:10,large_unary:10,'
TEST_CASES+='empty_stream:10,client_streaming:10,ping_pong:20,server_streaming:10,'
TEST_CASES+='status_code_and_message:10,custom_metadata:10'

while getopts :h:l:t: arg; do
  case ${arg} in
    h) HOST="${OPTARG}" ;;
    l) DURATION_IN_HOUR="${OPTARG}" ;;
    *) echo "Invalid option: -${OPTARG}" ;;
  esac
done

[[ -n "${HOST}" ]] || error_exit 'Please specify a host with -h option.'

# Waits for the proxy and backend to start.
HOST_IP=${HOST%:*}
HOST_PORT=${HOST#*:}
echo "HOST_IP: ${HOST_IP}, HOST_PORT: ${HOST_PORT}"
retry $ROOT/bin/interop_client --server_port "${HOST_PORT}" \
  --server_host "${HOST_IP}" \
  || error_exit 'Failed to send one request, the proxy did not start properly.'

DURATION_IN_SEC=$((DURATION_IN_HOUR * 60 * 60))
[[ ${DURATION_IN_SEC} -gt 120 ]] || DURATION_IN_SEC=120

echo "Starts interop stress test at $(date)."
echo "Test duration is: $((DURATION_IN_SEC / 60)) minutes."
echo "Test cases are: ${TEST_CASES}"

# Start a background test client job.
$ROOT/bin/stress_test \
  --server_addresses "${HOST}" \
  --num_channels_per_server 200 \
  --num_stubs_per_channel 1 \
  --test_cases "${TEST_CASES}" 2> /dev/null&
TEST_JOB=$!
trap "kill ${TEST_JOB}" EXIT

START_TIME=$(date +"%s")
FINAL_END_TIME=$((START_TIME + DURATION_IN_SEC))
ONE_ROUND_DURATION_IN_SEC=600
THIS_ROUND_END_TIME=$((START_TIME+ONE_ROUND_DURATION_IN_SEC))

RUN_COUNT=0
FAIL_COUNT=0
export GRPC_GO_LOG_SEVERITY_LEVEL=INFO

detect_memory_leak_init "${HOST_IP}"

while true; do
  CURR_TIME=$(date +"%s")
  sleep 10
  METRIC_RESULT=$("$ROOT/bin/metrics_client" \
    --total_only --metrics_server_address=localhost:8081 2>&1 | tail -1)
  QPS=$(echo ${METRIC_RESULT}|awk '{print $NF}')
  echo "Metric result: ${METRIC_RESULT}"
  echo "Metric report at $((CURR_TIME-START_TIME)) seconds: ${QPS} qps"
  # Count non zero QPS as success.
  [[ ${QPS} -gt 100 ]] || FAIL_COUNt=$((FAIL_COUNT++))

  if [[ $(date +"%s") -ge ${THIS_ROUND_END_TIME} ]] ; then
    RUN_COUNT=$((RUN_COUNT+1))
    detect_memory_leak_check ${RUN_COUNT}
    THIS_ROUND_END_TIME=$((THIS_ROUND_END_TIME+ONE_ROUND_DURATION_IN_SEC))
  fi

  # Break if test has run long enough.
  [[ $(date +"%s") -lt ${FINAL_END_TIME} ]] || break
done

echo "Total test count: ${RUN_COUNT}, failed count: ${FAIL_COUNT}."

# If failure time is more than %5 of total test time, mark failed.
RESULT=0
if [[ ${FAIL_COUNT} -gt $((RUN_COUNT / 20)) && ${FAIL_COUNT} -gt 1 ]] ; then
  RESULT=1
fi

unset GRPC_GO_LOG_SEVERITY_LEVEL

# We fail the test if memory increase is large.
detect_memory_leak_final ${RUN_COUNT}