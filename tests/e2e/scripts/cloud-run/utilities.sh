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

# This file contains utility functions that can only be used for Cloud Run

# Fail on any error.
set -eo pipefail

function get_cloud_run_service_name_with_sha() {
  local service_type=$1

  local service_format="cloudesf-testing-e2e-test-%h-${service_type}"
  local service_name="$(git show -q HEAD --pretty=format:"${service_format}")"

  echo -n "${service_name}"
  return 0
}

function get_anthos_cluster_name_with_sha() {

  local service_format="e2e-cloud-run-%h"
  local service_name="$(git show -q HEAD --pretty=format:"${service_format}")"

  echo -n "${service_name}"
  return 0
}