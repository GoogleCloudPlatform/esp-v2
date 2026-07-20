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

http_archive(
    name = "bazel_skylib",
    sha256 = "3b5b49006181f5f8ff626ef8ddceaa95e9bb8ad294f7b5d7b11ea9f7ddaf8c59",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-skylib/releases/download/1.9.0/bazel-skylib-1.9.0.tar.gz",
        "https://github.com/bazelbuild/bazel-skylib/releases/download/1.9.0/bazel-skylib-1.9.0.tar.gz",
    ],
)

http_archive(
    name = "com_google_absl",
    sha256 = "4314e2a7cbac89cac25a2f2322870f343d81579756ceff7f431803c2c9090195",
    strip_prefix = "abseil-cpp-20260107.1",
    urls = ["https://github.com/abseil/abseil-cpp/archive/20260107.1.tar.gz"],
)

http_archive(
    name = "quiche",
    build_file = "@envoy//bazel/external:quiche.BUILD",
    patch_args = ["-p1"],
    patch_cmds = ["find quiche/ -type f -name \"*.bazel\" -delete"],
    patches = ["//third_party/envoy:quiche.patch"],
    sha256 = "08033a0886b470d4ea836a6b785ef6ef7d638265e5523a37718cdd6d1ef6a409",
    strip_prefix = "quiche-e68fe05e70da74a3ea282d927c76f76b4bc4e710",
    urls = ["https://github.com/google/quiche/archive/e68fe05e70da74a3ea282d927c76f76b4bc4e710.tar.gz"],
)

# Updating Envoy version:
# 1) Modify `ENVOY_SHA1` and the corresponding date.
# 2) Check if there was any change in WORKSPACE of Envoy upstream. GCP Proxy
# WORKSPACE should resemble the Envoy upstream WORKSPACE.
# 3) Check if envoy_build_config/extensions_build_config.bzl is up-to-date.
# Try to match it with the one in source/extensions and comment out unneeded extensions.

ENVOY_SHA1 = "f1dd21b16c244bda00edfb5ffce577e12d0d2ec2"  # v1.38.0

ENVOY_SHA256 = "230a3c99b7813967939db2e39563e5f63e054a5758b2425fcbdc07d8c7c2ea1f"

http_archive(
    name = "envoy",
    patch_args = ["-p1"],
    patches = [
        "//third_party/envoy:histogram_impl.patch",
        "//third_party/envoy:session_idle_list.patch",
        "//third_party/envoy:prometheus_stats.patch",
    ],
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

http_archive(
    name = "bazel_features",
    sha256 = "adfdb3cffab3a99a63363d844d559a81965d2b61a6062dd51a3d2478d416768f",
    strip_prefix = "bazel_features-1.45.0",
    url = "https://github.com/bazel-contrib/bazel_features/releases/download/v1.45.0/bazel_features-v1.45.0.tar.gz",
)

load("@bazel_features//:deps.bzl", "bazel_features_deps")

bazel_features_deps()

load("@envoy//bazel:api_repositories.bzl", "envoy_api_dependencies")

envoy_api_dependencies()

load("@envoy//bazel:repo.bzl", "envoy_repo")

envoy_repo()

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

new_local_repository(
    name = "llvm_toolchain_llvm_obsolete",
    build_file_content = """
filegroup(name = "symbolizer", srcs = [])
filegroup(name = "bin/clang", srcs = [])
    """,
    path = "third_party",
)

new_local_repository(
    name = "llvm_toolchain_llvm",
    build_file_content = """
filegroup(name = "symbolizer", srcs = [])
filegroup(name = "bin/clang", srcs = [])
exports_files(["lib/clang/18/include/fuzzer/FuzzedDataProvider.h"])
    """,
    path = "third_party",
)
