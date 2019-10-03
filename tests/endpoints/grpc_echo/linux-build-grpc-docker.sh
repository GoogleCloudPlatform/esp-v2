#!/bin/bash
# This script builds docker image for grpc test server.
# It can build either grpc-echo-server or grpc-interop-server.
# Its usage:
#   linux-build-grpc-docker -i gcr.io/cloudesf-testing/grpc-echo-server
#   linux-build-grpc-docker -o -i gcr.io/cloudesf-testing/grpc-interop-server

SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
GRPC_ROOT="${ROOT}/tests/endpoints/grpc-echo"

. "${ROOT}/tests/e2e/scripts/prow-utilities.sh" || {
  echo "Cannot load Bash utilities"
  exit 1
}

TEST_SERVER_TARGET='//tests/endpoints/grpc_echo:grpc-test-server'
TEST_SERVER_BIN='tests/endpoints/grpc_echo/grpc-test-server'
TEST_SERVER_ARGS='0.0.0.0:8081'

while getopts :i:o arg; do
  case ${arg} in
    i) IMAGE="${OPTARG}" ;;
    o)
      TEST_SERVER_TARGET='@com_github_grpc_grpc//test/cpp/interop:interop_server'
      TEST_SERVER_BIN='external/com_github_grpc_grpc/test/cpp/interop/interop_server'
      TEST_SERVER_ARGS='--port=8081'
      ;;
    *) error_exit "Unrecognized argument -${OPTARG}" ;;
  esac
done

[[ -n "${IMAGE}" ]] || error_exit "Specify required image argument via '-i'"

echo "Checking if docker image ${IMAGE} exists.."
gcloud docker -- pull "${IMAGE}" &&
{
  echo "Image ${IMAGE} already exists; skipping"
  exit 0
}

BAZEL_TARGET="${ROOT}/bazel-bin/${TEST_SERVER_BIN}"
if ! [[ -e "${BAZEL_TARGET}" ]]; then
  echo "Building ${TEST_SERVER_TARGET}"
  bazel build --config=release "${TEST_SERVER_TARGET}" \
    || error_exit 'Could not build ${TEST_SERVER_BIN}'
fi

cp -f "${BAZEL_TARGET}" "${GRPC_ROOT}" ||
error_exit "Could not copy ${BAZEL_TARGET} to ${GRPC_ROOT}"

sed -e "s|TEST_SERVER_BIN|$(basename ${TEST_SERVER_BIN})|g" \
  -e "s|TEST_SERVER_ARGS|${TEST_SERVER_ARGS}|g" \
  "${GRPC_ROOT}/Dockerfile.temp" >"${GRPC_ROOT}/Dockerfile"

echo "Building Endpoints Runtime grpc docker image."
retry -n 3 docker build --no-cache -t "${IMAGE}" \
  -f "${GRPC_ROOT}/Dockerfile" "${GRPC_ROOT}" ||
error_exit "Docker image build failed."

echo "Pushing Docker image: ${IMAGE}"
# Try 10 times, shortest wait is 10 seconds, exponential back-off.
retry -n 10 -s 10 \
  gcloud docker -- push "${IMAGE}" ||
error_exit "Failed to upload Docker image to gcr."
