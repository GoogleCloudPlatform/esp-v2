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

set -o errexit
set -o nounset

SERVICE_NAME=$1
CONFIG_ID=$2

DIR="$(cd "$(dirname "$0")" || exit ; pwd -P)"
WD="$(dirname "${DIR}")"

cd "$WD"

echo "Install dependents............"
make tools
make depend.install

echo "Build envoy............"
bazel build src/envoy:envoy
sudo cp bazel-bin/src/envoy/envoy vendor

echo "Build Config Manger............"
CGO_ENABLED=0 GOOS=linux go build -o vendor/configmanager src/go/server/server.go

sudo docker build -f docker/Dockerfile-proxy -t gcpproxy .

sudo docker run --rm -it -p 8080:8080 gcpproxy "${SERVICE_NAME}" "${CONFIG_ID}"

