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



# This script will:
#   * download a Docker image with the given SHA, re-tag it with
#     release version and publish it in Cloud Container Registry.
#


ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
. ${ROOT}/scripts/all-utilities.sh || { echo "Cannot load Bash utilities"; exit 1; }


function usage() {
  [[ -n "${1}" ]] && echo "${1}"
  cat <<EOF
usage: ${BASH_SOURCE[0]} -s <commit sha> [options]"

options are:
  -g <path_to_gcloud>
  -u <path_to_gsutil>
EOF
  exit 1
}

GSUTIL="$(which gsutil)" || GSUTIL=~/google-cloud-sdk/bin/gsutil
GCLOUD="$(which gcloud)" || GCLOUD=~/google-cloud-sdk/bin/gcloud
SHA=""

while getopts :g:u:s: arg; do
  case ${arg} in
    g) GCLOUD="${OPTARG}";;
    u) GSUTIL="${OPTARG}";;
    s) SHA="${OPTARG}";;
    *) usage "Invalid option: -${OPTARG}";;
  esac
done

[[ -n "${SHA}" ]] \
  || usage "Must provide commit sha via '-s' parameter."
[[ "${SHA}" =~ ^[0-9a-f]{40}$ ]] \
  || usage "Invalid SHA: ${SHA}."
[[ -x "${GCLOUD}" ]] \
  || usage "Cannot find gcloud (${GCLOUD}), provide it via '-g' flag."
[[ -x "${GSUTIL}" ]] \
  || usage "Cannot find gsutil (${GSUTIL}), provide it via '-u' flag."


set -x

VERSION="$(command cat ${ROOT}/VERSION)" \
  || error_exit "Cannot find release version (${ROOT}/VERSION)"
CURRENT_BRANCH="$(git rev-parse --abbrev-ref HEAD)"
if RELEASE_BRANCH_SHA="$(git rev-parse upstream/${CURRENT_BRANCH})"; then
  if [[ "${SHA}" != "${RELEASE_BRANCH_SHA}" ]]; then
    printf "\e[31m
WARNING: Release branch commit (${RELEASE_BRANCH_SHA}) doesn't match ${SHA}.
\e[0m"
  fi
else
  printf "\e[31m
WARNING: Cannot find release branch origin/release-${VERSION}.
\e[0m"
fi

function push_docker_image() {
  local source_image="${1}"
  local target_image="${2}"

  docker_tag "${source_image}" "${target_image}" \
    || { echo "Could not tag ${source_image} to ${target_image}."; return 1; }

  echo "Pushing ${target_image} to Cloud Container Registry."
  "${GCLOUD}" docker -- push "${target_image}" \
    || { echo "Cloud Container Registry push of ${target_image} failed."; return 1; }

  return 0;
}

push_docker_image \
  "$(get_proxy_image_name_with_sha)" \
  "$(get_proxy_image_release_name):${VERSION}" \
  || error_exit "Docker image push failed."

push_docker_image \
  "$(get_serverless_image_name_with_sha)" \
  "$(get_serverless_image_release_name):${VERSION}" \
  || error_exit "Docker image push failed."

printf '\e[31m
***************************************************************************
*      Please paste the script output verbatim into the release bug.      *
***************************************************************************
\e[0m'
