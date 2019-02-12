#!/bin/bash

# Copyright 2018 Google LLC

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

SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_PATH}/../../.." && pwd)"
YAML_FILE=${SCRIPT_PATH}/../testdata/grpc-bookstore.yaml
APP="api-proxy-grpc-bookstore"

. ${SCRIPT_PATH}/prow-utilities.sh || { echo "Cannot load Bash utilities" ; exit 1 ; }
e2e_options "${@}"

TEST_ID="gke-${COUPLING_OPTION}-${TEST_TYPE}-${BACKEND}"
LOG_DIR="$(mktemp -d /tmp/log.XXXX)"

# Testing protocol
# TODO(jilinxia): parse arguments
# TODO(jilinxia): use APIPROXY image, instead of ESP image.
run kubectl create -f ${YAML_FILE}

HOST=$(get_cluster_host)

# Running Test
run_nonfatal long_running_test \
  "${HOST}" \
  "${DURATION_IN_HOUR}" \
  "${API_KEY}" \
  "${APIPROXY_SERVICE}" \
  "${LOG_DIR}" \
  "${TEST_ID}" \
  "${UNIQUE_ID}"