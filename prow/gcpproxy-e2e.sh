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

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

PROJECT_ID="api_proxy_e2e_test"
UNIQUE_ID=test

cd "${ROOT}"
. ${ROOT}/tests/e2e/scripts/prow-utilities.sh || { echo 'Cannot load Bash utilities';
exit 1; }


function e2eGKE() {
  local OPTIND OPTARG arg
  while getopts :c:g:m:R:t: arg; do
    case ${arg} in
      c) COUPLING_OPTION="$(echo ${OPTARG} | tr '[A-Z]' '[a-z]')" ;;
      g) BACKEND="${OPTARG}" ;;
      m) APIPROXY_IMAGE="${OPTARG}" ;;
      R) ROLLOUT_STRATEGY="${OPTARG}" ;;
      t) TEST_TYPE="$(echo ${OPTARG} | tr '[A-Z]' '[a-z]')" ;;
    esac
  done

  local APIPROXY_SERVICE=$(get_apiproxy_service ${BACKEND})
  local UNIQUE_ID=$(get_unique_id "gke-${TEST_TYPE}-${BACKEND}")

  ${ROOT}/tests/e2e/scripts/e2e-kube.sh  \
    -a "${APIPROXY_SERVICE}"  \
    -t "${TEST_TYPE}"  \
    -g "${BACKEND}"  \
    -B "${BUCKET}"  \
    -m "${APIPROXY_IMAGE}"  \
    -R "${ROLLOUT_STRATEGY}"  \
    -i "${UNIQUE_ID}"
}

if [ ! -d "$GOPATH/bin" ]; then
  mkdir $GOPATH/bin
fi
if [ ! -d "bin" ]; then
  mkdir bin
fi

export GO111MODULE=on
install_e2e_dependencies

# Wait for image build and push.
wait_apiproxiy_image || { echo "Failed in waiting images;";
exit 1; }

download_client_binaries || { echo "Failed in downloading client binaries;";
exit 1; }

echo '======================================================='
echo '=====================   e2e test  ====================='
echo '======================================================='

case ${TEST_CASE} in
  "tight-http-bookstore-managed")
    e2eGKE -c "tight" -t "http" -g "bookstore" -R "managed" -m "$(get_proxy_image_name_with_sha)" -B "${Bucket}"
    ;;
  "tight-grpc-echo-managed")
    e2eGKE -c "tight" -t "grpc" -g "echo" -R "managed" -m "$(get_proxy_image_name_with_sha)" -B "${Bucket}"
    ;;
  "tight-grpc-interop-managed")
    e2eGKE -c "tight" -t "grpc" -g "interop" -R "managed" -m "$(get_proxy_image_name_with_sha)" -B "${Bucket}"
    ;;
  *)
    echo "No such test case ${TEST_CASE}"
    exit 1
    ;;
esac