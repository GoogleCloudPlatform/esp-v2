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

# run this script at esp-v2 root folder
VERSION_FILE="api/VERSION"

CURR_VERSION=$(cat $VERSION_FILE)
VERSION_NUM=${CURR_VERSION#v}
NEXT_VERSION="v$((VERSION_NUM + 1))"

# rename version folder
git mv "api/envoy/${CURR_VERSION}" "api/envoy/${NEXT_VERSION}"

ALL_FILES=$(find ./ -type f)
sed -i -e "s|envoy/${CURR_VERSION}/http|envoy/${NEXT_VERSION}/http|g" $ALL_FILES
sed -i -e "s|envoy.${CURR_VERSION}.http|envoy.${NEXT_VERSION}.http|g" $ALL_FILES
sed -i -e "s|envoy::${CURR_VERSION}::http|envoy::${NEXT_VERSION}::http|g" $ALL_FILES
