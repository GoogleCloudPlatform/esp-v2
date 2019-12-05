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

SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_PATH}/../../../.." && pwd)"
ARGS="\
  \"--backend=127.0.0.1:8081\",\
"

. ${ROOT}/tests/e2e/scripts/prow-utilities.sh || { echo "Cannot load Bash utilities";
exit 1; }
. ${ROOT}/tests/e2e/scripts/gke/utilities.sh || { echo "Cannot load GKE utilities";
exit 1; }
. ${ROOT}/tests/e2e/scripts/linux-install-wrk.sh || { echo "Cannot load WRK utilities";
exit 1; }

e2e_options "${@}"

echo "Installing tools if necessary"
install_e2e_dependencies
update_wrk

TEST_ID="gke-${TEST_TYPE}-${BACKEND}"
LOG_DIR="$(mktemp -d /tmp/log.XXXX)"
PROJECT_ID="cloudesf-testing"

# Parses parameters into config file.
ARGS="$ARGS \"--service=${APIPROXY_SERVICE}\","
ARGS="$ARGS \"--rollout_strategy=${ROLLOUT_STRATEGY}\","
ARGS="$ARGS \"--tracing_sample_rate=0.00001\","
ARGS="$ARGS \"--enable_admin\""
case "${BACKEND}" in
  'bookstore')
    YAML_TEMPLATE=${ROOT}/tests/e2e/testdata/bookstore/gke/http-bookstore.yaml.template
    YAML_FILE=${ROOT}/tests/e2e/testdata/bookstore/gke/http-bookstore.yaml
    ARGS="$ARGS , \"--backend_protocol=http1\"" ;;
  'echo')
    YAML_TEMPLATE=${ROOT}/tests/e2e/testdata/grpc_echo/gke/grpc-echo.yaml.template
    YAML_FILE=${ROOT}/tests/e2e/testdata/grpc_echo/gke/grpc-echo.yaml
    ARGS="$ARGS , \"--backend_protocol=grpc\"" ;;
  'interop')
    YAML_TEMPLATE=${ROOT}/tests/e2e/testdata/grpc_interop/gke/grpc-interop.yaml.template
    YAML_FILE=${ROOT}/tests/e2e/testdata/grpc_interop/gke/grpc-interop.yaml
    ARGS="$ARGS , \"--backend_protocol=grpc\"" ;;
  *)
    echo "Invalid backend ${BACKEND}"
    return 1 ;;

esac

sed "s|APIPROXY_IMAGE|${APIPROXY_IMAGE}|g" ${YAML_TEMPLATE}  \
  | sed "s|ARGS|${ARGS}|g" | tee ${YAML_FILE}

# Push service config to service management. Only need to run when there
# is changes in the service config, and also remember to update the version
# number in kubernetes config.
#
case "${BACKEND}" in
  'bookstore')
    SERVICE_IDL="${ROOT}/tests/endpoints/bookstore/bookstore_swagger_template.json"
    CREATE_SERVICE_ARGS="${SERVICE_IDL}"
    ;;
  'echo')
    SERVICE_YAML="${ROOT}/tests/endpoints/grpc_echo/grpc-test.yaml"
    SERVICE_DSCP="${ROOT}/tests/endpoints/grpc_echo/proto/api_descriptor.pb"
    CREATE_SERVICE_ARGS="${SERVICE_YAML} ${SERVICE_DSCP}"
    ARGS="$ARGS -g" ;;
  'interop')
    SERVICE_YAML="${ROOT}/tests/endpoints/grpc_interop/grpc-interop.yaml"
    SERVICE_DSCP="${ROOT}/tests/endpoints/grpc_interop/proto/api_descriptor.pb"
    CREATE_SERVICE_ARGS="${SERVICE_YAML} ${SERVICE_DSCP}"
    ARGS="$ARGS -g" ;;
  *)
    echo "Invalid backend ${BACKEND}"
    return 1 ;;
esac

LOG_DIR="$(mktemp -d /tmp/log.XXXX)"

create_service ${CREATE_SERVICE_ARGS}

# Creates service on GKE cluster.
NAMESPACE="${UNIQUE_ID}"
run kubectl create namespace "${NAMESPACE}" || error_exit "Namespace already exists"
run kubectl create -f ${YAML_FILE} --namespace "${NAMESPACE}"
HOST=$(get_cluster_host "${NAMESPACE}")

# Running Test
STATUS=0
run_nonfatal long_running_test  \
  "${HOST}"  \
  "http" \
  "80" \
  "${DURATION_IN_HOUR}"  \
  "${API_KEY}"  \
  "${APIPROXY_SERVICE}"  \
  "${LOG_DIR}"  \
  "${TEST_ID}"  \
  "${UNIQUE_ID}" \
  "gke" \
  || STATUS=${?}

# Deploy new config and check new rollout on /endpoints_status
if [[ ( "${ROLLOUT_STRATEGY}" == "managed" ) && ( "${BACKEND}" == "bookstore" ) ]]; then
  # Deploy new service config
  create_service "${SERVICE_IDL}"
fi

if [[ -n ${REMOTE_LOG_DIR} ]]; then
  fetch_proxy_logs "${NAMESPACE}" "${LOG_DIR}"
  upload_logs "${REMOTE_LOG_DIR}" "${LOG_DIR}"
  rm -rf "${LOG_DIR}"
fi

exit ${STATUS}
