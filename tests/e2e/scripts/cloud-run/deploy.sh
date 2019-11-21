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

. ${ROOT}/tests/e2e/scripts/prow-utilities.sh || { echo "Cannot load Bash utilities";
exit 1; }
. ${ROOT}/tests/e2e/scripts/cloud-run/utilities.sh || { echo "Cannot load Cloud Run utilities";
exit 1; }
. ${ROOT}/tests/e2e/scripts/linux-install-wrk.sh || { echo "Cannot load WRK utilities";
exit 1; }

e2e_options "${@}"

echo "Installing tools if necessary"
install_e2e_dependencies
update_wrk

PROJECT_ID="cloudesf-testing"
TEST_ID="cloud-run-${BACKEND}"
PROXY_RUNTIME_SERVICE_ACCOUNT="e2e-cloud-run-proxy-runtime@${PROJECT_ID}.iam.gserviceaccount.com"
BACKEND_RUNTIME_SERVICE_ACCOUNT="e2e-cloud-run-backend-runtime@${PROJECT_ID}.iam.gserviceaccount.com"
JOB_KEY_PATH="${ROOT}/tests/e2e/client/gob-prow-jobs-secret.json"
LOG_DIR="$(mktemp -d /tmp/log.XXXX)"

# Determine names of all resources
UNIQUE_ID=$(get_unique_id | cut -c 1-6)
BOOKSTORE_SERVICE_NAME=$(get_cloud_run_service_name_with_sha "${BACKEND}")
BOOKSTORE_SERVICE_NAME="${BOOKSTORE_SERVICE_NAME}-${UNIQUE_ID}"
PROXY_SERVICE_NAME=$(get_cloud_run_service_name_with_sha "api-proxy")
PROXY_SERVICE_NAME="${PROXY_SERVICE_NAME}-${UNIQUE_ID}"
ENDPOINTS_SERVICE_TITLE=$(get_cloud_run_service_name_with_sha "${BACKEND}-service")
ENDPOINTS_SERVICE_TITLE="${ENDPOINTS_SERVICE_TITLE}-${UNIQUE_ID}"
ENDPOINTS_SERVICE_NAME=""
PROXY_HOST=""

STATUS=0

