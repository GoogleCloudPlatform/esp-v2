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

# Script to generate auth token based on `src/tools/auth_token_gen`

SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_PATH}/../../.." && pwd)"
. ${ROOT}/scripts/all-utilities.sh || { echo 'Cannot load Bash utilities'; exit 1; }

# By default audience is service name,  use -a to change it to your service
# name or other allowed audiences (check service swagger configuration).
AUDIENCE="apiproxy.cloudendpointsapis.com"
SERVICE_ACCOUNT="245521401045-compute@developer.gserviceaccount.com"

function usage() {
  echo "usage: $0 [options ...]"
  echo "options:"
  echo "  -s <secret file>"
  echo "  -a <audience>"
  echo "  -g <path to auth_token_gen file>"
  exit 2
}

while getopts a:s:? arg; do
  case ${arg} in
    a) AUDIENCE=${OPTARG};;
    s) SECRET_FILE=${OPTARG};;
    ?) usage;;
  esac
done

# By default, use jwk key. Can be switched to x509 or symmetric key.
SECRET_FILE="${SECRET_FILE:-$(get_test_client_key ${ROOT}/tests/e2e/client/test-client-secret-jwk.json)}"

go run ${ROOT}/tests/e2e/client/jwt_client.go \
  --service-account-file=${SECRET_FILE} \
  --service-account-email=${SERVICE_ACCOUNT} \
  --audience=${AUDIENCE} | grep 'Auth token' | awk -F': ' '{print $2}'
