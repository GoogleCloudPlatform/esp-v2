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

load(
    "@bazel_tools//tools/build_defs/repo:git.bzl",
    "git_repository",
)

def bazel_rules_python_repositories(load_repo = True):
    if load_repo:
        git_repository(
            name = "io_bazel_rules_python",
            commit = "8b5d0683a7d878b28fffe464779c8a53659fc645",
            remote = "https://github.com/bazelbuild/rules_python.git",
        )
