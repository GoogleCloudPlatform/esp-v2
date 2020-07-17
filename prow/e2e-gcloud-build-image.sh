#!/bin/bash

# E2E test for gcloud_build_image script.

# Fail on any error.
set -eo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROJECT_NAME="cloudesf-testing"

# gcloud config
gcloud config set project "${PROJECT_NAME}"
export CLOUDSDK_CORE_DISABLE_PROMPTS=1

# Use a service that is not used for any other tests.
SERVICE_NAME="hellogrpc.endpoints.cloudesf-testing.cloud.goog"
CONFIG_ID="2018-11-11r0"

function error_exit() {
  # ${BASH_SOURCE[1]} is the file name of the caller.
  echo "${BASH_SOURCE[1]}: line ${BASH_LINENO[0]}: ${1:-Unknown Error.} (exit ${2:-1})" 1>&2
  exit ${2:-1}
}

function formImageName() {
  local expected_version=$1
  echo "gcr.io/${PROJECT_NAME}/endpoints-runtime-serverless:${expected_version}-${SERVICE_NAME}-${CONFIG_ID}"
}

function cleanupOldImage() {
  local image_name=$1
  echo "Cleaning up old image if it exists (ignore any errors in the output here)."
  if gcloud container images describe "${image_name}"; then
    gcloud container images delete "${image_name}"
  fi
}

function expectImage() {
  local image_name=$1
  gcloud container images describe "${image_name}" || error_exit "Failed to find image: ${image_name}"
  echo "Successfully verified image exists: ${image_name}"
}

echo "=== Test 1: Specify a fully qualified version. ==="
EXPECTED_IMAGE_NAME=$(formImageName "2.7.0")
cleanupOldImage "${EXPECTED_IMAGE_NAME}"
${ROOT}/docker/serverless/gcloud_build_image \
    -s "${SERVICE_NAME}" \
    -c "${CONFIG_ID}" \
    -p "${PROJECT_NAME}" \
    -v "2.7.0"
expectImage "${EXPECTED_IMAGE_NAME}"

echo "=== Test 2: Specify a minor version. ==="
EXPECTED_IMAGE_NAME=$(formImageName "2.4.0")
cleanupOldImage "${EXPECTED_IMAGE_NAME}"
${ROOT}/docker/serverless/gcloud_build_image \
    -s "${SERVICE_NAME}" \
    -c "${CONFIG_ID}" \
    -p "${PROJECT_NAME}" \
    -v "2.4"
expectImage "${EXPECTED_IMAGE_NAME}"

echo "=== Test 3: Sepcify an invalid version fails. ==="
if ${ROOT}/docker/serverless/gcloud_build_image \
    -s "${SERVICE_NAME}" \
    -c "${CONFIG_ID}" \
    -p "${PROJECT_NAME}" \
    -v "2.11.47"; then
  error_exit "Script should fail for invalid version."
else
  echo "Script failed as expected."
fi

echo "=== Test 4: Specify a custom image. ==="
EXPECTED_IMAGE_NAME=$(formImageName "custom")
cleanupOldImage "${EXPECTED_IMAGE_NAME}"
${ROOT}/docker/serverless/gcloud_build_image \
    -s "${SERVICE_NAME}" \
    -c "${CONFIG_ID}" \
    -p "${PROJECT_NAME}" \
    -i "gcr.io/cloudesf-testing/apiproxy-serverless:gcloud-build-image-test"
expectImage "${EXPECTED_IMAGE_NAME}"

echo "=== Test 5: When no ESP version is specified, the script uses the latest ESPv2 release. ==="
# Knowing the latest ESP version number is hard, it depends on what is tagged in GCR.
# This is a chicken and egg problem, because `gcloud_build_image` uses that.
# That means we don't have a reliable way of checking if the output is correct.
# So just test the script passes, and allow the developer to manually verify the output.
${ROOT}/docker/serverless/gcloud_build_image \
    -s "${SERVICE_NAME}" \
    -c "${CONFIG_ID}" \
    -p "${PROJECT_NAME}"
echo ">>> WARNING: For the test above, manually verify the output version of the image is expected."
