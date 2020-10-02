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

# Presubmit script triggered by Prow.

# Fail on any error.
set -eo pipefail

WD=$(dirname "$0")
WD=$(cd "$WD";
pwd)
ROOT=$(dirname "$WD")
export PATH=$PATH:$GOPATH/bin


gcloud config set core/project cloudesf-testing
gcloud auth activate-service-account \
  --key-file="${GOOGLE_APPLICATION_CREDENTIALS}"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"
. ${ROOT}/scripts/all-utilities.sh || { echo 'Cannot load Bash utilities';
exit 1; }
. ${ROOT}/tests/e2e/scripts/prow-utilities.sh || { echo 'Cannot load Bash utilities';
exit 1; }

# This integration test is used to ensure the backward-compatibility between ESPv2
# and API Gateway use case. Once this test is broken, it reminds us to increment
# api version if necessary.
#
# The test will use the latest configmanager(configgen library) with the MASTER envoy to
# run all the MASTER integrations.

# Get the SHA of the head of master
SHA=$(git ls-remote https://github.com/GoogleCloudPlatform/esp-v2.git HEAD | cut -f 1)

# keep the current version
VERSION=$(cat ${ROOT}/VERSION)

# Checkout to the head of master
git checkout ${SHA}

# keep the current version
echo ${VERSION} > ${ROOT}/VERSION

make depend.install
make build build-envoy build-grpc-interop build-grpc-echo

echo '======================================================='
echo '============ Download latest configmanager ============'
echo '======================================================='
wait_apiproxy_image
download_configmanager_binary
chmod +x ${ROOT}/bin/configmanager
chmod +x ${ROOT}/bin/bootstrap

make integration-test-run-sequential
