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

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROJECT_ID="api_proxy_e2e_test"

cd "${ROOT}"
. ${ROOT}/tests/e2e/scripts/prow-utilities.sh || { echo 'Cannot load Bash utilities';
exit 1; }
gcloud components update

function runE2E() {
  local OPTIND OPTARG arg
  while getopts :f:p:c:g:m:R:t: arg; do
    case ${arg} in
      f) local backend_platform="${OPTARG}" ;;
      p) local platform="${OPTARG}" ;;
      c) local coupling_option="$(echo ${OPTARG} | tr '[A-Z]' '[a-z]')" ;;
      g) local backend="${OPTARG}" ;;
      m) local apiproxy_image="${OPTARG}" ;;
      R) local rollout_strategy="${OPTARG}" ;;
      t) local test_type="$(echo ${OPTARG} | tr '[A-Z]' '[a-z]')" ;;
    esac
  done

  local apiproxy_service=$(get_apiproxy_service ${backend})
  local unique_id=$(get_unique_id "gke-${test_type}-${backend}")

  local platform_deploy_script="${ROOT}/tests/e2e/scripts/${platform}/deploy.sh"
  echo "Deploying on platform ${platform}"

  ${platform_deploy_script}  \
    -a "${apiproxy_service}"  \
    -t "${test_type}"  \
    -g "${backend}"  \
    -m "${apiproxy_image}"  \
    -R "${rollout_strategy}"  \
    -i "${unique_id}"  \
    -B "${BUCKET}"  \
    -l "${DURATION_IN_HOUR}" \
    -f "${backend_platform}"
}

if [ ! -d "$GOPATH/bin" ]; then
  mkdir $GOPATH/bin
fi
if [ ! -d "bin" ]; then
  mkdir bin
fi

export GO111MODULE=on

# Wait for image build and push.
wait_apiproxy_image || { echo "Failed in waiting images;";
exit 1; }

download_client_binaries || { echo "Failed in downloading client binaries;";
exit 1; }

echo '======================================================='
echo '=====================   e2e test  ====================='
echo '======================================================='
case ${TEST_CASE} in
  "tight-http-bookstore-managed")
    runE2E -p "gke" -c "tight" -t "http" -g "bookstore" -R "managed" -m "$(get_proxy_image_name_with_sha)"
    ;;
  "tight-grpc-echo-managed")
    runE2E -p "gke" -c "tight" -t "grpc" -g "echo" -R "managed" -m "$(get_proxy_image_name_with_sha)"
    ;;
  "tight-grpc-interop-managed")
    runE2E -p "gke" -c "tight" -t "grpc" -g "interop" -R "managed" -m "$(get_proxy_image_name_with_sha)"
    ;;
  "cloud-run-cloud-run-http-bookstore")
    runE2E -p "cloud-run" -f "cloud-run" -t "http" -g "bookstore" -R "managed" -m "$(get_serverless_image_name_with_sha)"
    ;;
  "cloud-run-cloud-function-http-bookstore")
    runE2E -p "cloud-run" -f "cloud-function" -t "http" -g "bookstore" -R "managed" -m "$(get_serverless_image_name_with_sha)"
    ;;
  *)
    echo "No such test case ${TEST_CASE}"
    exit 1
    ;;
esac