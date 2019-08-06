#!/bin/bash

# Copyright 2019 Google Cloud Platform Proxy Authors

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

# HTTP filter common
bazel build //api/envoy/http/common:base_go_proto
diff bazel-bin/api/envoy/http/common/*/base_go_proto%/cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/common src/go/proto/api/envoy/http/common
# HTTP filter service_control
bazel build //api/envoy/http/service_control:config_go_proto
diff bazel-bin/api/envoy/http/service_control/*/config_go_proto%/cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/service_control src/go/proto/api/envoy/http/service_control
