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

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

def googleapis_repositories(bind = True):
    http_archive(
        name = "com_github_googleapis_googleapis",
        strip_prefix = "googleapis-ab2685d8d3a0e191dc8aef83df36773c07cb3d06",  # Feb 18, 2020
        url = "https://github.com/googleapis/googleapis/archive/ab2685d8d3a0e191dc8aef83df36773c07cb3d06.tar.gz",
        sha256 = "d4072ff0000e1dcb3a0a80930a628d860d5be55ebdf9733297206f4cd941cdd4",
    )