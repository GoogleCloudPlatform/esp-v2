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
set -e
set -u

echo '======================================================='
echo '=====================   e2e test  ====================='
echo '======================================================='

PROJECT_ID="api_proxy_e2e_test"
UNIQUE_ID=test

WD=$(dirname "$0")
WD=$(cd "$WD"; pwd)
ROOT=$(dirname "$WD")

cd "${ROOT}"

if [ ! -d "$GOPATH/bin" ]; then
  mkdir $GOPATH/bin
fi
if [ ! -d "bin" ]; then
  mkdir bin
fi

# libraries for go build
curl https://glide.sh/get | sh
glide install

# depedencies for envoy build
apt-get update && \
    apt-get -y install libtool cmake automake ninja-build curl unzip

function getApiProxyService() {
  if [[ "${1}" == "bookstore" ]]; then
    echo "bookstore.endpoints.cloudesf-testing.cloud.goog"
    return 0
  else
    echo "Service ${1} is not supported."
    return 1
  fi
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
  ${ROOT}/tests/e2e/scripts/e2e-kube.sh \
  -a ${APIPROXY_SERVICE} \
  -c ${COUPLING_OPTION} \
  -t ${TEST_TYPE} \
  -g ${BACKEND} \
  -m ${APIPROXY_IMAGE} \
  -R ${ROLLOUT_STRATEGY} \
  -i ${UNIQUE_ID}
}

function apiProxyGenericDockerImage() {
  # Generic docker image format. https://git-scm.com/docs/git-show.
  local image_format='gcr.io/cloudesf-testing/api-proxy:git-%H'
  local image="$(git show -q HEAD \
    --pretty=format:"${image_format}")"
  echo -n $image
  return 0
}

GENERIC_IMAGE=$(apiProxyGenericDockerImage)

function buildPackages() {
  ${ROOT}/scripts/robot-release.sh -i ${GENERIC_IMAGE}
}

buildPackages

# TODO(jilinxia): add other backend tests.
e2eGKE -c "tight" -t "http" -g "bookstore" -R "managed" -m ${GENERIC_IMAGE}
