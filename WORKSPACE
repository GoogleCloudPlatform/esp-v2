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

# ==============================================================================
# Load all non-envoy repositories first.

load("//bazel:repositories.bzl", "all_repositories")

all_repositories()

bind(
    name = "boringssl_crypto",
    actual = "//external:ssl",
)

# ==============================================================================
# Load google protobuf dependencies due to https://github.com/protocolbuffers/protobuf/issues/5472

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()

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
# 6) Fix all backward incompatiblity issues.
# 7) Run `make proto-consistency-test` and fix inconsistency if there is any.

ENVOY_SHA1 = "7ef20d7609fb6f570a058fcf4b4e000922d7eeba"  # 07.12.2019

# TODO(kyuc): add sha256
http_archive(
    name = "envoy",
    strip_prefix = "envoy-" + ENVOY_SHA1,
    url = "https://github.com/envoyproxy/envoy/archive/" + ENVOY_SHA1 + ".zip",
)

# ==============================================================================
# Load remaining envoy dependencies.
load("@envoy//bazel:api_repositories.bzl", "envoy_api_dependencies")

envoy_api_dependencies()

load("@envoy//bazel:repositories.bzl", "envoy_dependencies")

envoy_dependencies()

load("@rules_foreign_cc//:workspace_definitions.bzl", "rules_foreign_cc_dependencies")

rules_foreign_cc_dependencies()

# ==============================================================================
# Configure C/C++ compiler.

load("@envoy//bazel:cc_configure.bzl", "cc_configure")

cc_configure()

# ==============================================================================
# Load Go dependencies and register Go toolchains.

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains()
