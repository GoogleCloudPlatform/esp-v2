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

if [[ "$(uname)" != "Linux" ]]; then
  echo "Run on Linux only."
  exit 1
fi

SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_PATH}/../../.." && pwd)"
. ${ROOT}/scripts/all-utilities.sh || { echo "Cannot load Bash utilities" ; exit 1 ; }

TOOLS_DIR='/tmp/apiproxy-tools'

# 2019.04.11
WRK_VERSION='7594a95186ebdfa7cb35477a8a811f84e2a31b62'
WRK_DIRECTORY="${TOOLS_DIR}/wrk"

function clone_wrk() {
  local dir="${1}"
  echo "Cloning wrk."
  git clone https://github.com/wg/wrk.git "${dir}" \
    || error_exit "Cannot clone wrk repository."
}

function build_wrk() {
  local dir="${1}"

  echo 'Building wrk'
  pushd "$dir"
  git clean -dffx \
  && git fetch origin \
  && git fetch origin --tags \
  && git reset --hard ${WRK_VERSION} \
  && make WITH_OPENSSL=/usr \
  && ${SUDO} cp ./wrk /usr/local/bin/wrk \
  && ${SUDO} chmod a+rx /usr/local/bin/wrk \
  || error_exit "wrk build failed."

  # TODO(jilinxia): cache wrk into GCS bucket.
  # update_tool wrk "${WRK_VERSION}" ./wrk
  set_wrk
  echo $WRK
  popd
}

function update_wrk() {
  local wrk_current='none'
  if [[ -d "${WRK_DIRECTORY}" ]]; then
    wrk_current="$(git -C "${WRK_DIRECTORY}" log -n 1 --pretty=format:%H)"
  fi

  if [[ "${WRK_VERSION}" != "${wrk_current}" ]]; then
    local build_needed=true
    local wrk_tmp="$(mktemp /tmp/XXXXX.wrk.bin)"
    get_tool wrk "${WRK_VERSION}" "${wrk_tmp}" \
    && ${SUDO} cp "${wrk_tmp}" /usr/local/bin/wrk \
    && ${SUDO} chmod a+rx /usr/local/bin/wrk \
    && build_needed=false
    if [[ ${build_needed} == true ]]; then
      clone_wrk "${WRK_DIRECTORY}"
      build_wrk ${WRK_DIRECTORY}
    fi
  fi
  echo 'wrk up-to-date.'
}