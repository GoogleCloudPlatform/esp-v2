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
PROXY_RUNTIME_SERVICE_ACCOUNT="e2e-cloud-run-proxy-rt@${PROJECT_ID}.iam.gserviceaccount.com"
BACKEND_RUNTIME_SERVICE_ACCOUNT="e2e-${BACKEND_PLATFORM}-backend-rt@${PROJECT_ID}.iam.gserviceaccount.com"
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
BOOKSTORE_HOST=""

STATUS=0

function deployEndpoints() {
  case ${BACKEND_PLATFORM} in
    "cloud-run")
      gcloud run deploy "${BOOKSTORE_SERVICE_NAME}" \
        --image="gcr.io/cloudesf-testing/app:bookstore" \
        --no-allow-unauthenticated \
        --service-account "${BACKEND_RUNTIME_SERVICE_ACCOUNT}" \
        --platform managed \
        --quiet

      BOOKSTORE_HOST=$(gcloud run services describe "${BOOKSTORE_SERVICE_NAME}"  --platform=managed --quiet --format="value(status.address.url)")
      ;;
    "cloud-function")

      cd ${ROOT}/tests/endpoints/bookstore
      gcloud functions deploy ${BOOKSTORE_SERVICE_NAME}  --runtime nodejs8 \
        --trigger-http --service-account "${BACKEND_RUNTIME_SERVICE_ACCOUNT}" \
        --quiet --entry-point app
      cd ${ROOT}

      gcloud functions remove-iam-policy-binding ${BOOKSTORE_SERVICE_NAME} \
      --member=allUsers --role=roles/cloudfunctions.invoker --quiet || true

      BOOKSTORE_HOST=$(gcloud functions describe ${BOOKSTORE_SERVICE_NAME}  --format="value(httpsTrigger.url)" --quiet)
      ;;
    *)
      echo "No such backend platform ${BACKEND_PLATFORM}"
      exit 1
      ;;
  esac
}


function setup() {
  echo "Setup env"
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

  # Deploy backend service (authenticated) and set BOOKSTORE_HOST
  echo "Deploying backend ${BOOKSTORE_SERVICE_NAME} on ${BACKEND_PLATFORM}"
  deployEndpoints

  # Verify the backend is up using the identity of the current machine/user
  bookstore_health_code=$(curl \
      --write-out %{http_code} \
      --silent \
      --output /dev/null \
      -H "Authorization: Bearer $(gcloud auth print-identity-token)" \
    "${BOOKSTORE_HOST}"/shelves)

  if [[ "$bookstore_health_code" -ne 200 ]] ; then
    echo "Backend status is $bookstore_health_code, failing test"
    return 1
  fi
  echo "Backend deployed successfully"

  # Deploy initial ESPv2 service
  echo "Deploying ESPv2 ${BOOKSTORE_SERVICE_NAME} on Cloud Run"

  gcloud run deploy "${PROXY_SERVICE_NAME}" \
    --image="${APIPROXY_IMAGE}" \
    --allow-unauthenticated \
    --service-account "${PROXY_RUNTIME_SERVICE_ACCOUNT}" \
    --platform managed \
    --quiet

  # Get url of ESPv2 service
  PROXY_HOST=$(gcloud run services describe "${PROXY_SERVICE_NAME}" \
      --platform=managed \
      --format="value(status.address.url.basename())" \
    --quiet)

  # Modify the service config for Cloud Run
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
      | . + { \"x-google-backend\": { \"address\": \"${BOOKSTORE_HOST}\" } }  \
      | .paths.\"/echo_token/disable_auth\".get  +=  { \"x-google-backend\": { \"address\": \"${BOOKSTORE_HOST}\/echo_token\/disable_auth\", \"disable_auth\": true} } "\
    > "${service_idl}"

  # Deploy the service config
  create_service "${service_idl}"

  # Get the service name and config id to enable it
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

  # Build the service config into a new image
  echo "Building serverless image"
  local build_image_script="${ROOT}/docker/serverless/gcloud_build_image"
  chmod +x "${build_image_script}"
  $build_image_script \
    -s "${ENDPOINTS_SERVICE_NAME}" \
    -c "${endpoints_service_config_id}" \
    -p "${PROJECT_ID}" \
    -i "${APIPROXY_IMAGE}"

  # Redeploy ESPv2 to update the service config
  proxy_args="--tracing_sample_rate=0.00001"

  echo "Redeploying ESPv2 ${PROXY_SERVICE_NAME} on Cloud Run"
  gcloud run deploy "${PROXY_SERVICE_NAME}" \
    --image="gcr.io/${PROJECT_ID}/endpoints-runtime-serverless:${ENDPOINTS_SERVICE_NAME}-${endpoints_service_config_id}" \
    --set-env-vars=ESPv2_ARGS="${proxy_args}" \
    --allow-unauthenticated \
    --service-account "${PROXY_RUNTIME_SERVICE_ACCOUNT}" \
    --platform managed \
    --quiet

  # Ping the proxy to startup, sleep to finish setup
  curl --silent --output /dev/null "https://${PROXY_HOST}"/shelves
  sleep 5s
  echo "Setup complete"
}

function test_disable_auth() {
  local fake_token="FAKE-TOKEN"
  local echoed_overrided_token=$(curl "https://${PROXY_HOST}/echo_token/default_enable_auth" -H "Authorization:${fake_token}")
  local echoed_unoverrided_token=$(curl "https://${PROXY_HOST}/echo_token/disable_auth" -H "Authorization:${fake_token}")

  if [ "${echoed_unoverrided_token}" != "\"${fake_token}\"" ] ||  [ "${echoed_overrided_token}" == "\"${fake_token}\"" ]; then
    echo "disable_auth field of X-Google-Backend in Openapi does not work"
    return 1
  fi
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

  if [[ ${BACKEND_PLATFORM} = "cloud-run" ]]; then
    # Inorder to test disable_auth, iam of backend should be disabled so the
      # hardcoded token can be echoed back.
      gcloud run services add-iam-policy-binding "${BOOKSTORE_SERVICE_NAME}"\
        --member="allUsers" \
        --role="roles/run.invoker" \
        --platform=managed

      # wait allow-unauthenticated to be set for the whole backend instance
      sleep 2m

      run_nonfatal test_disable_auth
  fi

  echo "Testing complete with status ${STATUS}"
}

function tearDown() {
  echo "Waiting for Stackdriver Logging to collect logs"
  sleep 1m

  echo "Teardown env"

  # Delete the ESPv2 Cloud Run service
  gcloud run services delete "${PROXY_SERVICE_NAME}" \
    --platform managed \
    --quiet || true

  case ${BACKEND_PLATFORM} in
    "cloud-run")
      # Delete the backend Cloud Run service
      gcloud run services delete "${BOOKSTORE_SERVICE_NAME}" \
        --platform managed \
        --quiet || true
      ;;
    "cloud-function")
      gcloud functions delete "${BOOKSTORE_SERVICE_NAME}" --quiet || true
      ;;
    *)
      echo "No such backend platform ${BACKEND_PLATFORM}"
      exit 1
      ;;
  esac


  # Delete the endpoints service config
  gcloud endpoints services delete "${ENDPOINTS_SERVICE_NAME}" \
    --quiet || true

  echo "Teardown complete successfully"
}

run_nonfatal setup || STATUS=${?}

if [[ ${STATUS} == 0 ]] ; then
  run_nonfatal test || STATUS=${?}
fi

# Ignore pipe errors on cleanup
set +o pipefail
tearDown || true
set -o pipefail

exit ${STATUS}
