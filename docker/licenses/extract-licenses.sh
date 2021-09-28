#!/usr/bin/env bash
# Copyright 2021 Google LLC All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

append_license() {
  lib=$1
  path=$2
  echo "================================================================================" >> ${TMP_LICENSES}
  echo "= ${lib} licensed under: =" >> ${TMP_LICENSES}
  echo >> ${TMP_LICENSES}
  cat "$path" >> ${TMP_LICENSES}
  echo >> ${TMP_LICENSES}
  echo "= ${path} MD5 $(cat "${path}" | md5sum | awk '{print $1}')" >> ${TMP_LICENSES}
  echo "================================================================================" >> ${TMP_LICENSES}
  echo >> ${TMP_LICENSES}

}

SRC_ROOT=$(dirname "${BASH_SOURCE}")/../..
TMP_LICENSES=${SRC_ROOT}/docker/licenses/LICENSE

cd ${SRC_ROOT}

# Please don't change this tmp folder, line 55 is hardcoded to this.
TMP_FOLDER=/tmp/ggg

echo "Collecting LICENCE files from go modules..."
go-licenses save "src/go/configmanager/main/server.go" --save_path "${TMP_FOLDER}"

# Copy LICENSE files under external
for i in $(ls bazel-esp-v2/external/ -1);
do
  FILE=bazel-esp-v2/external/$i/LICENSE
  if [ -f ${FILE} ]; then
    echo "Copy $FILE"
    mkdir -p ${TMP_FOLDER}/$i
    cp ${FILE} ${TMP_FOLDER}/$i/LICENSE
  fi
done

# Clear file
echo > ${TMP_LICENSES}

append_license "ESPv2" "${SRC_ROOT}/LICENSE"

while read -r entry; do
  # This prefix removal is hardcoded to be /tmp/ggg
  LIBRARY=${entry#\/tmp\/ggg\/}
  LIBRARY=$(expr match "$LIBRARY" '\(.*\)/LICENSE.*\?')
  append_license ${LIBRARY} ${entry}
done <<< "$(find ${TMP_FOLDER} -regextype posix-extended -iregex '.*LICENSE(\.txt)?')"


