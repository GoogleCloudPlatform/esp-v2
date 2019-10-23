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

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
. ${ROOT}/scripts/all-utilities.sh || { echo "Cannot load Bash utilities"; exit 1; }

APIPROXY_RELEASE_PROJECT=apiproxy-release
DIRECT_REPO=''

function usage() {
  [[ -n "${1}" ]] && echo "${1}"
  cat << END_USAGE

Usage: ${BASH_SOURCE[0]} [-r <DIRECT_REPO>]

This script will show all release CloudESF image tags

END_USAGE
  exit 1
}


while getopts :r: arg; do
  case ${arg} in
    *) usage "Invalid option: -${OPTARG}";;
  esac
done

function list_image_tags() {
  local image=$1
  echo "show tags for ${image}"
  gcloud container images list-tags ${image} --project ${APIPROXY_RELEASE_PROJECT}
}

list_image_tags $(get_proxy_image_release_name)
list_image_tags $(get_serverless_image_release_name)


printf '\e[31m
***************************************************************************
*      Please paste the script output verbatim into the release bug.      *
***************************************************************************
\e[0m'