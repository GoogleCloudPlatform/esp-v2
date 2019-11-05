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

# This file contains utility functions that can only be used for GKE

# Fail on any error.
set -eo pipefail

function get_cluster_host() {
  local COUNT=10
  local SLEEP=15
  for i in $(seq 1 ${COUNT}); do
    local host=$(kubectl get service app -n ${1} | awk '{print $4}' | grep -v EXTERNAL-IP)
    [ '<pending>' != $host ] && break
    echo "Waiting for server external ip. Attempt  #$i/${COUNT}... will try again in ${SLEEP} seconds" >&2
    sleep ${SLEEP}
  done
  if [[ '<pending>' == $host ]]; then
    echo 'Failed to get the GKE cluster host.'
    return 1
  else
    echo "$host"
    return 0
  fi
}

# Fetch proxy logs from k8s container
function fetch_proxy_logs() {
  local namespace=${1}
  local log_dir=${2}
  local pod_id=$(kubectl get --no-headers=true pods -l app=app -n ${namespace} -o custom-columns=:metadata.name)
  touch ${LOG_DIR}/error.log
  (kubectl logs -p ${pod_id} -c apiproxy -n ${namespace} | tee -a ${LOG_DIR}/error.log) || echo "No apiproxy container crashed"
  kubectl logs ${pod_id} -c apiproxy -n ${namespace} | tee -a ${LOG_DIR}/error.log
}
