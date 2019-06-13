#!/bin/bash

# Copyright 2019 Google Cloud Platform Proxy Authors

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

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
. ${ROOT}/scripts/all-utilities.sh || { echo 'Cannot load Bash utilities'; exit 1; }

DOCKERFILE="Dockerfile-proxy"
DOCKERFILE_PATH="${ROOT}/docker"

while getopts :i: arg; do
  case ${arg} in
    i) IMAGE="${OPTARG}";;
    *) error_exit "Unrecognized argument -${OPTARG}";;
  esac
done

[[ -n "${IMAGE}" ]] || error_exit "Specify required image argument via '-i'"

echo "Building API PROXY docker image."

retry -n 3 docker build --no-cache -t "${IMAGE}" \
  -f "${DOCKERFILE_PATH}/${DOCKERFILE}" \
  "${ROOT}/" \
  || error_exit "Docker image build failed."

echo "Pushing Docker image: ${IMAGE}"

# Try 10 times, shortest wait is 10 seconds, exponential back-off.
retry -n 10 -s 10 \
    gcloud docker -- push "${IMAGE}" \
  || error_exit "Failed to upload Docker image to gcr."
