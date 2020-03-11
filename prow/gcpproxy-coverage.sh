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

# Presubmit script triggered by Prow.

# Fail on any error.
set -eo pipefail

gcloud config set core/project cloudesf-testing
gcloud auth activate-service-account \
  --key-file="${GOOGLE_APPLICATION_CREDENTIALS}"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"
. ${ROOT}/scripts/all-utilities.sh || { echo 'Cannot load Bash utilities';
exit 1; }

echo '======================================================='
echo '===================== Setup Cache ====================='
echo '======================================================='
try_setup_bazel_remote_cache "${PROW_JOB_ID}" "${IMAGE}" "${ROOT}" "${JOB_TYPE}-coverage"


echo '======================================================='
echo '==================== C++ Coverage ====================='
echo '======================================================='
. ${ROOT}/third_party/tools/coverage/cpp_unit.sh

echo '======================================================='
echo '=================== Upload Coverage ==================='
echo '======================================================='

# Note that JOB_TYPE is set by Prow.
# https://github.com/kubernetes/test-infra/blob/master/prow/jobs.md#job-environment-variables
PUBLIC_DIRECTORY=""
case "${JOB_TYPE}" in
  "presubmit")
    # Store in directory with the SHA for each presubmit run.
    PUBLIC_DIRECTORY=$(get_tag_name)
    ;;
  "periodic")
    # Overwrite global directory with latest coverage for all continuous runs.
    PUBLIC_DIRECTORY="latest"
    ;;
  *)
    # If running locally, just upload to special-case folder.
    echo "Unknown job type: ${JOB_TYPE}"
    PUBLIC_DIRECTORY="local-run"
    ;;
esac

# Upload folder.
gsutil -m rsync -r -d "${ROOT}/generated" "gs://esp-v2-coverage/${PUBLIC_DIRECTORY}"

# No browser cache since some directories change often and that would be misleading.
gsutil -m setmeta -h "Cache-Control:private, max-age=0, no-transform" "gs://esp-v2-coverage/${PUBLIC_DIRECTORY}/**"

echo '======================================================='
echo '==================== View Coverage ===================='
echo '======================================================='
echo "C++ Unit test coverage is viewable at the URL below"
echo "https://storage.googleapis.com/esp-v2-coverage/${PUBLIC_DIRECTORY}/third_party/tools/coverage/coverage_tests/index.html"
