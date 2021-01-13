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

# Running ASan tests requires compiler-rt.
# This depends on your clang install path, so place in a one-time setup script.
# Ref: https://github.com/envoyproxy/envoy/blob/master/bazel/setup_clang.sh
# Ref: https://github.com/envoyproxy/envoy/issues/14489

set -eo pipefail

LLVM_CONFIG_PATH="$(command -v llvm-config-10)"
LLVM_CONFIG_LIB_PATH="$(llvm-config-10 --libdir)"
RT_LIBRARY_PATH="$(dirname "$(find "${LLVM_CONFIG_LIB_PATH}" -name libclang_rt.ubsan_standalone_cxx-x86_64.a | head -1)")"
BAZELRC_FILE="${BAZELRC_FILE:-$(bazel info workspace)/.bazelrc}"

echo "
#
# Generated section below from ./scripts/setup-clang-asan.sh
# DO NOT check into git.
#
build:clang --action_env='LLVM_CONFIG=${LLVM_CONFIG_PATH}'
build:clang --repo_env='LLVM_CONFIG=${LLVM_CONFIG_PATH}'
build:clang --linkopt='-L${LLVM_CONFIG_LIB_PATH}'
build:clang --linkopt='-Wl,-rpath,${LLVM_CONFIG_LIB_PATH}'
build:clang-asan --linkopt='-L${RT_LIBRARY_PATH}'
build:clang-asan --linkopt=-l:libclang_rt.ubsan_standalone-x86_64.a
build:clang-asan --linkopt=-l:libclang_rt.ubsan_standalone_cxx-x86_64.a
# End generated section.
" >> "${BAZELRC_FILE}"