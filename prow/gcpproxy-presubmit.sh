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

cd "${ROOT}"
. ${ROOT}/scripts/all-utilities.sh || { echo 'Cannot load Bash utilities';
exit 1; }

echo '======================================================='
echo '===================== Setup Cache ====================='
echo '======================================================='
try_setup_bazel_remote_cache "${PROW_JOB_ID}" "${IMAGE}" "${ROOT}" "${PRESUBMIT_TEST_CASE}"
gcloud auth configure-docker

echo '======================================================='
echo '===================== Spelling Check ====================='
echo '======================================================='
make spelling.check

# golang test
echo '======================================================'
echo '=====================   Go test  ====================='
echo '======================================================'
if [ ! -d "$GOPATH/bin" ]; then
  mkdir $GOPATH/bin
fi
if [ ! -d "bin" ]; then
  mkdir bin
fi
export GO111MODULE=on
make tools
make depend.install
#make test

# c++ test
echo '======================================================'
echo '===================== Bazel test ====================='
echo '======================================================'
if [ -z ${PRESUBMIT_TEST_CASE} ];
then
  echo "running normal presubmit test"
else
  echo "running ${PRESUBMIT_TEST_CASE} presubmit test"
fi

case "${PRESUBMIT_TEST_CASE}" in
  "asan")
    make test-envoy-asan
    ;;
  "msan")
    make test-envoy-msan
    ;;
  "tsan")
    make test-envoy-tsan
    ;;
  *)
    make test-envoy
    ;;
esac

echo '======================================================'
echo '===================== Integration test  =============='
echo '======================================================'
make depend.install.endpoints
case "${PRESUBMIT_TEST_CASE}" in
  "asan")
    make integration-test-asan
    ;;
  "msan")
    make integration-test-tsan
    ;;
  "tsan")
    make integration-test-tsan
    ;;
  *)
    make integration-test
    ;;
esac

make clean
