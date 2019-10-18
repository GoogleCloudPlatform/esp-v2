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

# Fail on any error.
set -eo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
. ${ROOT}/scripts/all-utilities.sh || { echo 'Cannot load Bash utilities'; exit 1; }

while getopts :i: arg; do
  case ${arg} in
    i) IMAGE="${OPTARG}" ;;
    *) error_exit "Unrecognized argument -${OPTARG}" ;;
  esac
done

ENVOY_IMAGE_SHA_NAME=$(get_envoy_image_name_with_sha)
PROXY_IMAGE_SHA_NAME=$(get_proxy_image_name_with_sha)
SERVERLESS_IMAGE_SHA_NAME=$(get_serverless_image_name_with_sha)

SUBSTITUTIONS_ARG="_ENVOY_IMAGE_SHA_NAME=${ENVOY_IMAGE_SHA_NAME},\
_PROXY_IMAGE_SHA_NAME=${PROXY_IMAGE_SHA_NAME},\
_SERVERLESS_IMAGE_SHA_NAME=${SERVERLESS_IMAGE_SHA_NAME}"

DOCKERFILE_PROXY="Dockerfile-proxy"
DOCKERFILE_SERVERLESS="Dockerfile-serverless"
DOCKERFILE_PATH="${ROOT}/docker"

set -x
sed "s|_ENVOY_IMAGE_SHA_NAME|${ENVOY_IMAGE_SHA_NAME}|g" \
  "${DOCKERFILE_PATH}/${DOCKERFILE_PROXY}.tmpl" > "${DOCKERFILE_PATH}/${DOCKERFILE_PROXY}"
sed "s|_PROXY_IMAGE_SHA_NAME|${PROXY_IMAGE_SHA_NAME}|g" \
  "${DOCKERFILE_PATH}/${DOCKERFILE_SERVERLESS}.tmpl" > "${DOCKERFILE_PATH}/${DOCKERFILE_SERVERLESS}"

gcloud builds submit  ${ROOT} --config ${DOCKERFILE_PATH}/cloudbuild.yaml \
  --substitutions ${SUBSTITUTIONS_ARG}\
  --project cloudesf-testing || { exit 1;}
