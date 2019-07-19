#!/bin/bash

# Copyright 2018 Google Cloud Platform Proxy Authors

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
. ${ROOT}/scripts/all-utilities.sh || { echo 'Cannot load Bash utilities'; exit 1; }

IMAGE=$(get_image_name_with_sha)
while getopts :i: arg; do
    case ${arg} in
        i) IMAGE="${OPTARG}";;
    *) echo "Invalid argument -${OPTARG}, ignoring.";;
    esac
done

echo "Checking if docker image ${IMAGE} exists.."
gcloud docker -- pull "${IMAGE}" \
  && { echo "Image ${IMAGE} already exists; skipping"; exit 0; }

echo "Building Envoy"
${BAZEL} version

# Build binaries
if [ ! -d "${ROOT}/bin" ]; then
  mkdir ${ROOT}/bin
fi

make -C ${ROOT} depend.apiproto
make -C ${ROOT} build
make -C ${ROOT} build-envoy

# Build docker container image for GKE/GCE deployment.
${ROOT}/scripts/linux-build-docker.sh -i "${IMAGE}" \
    || error_exit 'Failed to build a generic Docker Image.'
