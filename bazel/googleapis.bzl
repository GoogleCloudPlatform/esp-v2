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
        strip_prefix = "googleapis-68c72c1d1ffff49b7d0019a21e65705b5d9c23c2",  # June 4, 2020
        url = "https://github.com/googleapis/googleapis/archive/68c72c1d1ffff49b7d0019a21e65705b5d9c23c2.tar.gz",
        sha256 = "0eaf8c4d0ea4aa3ebf94bc8f5ec57403c633920ada57a498fea4a8eb8c17b948",
    )

    if bind:
        # Bindings needed to allow envoy api build system to build cc proto.
        # Envoy will automatically look for `service_proto_cc_proto` instead of `service_cc_proto`.
        native.bind(
            name = "service_proto",
            actual = "@com_github_googleapis_googleapis//google/api:service_proto",
        )
        native.bind(
            name = "service_proto_cc_proto",
            actual = "@com_github_googleapis_googleapis//google/api:service_cc_proto",
        )
