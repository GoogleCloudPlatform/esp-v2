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
. ${ROOT}/scripts/all-utilities.sh || { echo 'Cannot load Bash utilities';
exit 1; }


function checkImageExistence() {
  local image_name=$1
  local tag=$2
  if gcloud container images list-tags ${image_name} | grep -q ${tag}; then
    return 0;
  else
    return 1;
  fi
}

# golang build
echo '======================================================='
echo '================= Build Config Manager ================='
echo '======================================================='

make tools
make depend.install
make build

# c++ build
echo '======================================================='
echo '===================== Build Envoy ====================='
echo '======================================================='

make build-envoy

echo "Checking if docker image $(get_envoy_image_name_with_sha) and image $(get_proxy_image_name_with_sha) exists.."

checkImageExistence $(get_envoy_image_name) $(get_tag_name)  \
  && checkImageExistence $(get_gcsrunner_image_name) $(get_tag_name)  \
  && checkImageExistence $(get_proxy_image_name) $(get_tag_name)  \
  && { echo "Images $(get_envoy_image_name_with_sha), $(get_gcsrunner_image_name_with_sha), and $(get_proxy_image_name_with_sha) already exists; Skip.";
exit 0; }

gcloud config list account --format "value(core.account)"
upload_envoy_binary

echo "Images $(get_envoy_image_name_with_sha), $(get_gcsrunner_image_name_with_sha), and $(get_proxy_image_name_with_sha) don't exist; Start to build."

echo '======================================================='
echo '================= Cloud Build Docker =================='
echo '======================================================='

${ROOT}/scripts/cloud-build-docker.sh  \
  || error_exit 'Failed to build a generic Docker Image.'
