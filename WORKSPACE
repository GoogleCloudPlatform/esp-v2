# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

workspace(name = "gcpproxy")

bind(
    name = "boringssl_crypto",
    actual = "//external:ssl",
)

# ==============================================================================
# Envoy extension configuration override. Must be before the envoy repository.

local_repository(
    name = "envoy_build_config",
    path = "envoy_build_config",
)

# ==============================================================================
# Load envoy repository.

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# Updating Envoy version:
# 1) Modify `ENVOY_SHA1` and the corresponding date.
# 2) Check if there was any change in WORKSPACE of Envoy upstream. GCP Proxy
# WORKSPACE should resemble the Envoy upstream WORKSPACE.
# 3) Check if envoy_build_config/extensions_build_config.bzl is up-to-date.
# Replace it with the one from upstream and comment out unneeded extensions.
# 4) Check if `GOOGLE_PROTOBUF_SHA1` in bazel/protobuf.bzl matches with the
# version that Envoy uses.
# 5) Check if `ABSL_COMMIT` and `ABSL_SHA256` in bazel/abseil.bzl matches with
# the version that Envoy uses.
# 6) Fix all backward incompatibility issues.
# 7) Run `make proto-consistency-test` and fix inconsistency if there is any.

ENVOY_SHA1 = "7eed7332d513248a07e493bb8ec7bb3081a18b3e"  # 08.21.2019

# TODO(kyuc): add sha256
http_archive(
    name = "envoy",
    strip_prefix = "envoy-" + ENVOY_SHA1,
    url = "https://github.com/envoyproxy/envoy/archive/" + ENVOY_SHA1 + ".zip",
)

# ==============================================================================
# Load remaining envoy dependencies.
load("@envoy//bazel:api_binding.bzl", "envoy_api_binding")

envoy_api_binding()

load("@envoy//bazel:api_repositories.bzl", "envoy_api_dependencies")

envoy_api_dependencies()

load("@envoy//bazel:repositories.bzl", "envoy_dependencies")

envoy_dependencies()

load("@envoy//bazel:dependency_imports.bzl", "envoy_dependency_imports")

envoy_dependency_imports()

# ==============================================================================
# Load service control repositories
load("//bazel:repositories.bzl", "service_control_repositories")

service_control_repositories()

load("@io_bazel_rules_python//python:pip.bzl", "pip_import", "pip_repositories")

pip_import(
    name = "grpc_python_dependencies",
    requirements = "@com_github_grpc_grpc//:requirements.bazel.txt",
)

# ==============================================================================
load("@com_github_grpc_grpc//bazel:grpc_python_deps.bzl", "grpc_python_deps")
load("@com_github_grpc_grpc//bazel:grpc_deps.bzl", "grpc_deps", "grpc_test_only_deps")

grpc_python_deps()

grpc_deps()

grpc_test_only_deps()

load("//bazel:grpc.bzl", "grpc_bindings")

grpc_bindings()
