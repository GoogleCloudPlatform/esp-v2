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
gcloud components update -q

PROJECT_ID="cloudesf-testing"
TEST_ID="cloud-run-${BACKEND}"
PROXY_RUNTIME_SERVICE_ACCOUNT="e2e-cloud-run-proxy-rt@${PROJECT_ID}.iam.gserviceaccount.com"
BACKEND_RUNTIME_SERVICE_ACCOUNT="e2e-${BACKEND_PLATFORM}-backend-rt@${PROJECT_ID}.iam.gserviceaccount.com"
JOB_KEY_PATH="$(mktemp /tmp/servie_account_cred.XXXX)"
LOG_DIR="$(mktemp -d /tmp/log.XXXX)"

# Determine names of all resources
UNIQUE_ID=$(get_unique_id | cut -c 1-6)
BACKEND_SERVICE_NAME="e2e-test-${BACKEND_PLATFORM}-${BACKEND}-${UNIQUE_ID}"

PROXY_SERVICE_NAME=$(get_proxy_service_name_with_sha "api-proxy")
PROXY_SERVICE_NAME="${PROXY_SERVICE_NAME}-${UNIQUE_ID}"
ENDPOINTS_SERVICE_TITLE=$(get_proxy_service_name_with_sha "${BACKEND}-service")
ENDPOINTS_SERVICE_TITLE="${ENDPOINTS_SERVICE_TITLE}-${UNIQUE_ID}"
ENDPOINTS_SERVICE_NAME=""
PROXY_HOST=""
BACKEND_HOST=""
# Cloud Run is only supported in a few regions currently
CLUSTER_ZONE="us-central1-a"
CLOUD_RUN_REGION="us-central1"

CLUSTER_VERSION="latest"

APP_ENGINE_IAP_CLIENT_ID="245521401045-qh1j3eq583qdkmn9m60pfc67303ps6cu.apps.googleusercontent.com"

STATUS=0
if [[ "${PROXY_PLATFORM}" == "anthos-cloud-run" ]] ; then
  CLUSTER_NAME=$(get_anthos_cluster_name_with_sha)-${UNIQUE_ID}
fi

GCLOUD_BETA="gcloud"
USE_HTTP2=""
# For grpc echo test, need to use gcloud with --use-http2 flag
# in order to support bidirectional streaming.
if [[ ${BACKEND_PLATFORM} == "cloud-run" ]] && [[ ${BACKEND} == "echo" ]]; then
  USE_HTTP2="--use-http2"
fi

function deployBackend() {
  case ${BACKEND_PLATFORM} in
    "cloud-run")

      local backend_image=""
      local backend_port=8080

      # Determine the backend image.
      case ${BACKEND} in
        "bookstore")
          backend_image="gcr.io/cloudesf-testing/http-bookstore:3"
          backend_port=8080
          ;;
        "echo")
          backend_image="gcr.io/cloudesf-testing/grpc-echo-server:latest"
          backend_port=8081
          ;;
        *)
          echo "No such backend image for backend ${BACKEND}"
          exit 1
          ;;
      esac

      ${GCLOUD_BETA} run deploy "${BACKEND_SERVICE_NAME}" ${USE_HTTP2} \
        --image="${backend_image}" \
        --port="${backend_port}" \
        --no-allow-unauthenticated \
        --service-account "${BACKEND_RUNTIME_SERVICE_ACCOUNT}" \
        --platform managed \
        --quiet

      BACKEND_HOST=$(gcloud run services describe "${BACKEND_SERVICE_NAME}"  --platform=managed --quiet --format="value(status.address.url)")
      ;;
    "anthos-cloud-run")
      gcloud run deploy "${BACKEND_SERVICE_NAME}" \
        --image="gcr.io/cloudesf-testing/http-bookstore:3" \
        --platform=gke \
        --quiet

      BACKEND_HOST=$(gcloud run services describe "${BACKEND_SERVICE_NAME}"   --platform=gke --quiet --format="value(status.address.url)")
      ;;
    "cloud-function")
      cd ${ROOT}/tests/endpoints/bookstore
      gcloud functions deploy ${BACKEND_SERVICE_NAME}  --runtime nodejs12 \
        --trigger-http --service-account "${BACKEND_RUNTIME_SERVICE_ACCOUNT}" \
        --quiet --entry-point app
      cd ${ROOT}

      gcloud functions remove-iam-policy-binding ${BACKEND_SERVICE_NAME} \
        --member=allUsers --role=roles/cloudfunctions.invoker --quiet || true

      BACKEND_HOST=$(gcloud functions describe ${BACKEND_SERVICE_NAME}  --format="value(httpsTrigger.url)" --quiet)
      ;;
    "app-engine")
      cd ${ROOT}/tests/endpoints/bookstore

      sed "s/SERVICE_NAME/${BACKEND_SERVICE_NAME}/g" app_template.yaml > app.yaml
      gcloud app deploy --quiet
      sleep_wrapper "1m" "Sleep 1m for App Engine backend setup"


      # For how requests are routed in App Engine, refer to
      # https://cloud.google.com/appengine/docs/standard/python/how-requests-are-routed#example_urls
      BACKEND_HOST="https://${BACKEND_SERVICE_NAME}-dot-cloudesf-testing.uc.r.appspot.com"

      cd ${ROOT}
      ;;

    *)
      echo "No such backend platform ${BACKEND_PLATFORM}"
      exit 1
      ;;
  esac
}

