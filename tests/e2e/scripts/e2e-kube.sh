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
ARGS="\
  \"--backend=127.0.0.1:8081\",\
"

. ${SCRIPT_PATH}/prow-utilities.sh || { echo "Cannot load Bash utilities" ; exit 1 ; }
e2e_options "${@}"

. ${SCRIPT_PATH}/linux-install-wrk.sh || { echo "Cannot load Bash utilities" ; exit 1 ; }
echo "Installing tools if necessary"
update_wrk

TEST_ID="gke-${TEST_TYPE}-${BACKEND}"
LOG_DIR="$(mktemp -d /tmp/log.XXXX)"
PROJECT_ID="cloudesf-testing"

# Parses parameters into config file.
ARGS="$ARGS \"--service=${APIPROXY_SERVICE}\","
ARGS="$ARGS \"--rollout_strategy=${ROLLOUT_STRATEGY}\","
ARGS="$ARGS \"--enable_tracing\", \"--tracing_sample_rate=0.005\""
 case "${BACKEND}" in
   'bookstore' )
      YAML_TEMPLATE=${SCRIPT_PATH}/../testdata/bookstore/http-bookstore.yaml.template
      YAML_FILE=${SCRIPT_PATH}/../testdata/bookstore/http-bookstore.yaml
      ARGS="$ARGS , \"--backend_protocol=http1\"";;
   'echo'      )
      YAML_TEMPLATE=${SCRIPT_PATH}/../testdata/grpc-echo/grpc-echo.yaml.template
      YAML_FILE=${SCRIPT_PATH}/../testdata/grpc-echo/grpc-echo.yaml
      ARGS="$ARGS , \"--backend_protocol=grpc\"";;
   'interop'      )
      YAML_TEMPLATE=${SCRIPT_PATH}/../testdata/grpc-interop/grpc-interop.yaml.template
      YAML_FILE=${SCRIPT_PATH}/../testdata/grpc-interop/grpc-interop.yaml
      ARGS="$ARGS , \"--backend_protocol=grpc\"";;
     *         )
    echo "Invalid backend ${BACKEND}"
    return 1;;

 esac

sed "s|APIPROXY_IMAGE|${APIPROXY_IMAGE}|g"  ${YAML_TEMPLATE} \
  | sed "s|ARGS|${ARGS}|g" | tee ${YAML_FILE}

# Push service config to service management servie. Only need to run when there
# is changes in the service config, and also remember to update the version
# number in kubernetes config.
#
case "${BACKEND}" in
   'bookstore' )
      SERVICE_IDL="${SCRIPT_PATH}/../testdata/bookstore/bookstore_swagger_template.json"
      CREATE_SERVICE_ARGS="${SERVICE_IDL}"
      ;;
   'echo'      )
      SERVICE_YAML="${ROOT}/tests/endpoints/grpc-echo/grpc-test.yaml"
      SERVICE_DSCP="${ROOT}/tests/endpoints/grpc-echo/proto/api_descriptor.pb"
      CREATE_SERVICE_ARGS="${SERVICE_YAML} ${SERVICE_DSCP}"
      ARGS="$ARGS -g";;
   'interop'      )
      SERVICE_YAML="${ROOT}/tests/endpoints/grpc-interop/grpc-interop.yaml"
      SERVICE_DSCP="${ROOT}/tests/endpoints/grpc-interop/proto/api_descriptor.pb"
      CREATE_SERVICE_ARGS="${SERVICE_YAML} ${SERVICE_DSCP}"
      ARGS="$ARGS -g";;
   *          )
    echo "Invalid backend ${BACKEND}"
    return 1;;
esac


create_service ${CREATE_SERVICE_ARGS}

# Creates service on GKE cluster.
NAMESPACE="${UNIQUE_ID}"
run kubectl create namespace "${NAMESPACE}" || error_exit "Namespace already exists"
run kubectl create -f ${YAML_FILE} --namespace "${NAMESPACE}"
HOST=$(get_cluster_host "${NAMESPACE}")

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
