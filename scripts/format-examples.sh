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

# Recurse on globs
shopt -s globstar

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
#shellcheck source=./scripts/all-utilities.sh
. "${ROOT}/scripts/all-utilities.sh" || { echo 'Cannot load Bash utilities' && exit 1; }

for filename in $ROOT/examples/**/*.json; do
    echo "Formatting $filename"

    TEMP_FILE=$(mktemp)
    jq -S '.' "$filename" > "$TEMP_FILE"
    cp -f "$TEMP_FILE" "$filename"
    rm "$TEMP_FILE"
done