function deployProxy() {
  local image_name="${1}"
  local env_vars="${2}"
  local args=" --image=${image_name} --quiet"
  if [[ -n ${env_vars} ]];
  then
    args+=" --set-env-vars=ESPv2_ARGS=${proxy_args}"
  fi

  case ${PROXY_PLATFORM} in
    "cloud-run")
      args+=" --allow-unauthenticated --service-account=${PROXY_RUNTIME_SERVICE_ACCOUNT} --platform=managed"
      ;;
    "anthos-cloud-run")
      args+=" --platform=gke"
      ;;
    *)
      echo "No such backend platform ${PROXY_PLATFORM}"
      exit 1
      ;;
  esac

  ${GCLOUD_BETA} run deploy "${PROXY_SERVICE_NAME}" ${USE_HTTP2} ${args}
}


function setup() {
  echo "Setup env"
  local bookstore_health_code=0
  local proxy_args=""
  local endpoints_service_config_id=""
  local backend_protocol=""

  # Get the service account for the prow job due to b/144867112
  # TODO(b/144445217): We should let prow handle this instead of manually doing so
  get_test_client_key "gob-prow-jobs-service-account.json" "${JOB_KEY_PATH}"
  gcloud auth activate-service-account --key-file="${JOB_KEY_PATH}"


  if [[ -n ${CLUSTER_NAME} ]] ;
  then
    # Cloud Run version depends on the cluster version and the auto-assigned version may not work.
    # For details, refer to https://cloud.google.com/run/docs/gke/cluster-versions.
    # b/142752619: The cluster version should be >= 1.15 to be compatible with istio.
    gcloud container clusters create ${CLUSTER_NAME} \
      --addons=HorizontalPodAutoscaling,HttpLoadBalancing,CloudRun \
      --machine-type=e2-standard-4 \
      --cluster-version=${CLUSTER_VERSION} \
      --enable-stackdriver-kubernetes \
      --service-account=${PROXY_RUNTIME_SERVICE_ACCOUNT} \
      --network=default \
      --zone=${CLUSTER_ZONE} \
      --scopes cloud-platform
    sleep_wrapper "1m" "Sleep 1m for Anthos cluster setup"
    gcloud config set run/cluster ${CLUSTER_NAME}
    gcloud config set run/cluster_location ${CLUSTER_ZONE}
  else
    gcloud config set run/region ${CLOUD_RUN_REGION}
  fi



  # Deploy backend service (authenticated) and set BACKEND_HOST
  echo "Deploying backend ${BACKEND_SERVICE_NAME} on ${BACKEND_PLATFORM}"
  deployBackend

  #  # Only enable for http backends with external IP.
  #  # Verify the backend is up using the identity of the current machine/user
  if [[ "${PROXY_PLATFORM}" == "cloud-run"  && "${BACKEND}" == "bookstore" ]]; then
    if [[ ${BACKEND_PLATFORM} == "app-engine" ]]; then
      token=$(gcloud auth print-identity-token --audiences=${APP_ENGINE_IAP_CLIENT_ID})
    else
      token=$(gcloud auth print-identity-token)
    fi

    bookstore_health_code=$(curl \
        --write-out %{http_code} \
        --silent \
        --output /dev/null \
        -H "Authorization: Bearer ${token}" \
      "${BACKEND_HOST}"/shelves)

    if [[ "$bookstore_health_code" -ne 200 ]] ; then
      echo "Backend status is $bookstore_health_code, failing test"
      return 1
    fi
  fi



  # For Cloud Run(Fully managed), deploy initial ESPv2 service to get assigend host
  if [[ ${PROXY_PLATFORM} == "cloud-run" ]]; then
    echo "Deploying ESPv2 ${BACKEND_SERVICE_NAME} on Cloud Run(Fully managed)"
    deployProxy "${APIPROXY_IMAGE}" ""
  fi


  # Get url of ESPv2 service
  if [[ ${PROXY_PLATFORM} == "anthos-cloud-run" ]]; then
    # The internal host of cloud run on anthos
    PROXY_HOST="${PROXY_SERVICE_NAME}.default.example.com"
    # The proxy host is inaccessible and cannot be verified by servicemanagement
    # so fake a host name for openapi service config.
    ENDPOINTS_SERVICE_NAME="${PROXY_SERVICE_NAME}.endpoints.cloudesf-testing.cloud.goog"
    local scheme="http"
    backend_protocol="http/1.1"
  else
    PROXY_HOST=$(gcloud run services describe "${PROXY_SERVICE_NAME}" \
        --platform=managed \
        --format="value(status.address.url.basename())" \
      --quiet)
    ENDPOINTS_SERVICE_NAME=${PROXY_HOST}
    local scheme="https"
    backend_protocol="h2"
  fi

  case "${BACKEND}" in
    'bookstore')
      local service_idl_tmpl="${ROOT}/tests/endpoints/bookstore/bookstore_swagger_template.json"
      local service_idl="${ROOT}/tests/endpoints/bookstore/bookstore_swagger.json"
      local create_service_args=${service_idl}

      # Change the `host` to point to the proxy host (required by validation in service management).
      # Change the `title` to identify this test (for readability in cloud console).
      # Change the jwt audience to point to the proxy host (required for calling authenticated endpoints).
      # Add in the `x-google-backend` to point to the backend URL (required for backend routing).
      # Modify one path with `disable_auth`.
      cat "${service_idl_tmpl}" \
        | jq ".host = \"${ENDPOINTS_SERVICE_NAME}\" \
        | .\"x-google-endpoints\"[0].name = \"${ENDPOINTS_SERVICE_NAME}\" \
        | .schemes = [\"${scheme}\"] \
        | .info.title = \"${ENDPOINTS_SERVICE_TITLE}\" \
        | .securityDefinitions.auth0_jwk.\"x-google-audiences\" = \"${PROXY_HOST}\" \
        | . + { \"x-google-backend\": { \"address\": \"${BACKEND_HOST}\", \"protocol\": \"${backend_protocol}\" } }  \
        | .paths.\"/echo_token/disable_auth\".get  +=  { \"x-google-backend\": { \"address\": \"${BACKEND_HOST}\/echo_token\/disable_auth\", \"disable_auth\": true } } "\
        > "${service_idl}"

      if [[ ${BACKEND_PLATFORM} == "app-engine" ]]; then
        tmpfile=$(mktemp)
        cp "${service_idl}" "${tmpfile}"
        cat "${tmpfile}" \
          | jq ".\"x-google-backend\" += { \"jwt_audience\": \"${APP_ENGINE_IAP_CLIENT_ID}\"  }" \
          > "${service_idl}"
      fi
      ;;
    'echo')
      local service_idl_tmpl="${ROOT}/tests/endpoints/grpc_echo/grpc-test-dynamic-routing.tmpl.yaml"
      local service_idl="${ROOT}/tests/endpoints/grpc_echo/grpc-test-dynamic-routing.yaml"
      local service_descriptor="${ROOT}/tests/endpoints/grpc_echo/proto/api_descriptor.pb"
      local create_service_args="${service_idl} ${service_descriptor}"

      # Replace values for dynamic routing.
      sed -e "s/ENDPOINTS_SERVICE_NAME/${ENDPOINTS_SERVICE_NAME}/g" \
        -e "s/ENDPOINTS_SERVICE_TITLE/${ENDPOINTS_SERVICE_TITLE}/g" \
        -e "s/BACKEND_ADDRESS/${BACKEND_HOST#https://}/g" \
        "${service_idl_tmpl}" > "${service_idl}"
      ;;
    *)
      echo "Invalid backend ${BACKEND} for creating endpoints service"
      return 1 ;;
  esac

  # Deploy the service config
  create_service ${create_service_args}

  # Get the service name and config id to enable it
  # Assumes that the names of the endpoinds service and cloud run host match
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

  # Redeploy ESPv2 to update the service config. Set flags as follows:
  proxy_args="^++^--tracing_sample_rate=0.0005"

  if [[ ${PROXY_PLATFORM} == "cloud-run" ]];
  then
    echo "Redeploying ESPv2 ${BACKEND_SERVICE_NAME} on Cloud Run(Fully managed)"
    # - Hops: Allow our fake client IP restriction test (via API keys) to function.
    #         If we were restricting by our actual client ip, then the default of 0 would work.
    #         But we are actually testing with a fake xff header, so we need a higher hops count.
    #         On GKE we default to 2. AppHosting infra adds one more IP to xff, so 3 for serverless.
    proxy_args="${proxy_args}++--envoy_xff_num_trusted_hops=3"
  else
    echo "Deploying ESPv2 ${BACKEND_SERVICE_NAME} on Cloud Run(Anthos)"
    # - Hops: Allow our fake client IP restriction test (via API keys) to function.
    #         Anthos has 2 more proxies than Cloud Run(Fully managed).
    proxy_args="${proxy_args}++--envoy_xff_num_trusted_hops=5"
  fi
  proxy_args="${proxy_args}++--enable_debug"

  deployProxy "gcr.io/${PROJECT_ID}/endpoints-runtime-serverless:custom-${ENDPOINTS_SERVICE_NAME}-${endpoints_service_config_id}"  "${proxy_args}"
}

