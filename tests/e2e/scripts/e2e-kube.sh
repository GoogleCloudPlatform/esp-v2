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

# Fail on any error.
set -eo pipefail

SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_PATH}/../../.." && pwd)"
YAML_FILE=${SCRIPT_PATH}/../testdata/bookstore/http-bookstore.yaml

ARGS="\
  \"--backend=http://127.0.0.1:8081\",\
"

. ${SCRIPT_PATH}/prow-utilities.sh || { echo "Cannot load Bash utilities" ; exit 1 ; }
e2e_options "${@}"

TEST_ID="gke-${COUPLING_OPTION}-${TEST_TYPE}-${BACKEND}"
LOG_DIR="$(mktemp -d /tmp/log.XXXX)"
PROJECT_ID="cloudesf-testing"

# Parses parameters into config file.
ARGS="$ARGS \"--service=${APIPROXY_SERVICE}\","
ARGS="$ARGS \"--rollout_strategy=${ROLLOUT_STRATEGY}\","
ARGS="$ARGS \"--enable_tracing\", \"--tracing_project_id=${PROJECT_ID}\", \"--tracing_sample_rate=1.0\""
run sed_i "s|APIPROXY_IMAGE|${APIPROXY_IMAGE}|g" ${YAML_FILE}
run sed_i "s|ARGS|${ARGS}|g" ${YAML_FILE}

# Push service config to service management servie. Only need to run when there
# is changes in the service config, and also remember to update the version
# number in kubernetes config.
#
SERVICE_IDL="${SCRIPT_PATH}/../testdata/bookstore/bookstore_swagger_template.json"
run sed -i "s|\${ENDPOINT_SERVICE}|${APIPROXY_SERVICE}|g" ${SERVICE_IDL}

# Creates service on GKE cluster.
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
STATUS=${?}

# Deploy new config and check new rollout on /endpoints_status
if [[ ("${ROLLOUT_STRATEGY}" == "managed") && ("${BACKEND}" == "bookstore") ]] ; then
  # Deploy new service config
  create_service "${SERVICE_IDL}"
fi

exit ${STATUS}
