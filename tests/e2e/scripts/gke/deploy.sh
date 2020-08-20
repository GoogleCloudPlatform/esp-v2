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
ARGS=""
SCHEME="http"
LISTENER_PORT="80"

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
ARGS="$ARGS \"--tracing_sample_rate=0.01\","
ARGS="$ARGS \"--tracing_outgoing_context=x-cloud-trace-context\","
ARGS="$ARGS \"--admin_port=8001\""
case "${BACKEND}" in
  'bookstore')
    YAML_TEMPLATE=${ROOT}/tests/e2e/testdata/bookstore/gke/http-bookstore.yaml.template
    YAML_FILE=${ROOT}/tests/e2e/testdata/bookstore/gke/http-bookstore.yaml
    ARGS="$ARGS , \"--backend=http://127.0.0.1:8081\"" ;;
  'echo')
    YAML_TEMPLATE=${ROOT}/tests/e2e/testdata/grpc_echo/gke/grpc-echo.yaml.template
    YAML_FILE=${ROOT}/tests/e2e/testdata/grpc_echo/gke/grpc-echo.yaml
    ARGS="$ARGS , \"--backend=grpc://127.0.0.1:8081\"" ;;
  'interop')
    YAML_TEMPLATE=${ROOT}/tests/e2e/testdata/grpc_interop/gke/grpc-interop.yaml.template
    YAML_FILE=${ROOT}/tests/e2e/testdata/grpc_interop/gke/grpc-interop.yaml
    ARGS="$ARGS , \"--backend=grpc://127.0.0.1:8081\"" ;;
  *)
    echo "Invalid backend ${BACKEND}"
    return 1 ;;
esac

if [ ${BACKEND} == "bookstore" ]; then
  # These file mount paths are set in tests/e2e/testdata/bookstore/gke/http-bookstore.yaml

  # Support service account credentials for non-gcp deployment.
  SA_CRED_PATH="$(mktemp  /tmp/servie_account_cred.XXXX)"
  [[ -n ${USING_SA_CRED} ]] && ARGS="$ARGS, \"--service_account_key=/etc/creds/$(basename "${SA_CRED_PATH}")\""

  # Support TLS termination in ESPv2.
  ARGS="$ARGS, \"--ssl_server_cert_path=/etc/esp/ssl\""
fi

sed "s|APIPROXY_IMAGE|${APIPROXY_IMAGE}|g" ${YAML_TEMPLATE}  \
  | sed "s|ARGS|${ARGS}|g" | tee ${YAML_FILE}

# Push service config to service management. Only need to run when there
# is changes in the service config, and also remember to update the version
# number in kubernetes config.
#
case "${BACKEND}" in
  'bookstore')
    SERVICE_IDL_TMPL="${ROOT}/tests/endpoints/bookstore/bookstore_swagger_template.json"
    SERVICE_IDL="${ROOT}/tests/endpoints/bookstore/bookstore_swagger.json"

    cat "${SERVICE_IDL_TMPL}" \
        | jq ".host = \"${APIPROXY_SERVICE}\" \
        | .securityDefinitions.auth0_jwk.\"x-google-audiences\" = \"${APIPROXY_SERVICE}\"" \
        > "${SERVICE_IDL}"

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
    exit 1;;
esac



LOG_DIR="$(mktemp -d /tmp/log.XXXX)"

create_service ${CREATE_SERVICE_ARGS}

# Creates service on GKE cluster.
NAMESPACE="${UNIQUE_ID}"
run kubectl create namespace "${NAMESPACE}" || error_exit "Namespace already exists"

if [ "${BACKEND}" == 'bookstore' ]; then
  # Service account key secret.
  get_test_client_key "e2e-non-gcp-instance-proxy-rt-sa.json" "${SA_CRED_PATH}"
  run kubectl create secret generic service-account-cred --from-file="${SA_CRED_PATH}" --namespace "${NAMESPACE}"

  # Generate untrusted self-signed cert.
  # Common name doesn't matter, client will not verify it.
  openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
      -keyout ./server.key -out ./server.crt \
      -subj "/C=US/ST=CA/O=Google/CN=fake-fqdn.cloud.google.com"

  # SSL cert secret.
  run kubectl create secret generic esp-ssl --from-file=./server.crt --from-file=./server.key --namespace "${NAMESPACE}"

  SCHEME="https"
  LISTENER_PORT="443"
fi

run kubectl create -f ${YAML_FILE} --namespace "${NAMESPACE}"
HOST=$(get_cluster_host "${NAMESPACE}")

# Run in background while e2e tests are running.
# ESPv2 is deployed in managed mode for all these e2e tests.
# This will cause ESPv2 to rebuild the Envoy listener while lots of traffic is running through.
function doServiceRollout() {
  while true; do
    echo 'doServiceRollout: Sleeping until next service rollout'
    sleep 15m
    echo "doServiceRollout: Deploying and rolling out new config for service ${APIPROXY_SERVICE}"
    create_service ${CREATE_SERVICE_ARGS}
  done
}

# Start background process, only supported for bookstore backend.
if [ "${BACKEND}" == 'bookstore' ]; then
  doServiceRollout &
fi

# Running Test
STATUS=0
run_nonfatal long_running_test  \
  "${HOST}"  \
  "${SCHEME}" \
  "${LISTENER_PORT}" \
  "${DURATION_IN_HOUR}"  \
  "${API_KEY}"  \
  "${APIPROXY_SERVICE}"  \
  "${LOG_DIR}"  \
  "${TEST_ID}"  \
  "${UNIQUE_ID}" \
  "gke" \
  "" \
  || STATUS=${?}

# Kill background process.
if [ "${BACKEND}" == 'bookstore' ]; then
  kill $(jobs -p)
fi

if [[ -n ${REMOTE_LOG_DIR} ]]; then
  fetch_proxy_logs "${NAMESPACE}" "${LOG_DIR}"
  upload_logs "${REMOTE_LOG_DIR}" "${LOG_DIR}"
  rm -rf "${LOG_DIR}"
fi

exit ${STATUS}
