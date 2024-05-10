# Copyright 2019 Google LLC
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
# Try to match it with the one in source/extensions and comment out unneeded extensions.

ENVOY_SHA1 = "816188b86a0a52095b116b107f576324082c7c02"  # v1.30.1

ENVOY_SHA256 = "41064ee8fbafc2ac3fd0d6531a106139adbbea3585defff22fdb99e06d7862e5"

http_archive(
    name = "envoy",
    sha256 = ENVOY_SHA256,
    strip_prefix = "envoy-" + ENVOY_SHA1,
    url = "https://github.com/envoyproxy/envoy/archive/" + ENVOY_SHA1 + ".zip",
)

# A hack to load zlib first before loading envoy_dependencies.
# grpc_deps() is using @zlib directly. But Envoy doesn't have @zlib, it only
# binds zlib to net_zlib so grpc_deps() fails.
# It is copied from https://github.com/grpc/grpc/blob/master/bazel/grpc_deps.bzl
http_archive(
    name = "zlib",
    build_file = "//bazel:zlib.BUILD",
    sha256 = "6d4d6640ca3121620995ee255945161821218752b551a1a180f4215f7d124d45",
    strip_prefix = "zlib-cacf7f1d4e3d44d871b605da3b647f07d718623f",
    url = "https://github.com/madler/zlib/archive/cacf7f1d4e3d44d871b605da3b647f07d718623f.tar.gz",
)

# ==============================================================================
# Load remaining envoy dependencies.
load("@envoy//bazel:api_binding.bzl", "envoy_api_binding")

envoy_api_binding()

load("@envoy//bazel:api_repositories.bzl", "envoy_api_dependencies")

envoy_api_dependencies()

load("@envoy//bazel:repositories.bzl", "envoy_dependencies")

envoy_dependencies()

load("@envoy//bazel:repositories_extra.bzl", "envoy_dependencies_extra")

envoy_dependencies_extra(ignore_root_user_error = True)

load("@envoy//bazel:python_dependencies.bzl", "envoy_python_dependencies")

envoy_python_dependencies()

load("@base_pip3//:requirements.bzl", "install_deps")

install_deps()

load("@envoy//bazel:dependency_imports.bzl", "envoy_dependency_imports")

envoy_dependency_imports()

# ==============================================================================
# Load service control repositories
load("//bazel:repositories.bzl", "service_control_repositories")

service_control_repositories()

load("@io_bazel_rules_python//python:pip.bzl", "pip_install")

pip_install(
    name = "grpc_python_dependencies",
    requirements = "@com_github_grpc_grpc//:requirements.bazel.txt",
)

load("@com_github_grpc_grpc//bazel:grpc_deps.bzl", "grpc_deps", "grpc_test_only_deps")

# ==============================================================================
load("@com_github_grpc_grpc//bazel:grpc_python_deps.bzl", "grpc_python_deps")

grpc_python_deps()

grpc_deps()

grpc_test_only_deps()

load("//bazel:grpc.bzl", "grpc_bindings")

grpc_bindings()
