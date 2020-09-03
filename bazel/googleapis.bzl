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
        strip_prefix = "googleapis-1d5522ad1056f16a6d593b8f3038d831e64daeea",  # Sept 03, 2020
        url = "https://github.com/googleapis/googleapis/archive/1d5522ad1056f16a6d593b8f3038d831e64daeea.tar.gz",
        sha256 = "cd13e547cffaad217c942084fd5ae0985a293d0cce3e788c20796e5e2ea54758",
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
