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
        strip_prefix = "googleapis-736857e7a655eea72322e078b1988bd0d25aae0f",  # 10/19/2022
        url = "https://github.com/googleapis/googleapis/aPPrchive/736857e7a655eea72322e078b1988bd0d25aae0f.tar.gz",
        sha256 = "b165b0f397f143d2e09d22c51aa90028d24ac3b755a103688e7a49090993155f",
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
        native.bind(
            name = "service_proto_py_proto",
            actual = "@com_github_googleapis_googleapis//google/api:service_py_proto",
        )
        native.bind(
            name = "service_proto_py_proto_genproto",
            actual = "@com_github_googleapis_googleapis//google/api:service_py_proto_genproto",
        )
