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

rm -rf src/go/proto
rm -rf vendor/github.com/envoyproxy/data-plane-api/api
rm -rf vendor/gogoproto
rm -rf vendor/github.com/census-instrumentation/opencensus-proto/gen-go

#TODO(bochun): probably we can programatically generate these.
# HTTP filter common
bazel build //api/envoy/http/common:base_go_proto
mkdir -p src/go/proto/api/envoy/http/common
cp -f bazel-bin/api/envoy/http/common/*/base_go_proto%/github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/common/* src/go/proto/api/envoy/http/common
# HTTP filter service_control
bazel build //api/envoy/http/service_control:config_go_proto
mkdir -p src/go/proto/api/envoy/http/service_control
cp -f bazel-bin/api/envoy/http/service_control/*/config_go_proto%/github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/service_control/* src/go/proto/api/envoy/http/service_control
# HTTP filter path_matcher
bazel build //api/envoy/http/path_matcher:config_go_proto
mkdir -p src/go/proto/api/envoy/http/path_matcher
cp -f bazel-bin/api/envoy/http/path_matcher/*/config_go_proto%/github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/path_matcher/* src/go/proto/api/envoy/http/path_matcher
# HTTP filter backend_auth
bazel build //api/envoy/http/backend_auth:config_go_proto
mkdir -p src/go/proto/api/envoy/http/backend_auth
cp -f bazel-bin/api/envoy/http/backend_auth/*/config_go_proto%/github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/backend_auth/* src/go/proto/api/envoy/http/backend_auth
# HTTP filter backend_routing
bazel build //api/envoy/http/backend_routing:config_go_proto
mkdir -p src/go/proto/api/envoy/http/backend_routing
cp -f bazel-bin/api/envoy/http/backend_routing/*/config_go_proto%/github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/backend_routing/* src/go/proto/api/envoy/http/backend_routing
# HTTP filter error_translator
bazel build //api/envoy/http/error_translator:config_go_proto
mkdir -p src/go/proto/api/envoy/http/error_translator
cp -f bazel-bin/api/envoy/http/error_translator/*/config_go_proto%/github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/error_translator/* src/go/proto/api/envoy/http/error_translator