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

for filename in $ROOT/examples/**/*.json; do
    echo "Formatting $filename"
    TEMP_FILE=$(mktemp)

    # jq is a common bash utility used to format/sort/filter json.
    # Sort keys (-S) for all fields (.) in the input file and output to the temp file.
    jq -S '.' "$filename" > "$TEMP_FILE"
    cp -f "$TEMP_FILE" "$filename"
    rm "$TEMP_FILE"
done