function test_disable_auth() {
  local fake_token="FAKE-TOKEN"
  local echoed_overrided_token
  local echoed_unoverrided_token
  echoed_overrided_token=$(curl "https://${PROXY_HOST}/echo_token/default_enable_auth" -H "Authorization:${fake_token}")
  echoed_unoverrided_token=$(curl "https://${PROXY_HOST}/echo_token/disable_auth" -H "Authorization:${fake_token}")

  if [ "${echoed_unoverrided_token}" != "\"${fake_token}\"" ] ||  [ "${echoed_overrided_token}" == "\"${fake_token}\"" ]; then
    echo "disable_auth field of X-Google-Backend in Openapi does not work"
    return 1
  fi
}

function test() {
  echo "Testing"

  # Wait a few minutes for service to be enabled and the permissions to propagate
  # If not waiting long enough, some stress tests may get 403 permission denied. b/250920830
  sleep_wrapper "10m" "Sleep 10m for the endpoints service to be enabled"

  if [[ ${PROXY_PLATFORM} == "anthos-cloud-run" ]];
  then
    local scheme="http"
    local port="80"

    # Get the external ip of cluster
    sleep_wrapper "3m" "Sleep 3m for external IP to be allocated"
    local host
    host=$( kubectl get svc istio-ingress -n gke-system | awk 'END {print $4}')
    if [[ $host == *"pending"* ]]; then
      echo "IP Address not allocated, even after sleep"
      return 1
    fi

    # Pass the real host by header `HOST`
    local host_header=${PROXY_HOST}
    local service_name=${PROXY_HOST}
  else
    local scheme="https"
    local port="443"
    local host=${PROXY_HOST}
    local service_name=${ENDPOINTS_SERVICE_NAME}
  fi

  run_nonfatal long_running_test  \
    "${host}"  \
    "${scheme}" \
    "${port}" \
    "${DURATION_IN_HOUR}"  \
    ""  \
    "${service_name}"  \
    "${LOG_DIR}"  \
    "${TEST_ID}"  \
    "${UNIQUE_ID}" \
    "cloud-run" \
    "${host_header}" \
    || STATUS=${?}

  echo "Testing complete with status ${STATUS}"
}

function tearDown() {
  sleep_wrapper "1m" "Sleep 1m for Stackdriver Logging to collect logs"

  echo "Teardown env"

  # Delete the ESPv2 Cloud Run service


  case ${BACKEND_PLATFORM} in
    "cloud-run")
      # Delete the backend Cloud Run service
      gcloud run services delete "${BACKEND_SERVICE_NAME}" \
        --platform=managed \
        --quiet || true
      gcloud run services delete "${PROXY_SERVICE_NAME}" \
        --platform=managed \
        --quiet || true
      ;;
    "anthos-cloud-run")
      # Delete the whole Anthos cluster
      gcloud container clusters delete ${CLUSTER_NAME} --quiet --region ${CLUSTER_ZONE}
      ;;

    "cloud-function")
      gcloud functions delete "${BACKEND_SERVICE_NAME}" --quiet || true
      ;;

    "app-engine")
      gcloud app services delete "${BACKEND_SERVICE_NAME}" --quiet
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
