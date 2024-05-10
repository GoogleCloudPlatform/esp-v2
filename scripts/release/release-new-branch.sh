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

function usage() {
  [[ -n "${1}" ]] && echo "${1}"
  cat <<EOF
usage: ${BASH_SOURCE[0]} -s <release SHA> -n <next version number>"

This script will create a new release branch with the SHA.
Then update the version number for next release and push for review.

example:
${BASH_SOURCE[0]} \\
    -n 1.2.0 \\
    -s be2eb101f1b1b3e671e852656066c2909c41049b
EOF
  exit 1
}

SHA=''

while getopts :n:s: arg; do
  case ${arg} in
    s) SHA="${OPTARG}" ;;
    *) usage "Invalid option: -${OPTARG}" ;;
  esac
done

[[ -n "${SHA}" ]] \
  || usage "Please provide the release SHA."

# Assume the ${ROOT}/VERSION has been updated to be the version number for
# this release.
CURRENT_BRANCH="$(git rev-parse --abbrev-ref HEAD)"
VERSION="$(command cat ${ROOT}/VERSION)"
# If version name is 1.0.0, the release branch name is: v1.0.x
RELEASE_BRANCH=v${VERSION%.0}.x
echo "Current branch: ${CURRENT_BRANCH}."
echo "New release branch: ${RELEASE_BRANCH}."
[[ -z $(git diff --name-only) ]] \
  || error_exit "Current branch is not clean."

git fetch upstream \
  || error_exit "Could not fetch upstream."
git branch ${RELEASE_BRANCH} ${SHA} \
  || error_exit "Could not create a local release branch."
git push upstream ${SHA}:refs/heads/${RELEASE_BRANCH} \
  || error_exit "Failed to create a remote release branch."



MASTER_BRANCH="${VERSION}-master"
git checkout -b "${MASTER_BRANCH}" upstream/master

git checkout ${RELEASE_BRANCH}

printf '\e[31m
***************************************************************************
*      Please paste the script output verbatim into the release bug.      *
***************************************************************************
\e[0m'
