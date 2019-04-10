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
# Envoy extension configuration override. Must be before the envoy repository.

local_repository(
    name = "envoy_build_config",
    path = "envoy_build_config",
)

# ==============================================================================
# Load envoy repository.

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

ENVOY_SHA1 = "6d6a5692d2f22a4393ec425cc0b0512a784885e4"  # 01.07.2019

http_archive(
    name = "envoy",
    strip_prefix = "envoy-" + ENVOY_SHA1,
    url = "https://github.com/envoyproxy/envoy/archive/" + ENVOY_SHA1 + ".zip",
)

# ==============================================================================
# Load remaining envoy dependencies.

load("@envoy//bazel:repositories.bzl", "envoy_dependencies")

envoy_dependencies()

# ==============================================================================
# Configure C/C++ compiler.

load("@envoy//bazel:cc_configure.bzl", "cc_configure")

cc_configure()

# ==============================================================================
# Load remaining envoy dependencies.

load("@envoy_api//bazel:repositories.bzl", "api_dependencies")

api_dependencies()

# ==============================================================================
# Load Go dependencies and register Go toolchains.

load("@io_bazel_rules_go//go:def.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains()
