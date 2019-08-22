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

# Build script triggered by Prow.

# Fail on any error.
set -eo pipefail

WD=$(dirname "$0")
WD=$(cd "$WD"; pwd)
ROOT=$(dirname "$WD")
export PATH=$PATH:$GOPATH/bin

cd "${ROOT}"
. ${ROOT}/scripts/all-utilities.sh || { echo 'Cannot load Bash utilities'; exit 1; }

echo '======================================================='
echo '===================== Setup Cache ====================='
echo '======================================================='
try_setup_bazel_remote_cache "${PROW_JOB_ID}" "${IMAGE}" "${ROOT}"

# generated go proto files are consistent with original proto
echo '======================================================='
echo '================== Proto consistency =================='
echo '======================================================='
make proto-consistency-test

# golang build
echo '======================================================='
echo '=====================   Go build  ====================='
echo '======================================================='
if [ ! -d "$GOPATH/bin" ]; then
  mkdir $GOPATH/bin
fi
if [ ! -d "bin" ]; then
  mkdir bin
fi
make tools
make depend.install
make build
make clean

# c++ build
echo '======================================================='
echo '===================== Bazel build ====================='
echo '======================================================='
make build-envoy
