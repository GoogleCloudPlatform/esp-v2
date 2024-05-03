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

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
. ${ROOT}/scripts/all-utilities.sh || { echo "Cannot load Bash utilities"; exit 1; }

# This script will create a tag for a release branch.

function usage() {
  [[ -n "${1}" ]] && echo "${1}"
  cat <<EOF
usage: ${BASH_SOURCE[0]} -t <tag git ref> -b <build git ref>  [-n <current version number>]"

tag git ref: commit which to tag with the release
    (typically release branch HEAD)
build git ref: commit at which the build was produced
    this is typically used when subsequent commits (such as changelog)
    were made in the release branch after the release build was produced.

example:
${BASH_SOURCE[0]} \\
    -t HEAD \\
    -b be2eb101f1b1b3e671e852656066c2909c41049b
EOF
  exit 1
}

BUILD_REF=''
TAG_REF=''

while getopts :b:t:n: arg; do
  case ${arg} in
    b) BUILD_REF="${OPTARG}" ;;
    t) TAG_REF="${OPTARG}" ;;
    n) VERSION="${OPTARG}" ;;
    *) usage "Invalid option: -${OPTARG}" ;;
  esac
done

[[ -n "${BUILD_REF}" ]] \
  || usage "Please provide the release build ref via '-b' parameter."
[[ -n "${TAG_REF}" ]] \
  || usage "Please provide the release tag ref via '-t' parameter."

BUILD_SHA=$(git rev-parse --verify "${BUILD_REF}") \
  || usage "Invalid Git reference \"${BUILD_REF}\"."
TAG_SHA=$(git rev-parse --verify "${TAG_REF}") \
  || usage "Invalid Git reference \"${TAG_REF}\"."

if [ "${VERSION}" = "" ]; then
  VERSION="$(command cat ${ROOT}/VERSION)" \
    || usage "Cannot determine release version (${ROOT}/VERSION)."
fi
# Prefix 'v' for the tag name
VERSION_TAG="v${VERSION}"

set -x

git tag --annotate --force --file=- ${VERSION_TAG} ${TAG_SHA} <<EOF
ESPv2 Release ${VERSION}

The release build was produced at ${BUILD_SHA}.
The Docker image released is:
  $(get_proxy_image_release_name):${VERSION}
EOF

# Check the version is correct.
git show -q ${VERSION_TAG}

{ set +x; } 2>/dev/null

printf "\\e[31m
You are about to push the tag ${VERSION_TAG} for ${TAG_SHA} to origin.
Once pushed, the tag cannot be removed. Are you sure? [Y/N] \\e[0m"

read yn
if [[ "${yn}" != "y" && "${yn}" != "Y" ]]; then
  echo "Aborting."
  exit 1
fi

# Push the tag to the server.
set -x
git push upstream ${VERSION_TAG}
{ set +x; } 2>/dev/null

printf '\e[31m
***************************************************************************
*      Please paste the script output verbatim into the release bug.      *
***************************************************************************
\e[0m'
