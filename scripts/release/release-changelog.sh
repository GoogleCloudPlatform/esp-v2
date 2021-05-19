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

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
. ${ROOT}/scripts/all-utilities.sh || { echo "Cannot load Bash utilities"; exit 1; }

# This script will generate changelog from last commit to {SHA}
# You can use either tag or SHA to specify last commmit.

function usage() {
  [[ -n "${1}" ]] && echo "${1}"
  cat <<EOF
usage: ${BASH_SOURCE[0]} -s <commit sha> -l <last release sha or tag> -d <directory> [-n <current version number>]

Generates changelog for changes between last release and current release.
EOF
  exit 1
}

function push_tool() {
  echo "Push directory "
  pushd ${1}
}

function pop_tool() {
  popd
}


SHA=""
LAST_COMMIT=""
DIRECTORY="."

while getopts :s:l:d:n: arg; do
  case ${arg} in
    s) SHA="${OPTARG}";;
    l) LAST_COMMIT="${OPTARG}";;
    d) DIRECTORY="${OPTARG}";;
    n) VERSION="${OPTARG}";;
    *) usage "Invalid option: -${OPTARG}";;
  esac
done

[[ -n "${SHA}" ]] || usage "Must provide commit sha via '-s' parameter."
[[ "${SHA}" =~ ^[0-9a-f]{40}$ ]] || usage "Invalid SHA: ${SHA}."
[[ -n "${LAST_COMMIT}" ]] || usage "Must provide last commit sha or tag via '-l' parameter."

if [ "${DIRECTORY}" != "." ]; then
  push_tool ${DIRECTORY}
fi

if [ "${VERSION}" = "" ]; then
  VERSION="$(command cat ${ROOT}/VERSION)" \
    || usage "Cannot determine release version (${ROOT}/VERSION)."
fi

echo $(pwd)

echo "The change logs from last commit: ${LAST_COMMIT}"
echo "To SHA: ${SHA}"
echo "In directory: ${DIRECTORY}"
echo "Please copy this result to the release bug."

CHANGELOG="$(mktemp)"
trap "rm '${CHANGELOG}'" EXIT

cat <<EOF > "${CHANGELOG}"
# Release ${VERSION} $(date +%d-%m-%Y)

TODO: Edit the section below before submitting! DIRECTORY ${PWD}
===============================================

EOF

# List all commits since the last release and perform basic processing
# for release notes.
#
# git log --pretty format outputs an asterisk, commit subject (%s),
#   newline (%n) and body (%b) indented by 2 spaces and re-wrapped to 76
#   columns which is a git log default (%w(76,2)).
#
# The perl command strips the Change-Id line and surrounding whitespace.
#   -p: assumes "while (<>) { ... }" loop around program and prints
#       the line also, like sed.
#   -e: takes a single line of Perl program
#
# BEGIN {undef $/;} enables 'slurp' mode on input. Perl special variable
#   $/ means input record separator (newline by default) so we reset it
#   to undef so Perl reads all input in order to process multi-line regex
#   matching (http://perldoc.perl.org/perlvar.html).
#
# `s/\s+Change-Id:[^\n]*\s+/\n/gs` replaces Change-Id lines surrounded by
#   whitespace with a single line so they don't need to be manually removed
#   from the generated changelog.

echo $(pwd)
git log ${LAST_COMMIT}..${SHA} --pretty="- %s%w(76,2)" \
  | perl -pe'BEGIN {undef $/;} s/\s+Change-Id:[^\n]*\s+/\n/gs;' \
    >> "${CHANGELOG}"

cat <<EOF >> "${CHANGELOG}"

TODO: Edit the section above before submitting! DIRECTORY ${PWD}
===============================================
EOF

if [[ -f "${ROOT}/CHANGELOG.md" ]]; then
  cat "${ROOT}/CHANGELOG.md" >> "${CHANGELOG}"
fi

cp "${CHANGELOG}" "${ROOT}/CHANGELOG.md"

git log ${LAST_COMMIT}..${SHA} --pretty=oneline

printf '\e[31m
***************************************************************************
*      Please paste the script output verbatim into the release bug.      *
*                                                                         *
*       Edit /CHANGELOG.md to summarize the changes in the release,       *
*            send for review and submit to the release branch.            *
***************************************************************************
\e[0m'

if [ "${DIRECTORY}" != "." ]; then
  pop_tool
fi