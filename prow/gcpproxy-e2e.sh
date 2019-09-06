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
. ${ROOT}/scripts/all-utilities.sh || { echo 'Cannot load Bash utilities'; exit 1; }

echo '======================================================='
echo '===================== Setup Cache ====================='
echo '======================================================='
try_setup_bazel_remote_cache "${PROW_JOB_ID}" "${IMAGE}" "${ROOT}"

echo '======================================================='
echo '=====================   e2e test  ====================='
echo '======================================================='

if [ ! -d "$GOPATH/bin" ]; then
  mkdir $GOPATH/bin
fi
if [ ! -d "bin" ]; then
  mkdir bin
fi

# libraries for go build
curl https://glide.sh/get | sh

# dependencies for envoy build
apt-get update && \
    apt-get -y install libtool cmake automake ninja-build curl unzip libssl-dev

pip install python-gflags

function getApiProxyService() {
  if [[ "${1}" == "bookstore" ]]; then
    echo "bookstore.endpoints.cloudesf-testing.cloud.goog"
    return 0
  elif [[ "${1}" == "echo" ]]; then
    echo "echo.endpoints.cloudesf-testing.cloud.goog"
    return 0
  else
    echo "Service ${1} is not supported."
    return 1
  fi
}

function getUniqueID() {
  local uuid=$(cat /dev/urandom | tr -dc 'a-z0-9' | fold -w 32 | head -n 1)
  echo "${1}-${uuid}"
  return 0
}

function e2eGKE() {
  local OPTIND OPTARG arg
  while getopts :c:g:m:R:t: arg; do
    case ${arg} in
      c) COUPLING_OPTION="$(echo ${OPTARG} | tr '[A-Z]' '[a-z]')";;
      g) BACKEND="${OPTARG}";;
      m) APIPROXY_IMAGE="${OPTARG}";;
      R) ROLLOUT_STRATEGY="${OPTARG}";;
      t) TEST_TYPE="$(echo ${OPTARG} | tr '[A-Z]' '[a-z]')";;
    esac
  done

  local APIPROXY_SERVICE=$(getApiProxyService ${BACKEND})
  local UNIQUE_ID=$(getUniqueID "gke-${TEST_TYPE}-${BACKEND}")

  ${ROOT}/tests/e2e/scripts/e2e-kube.sh \
  -a ${APIPROXY_SERVICE} \
  -t ${TEST_TYPE} \
  -g ${BACKEND} \
  -m ${APIPROXY_IMAGE} \
  -R ${ROLLOUT_STRATEGY} \
  -i ${UNIQUE_ID}
}

# IMAGE veriable will be set by the script
${ROOT}/scripts/robot-release.sh

build_e2e_dependency
# TODO(jilinxia): add other backend tests.
e2eGKE -c "tight" -t "http" -g "bookstore" -R "managed" -m $(get_proxy_image_name_with_sha)
e2eGKE -c "tight" -t "http2" -g "echo" -R "managed" -m $(get_proxy_image_name_with_sha)