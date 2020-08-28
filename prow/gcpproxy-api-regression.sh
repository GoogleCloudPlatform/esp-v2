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

git remotve -v
git branch
wait_apiproxy_image

echo '======================================================='
echo '================ Download latest envoy ================'
echo '======================================================='
download_envoy_binary
chmod +x ${ROOT}/bin/envoy

# keep the current version
VERSION=$(cat ${ROOT}/VERSION)

# checkout to the head of master
git checkout master

# keep the current version
echo ${VERSION} > ${ROOT}/VERSION

# Since the Makefile is on the submitted commit, the following cmds can only be
# replaced with `make integration-test-without-envoy-build` after this pr submitted
make depend.update
make build
make build-grpc-interop
make build-grpc-echo
make integration-test-run-sequential