function setup() {
  echo "Setup env"
  local bookstore_host=""
  local bookstore_health_code=0
  local proxy_args=""
  local endpoints_service_config_id=""

  # Cloud Run is only supported in a few regions currently
  gcloud config set run/region us-central1

  # Ensure all resources and quota is against our test project, not the CI system
  gcloud config set core/project "${PROJECT_ID}"
  gcloud config set billing/quota_project "${PROJECT_ID}"

  # Get the service account for the prow job due to b/144867112
  # TODO(b/144445217): We should let prow handle this instead of manually doing so
  get_test_client_key "gob-prow-jobs-service-account.json" "${JOB_KEY_PATH}"
  gcloud auth activate-service-account --key-file="${JOB_KEY_PATH}"

  # 1) Deploy backend service (authenticated)
  echo "Deploying backend ${BOOKSTORE_SERVICE_NAME} on Cloud Run"
  gcloud run deploy "${BOOKSTORE_SERVICE_NAME}" \
      --image="gcr.io/apiproxy-release/bookstore:1" \
      --no-allow-unauthenticated \
      --service-account "${BACKEND_RUNTIME_SERVICE_ACCOUNT}" \
      --platform managed \
      --quiet

  # 2) Get url of backend service
  bookstore_host=$(gcloud run services describe "${BOOKSTORE_SERVICE_NAME}" \
      --platform=managed \
      --format="value(status.address.url.basename())" \
      --quiet)

  # 3) Verify the backend is up using the identity of the current machine/user
  bookstore_health_code=$(curl \
      --write-out %{http_code} \
      --silent \
      --output /dev/null \
      -H "Authorization: Bearer $(gcloud auth print-identity-token)" \
      "https://${bookstore_host}"/shelves)

  if [[ "$bookstore_health_code" -ne 200 ]] ; then
    echo "Backend status is $bookstore_health_code, failing test"
    return 1
  fi
  echo "Backend deployed successfully"

  # 4) Deploy initial API Proxy service
  echo "Deploying API Proxy ${BOOKSTORE_SERVICE_NAME} on Cloud Run"

  gcloud run deploy "${PROXY_SERVICE_NAME}" \
      --image="${APIPROXY_IMAGE}" \
      --allow-unauthenticated \
      --service-account "${PROXY_RUNTIME_SERVICE_ACCOUNT}" \
      --platform managed \
      --quiet

  # 5) Get url of API Proxy service
  PROXY_HOST=$(gcloud run services describe "${PROXY_SERVICE_NAME}" \
      --platform=managed \
      --format="value(status.address.url.basename())" \
      --quiet)

  # 6) Modify the service config for Cloud Run
  local service_idl_tmpl="${ROOT}/tests/endpoints/bookstore/bookstore_swagger_template.json"
  local service_idl="${ROOT}/tests/endpoints/bookstore/bookstore_swagger.json"

  # Change the `host` to point to the proxy host (required by validation in service management)
  # Change the `title` to identify this test (for readability in cloud console)
  # Change the jwt audience to point to the proxy host (required for calling authenticated endpoints)
  # Add in the `x-google-backend` to point to the backend URL (required for backend routing)
  cat "${service_idl_tmpl}" \
      | jq ".host = \"${PROXY_HOST}\" \
      | .info.title = \"${ENDPOINTS_SERVICE_TITLE}\" \
      | .securityDefinitions.auth0_jwk.\"x-google-audiences\" = \"${PROXY_HOST}\" \
      | . + { \"x-google-backend\": { \"address\": \"https://${bookstore_host}\" } } " \
      > "${service_idl}"

  # 7) Deploy the service config
  create_service "${service_idl}"

  # 8) Get the service name and config id to enable it
  # Assumes that the names of the endpoinds service and cloud run host match
  ENDPOINTS_SERVICE_NAME="${PROXY_HOST}"
  endpoints_service_config_id=$(gcloud endpoints configs list \
      --service="${ENDPOINTS_SERVICE_NAME}" \
      --quiet \
      --limit=1 \
      --format=json \
      | jq -r '.[].id')

  # Then enable the service
  gcloud services enable "${ENDPOINTS_SERVICE_NAME}"

  # 9) Build the service config into a new image
  echo "Building serverless image"
  local build_image_script="${ROOT}/docker/serverless/gcloud_build_image"
  chmod +x "${build_image_script}"
  $build_image_script \
      -s "${ENDPOINTS_SERVICE_NAME}" \
      -c "${endpoints_service_config_id}" \
      -p "${PROJECT_ID}" \
      -i "${APIPROXY_IMAGE}"

  # 10) Redeploy API Proxy to update the service config
  proxy_args="--tracing_sample_rate=0.0001"

  echo "Redeploying API Proxy ${PROXY_SERVICE_NAME} on Cloud Run"
  gcloud run deploy "${PROXY_SERVICE_NAME}" \
      --image="gcr.io/${PROJECT_ID}/apiproxy-serverless:${ENDPOINTS_SERVICE_NAME}-${endpoints_service_config_id}" \
      --set-env-vars=APIPROXY_ARGS="${proxy_args}" \
      --allow-unauthenticated \
      --service-account "${PROXY_RUNTIME_SERVICE_ACCOUNT}" \
      --platform managed \
      --quiet

  # Ping the proxy to startup, sleep to finish setup
  curl --silent --output /dev/null "https://${PROXY_HOST}"/shelves
  sleep 5s
  echo "Setup complete"
}

function test() {
  echo "Testing"
  local proxy_health_code=0

  # Sanity check to ensure the proxy is working
  echo "Health check against ${PROXY_HOST} host"
  proxy_health_code=$(curl --write-out %{http_code} --silent --output /dev/null "https://${PROXY_HOST}"/shelves)
  if [[ "$proxy_health_code" -ne 200 ]] ; then
    echo "Proxy status is $proxy_health_code, failing test"
    return 1
  fi
  echo "Proxy is healthy"

  # Wait a few minutes for service to be enabled and the permissions to propagate
  echo "Waiting for the endpoints service to be enabled"
  sleep 10m

  run_nonfatal long_running_test  \
      "${PROXY_HOST}"  \
      "https" \
      "443" \
      "${DURATION_IN_HOUR}"  \
      ""  \
      "${ENDPOINTS_SERVICE_NAME}"  \
      "${LOG_DIR}"  \
      "${TEST_ID}"  \
      "${UNIQUE_ID}" \
      "cloud-run" \
      || STATUS=${?}
  echo "Testing complete with status ${STATUS}"
}

function tearDown() {
  echo "Waiting for Stackdriver Logging to collect logs"
  sleep 1m

  echo "Teardown env"

  # Delete the API Proxy Cloud Run service
  gcloud run services delete "${PROXY_SERVICE_NAME}" \
      --platform managed \
      --quiet || true

  # Delete the backend Cloud Run service
  gcloud run services delete "${BOOKSTORE_SERVICE_NAME}" \
      --platform managed \
      --quiet || true

  # Delete the endpoints service config
  gcloud endpoints services delete "${ENDPOINTS_SERVICE_NAME}" \
      --quiet || true

  echo "Teardown complete successfully"
}

run_nonfatal setup || STATUS=${?}

if [[ "$STATUS" == 0 ]] ; then
  run_nonfatal test || STATUS=${?}
fi

# Ignore pipe errors on cleanup
set +o pipefail
tearDown || true
set -o pipefail

exit ${STATUS}
