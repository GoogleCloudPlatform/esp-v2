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

load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

GOOGLE_PROTOBUF_SHA1 = "582743bf40c5d3639a70f98f183914a2c0cd0680"  # Match SHA used by Envoy
PUBREF_PROTOBUF_SHA1 = "563b674a2ce6650d459732932ea2bc98c9c9a9bf"  # Nov 28, 2017 (bazel 0.8.0 support)

def protobuf_repositories(load_repo = True, bind = True):
    if load_repo:
        git_repository(
            name = "com_google_protobuf",
            commit = GOOGLE_PROTOBUF_SHA1,
            remote = "https://github.com/google/protobuf.git",
        )
        git_repository(
            name = "org_pubref_rules_protobuf",
            commit = PUBREF_PROTOBUF_SHA1,
            remote = "https://github.com/pubref/rules_protobuf",
        )

    if bind:
        native.bind(
            name = "protoc",
            actual = "@com_google_protobuf//:protoc",
        )

        native.bind(
            name = "protocol_compiler",
            actual = "@com_google_protobuf//:protoc",
        )

        native.bind(
            name = "protobuf",
            actual = "@com_google_protobuf//:protobuf",
        )

        native.bind(
            name = "cc_wkt_protos",
            actual = "@com_google_protobuf//:cc_wkt_protos",
        )

        native.bind(
            name = "cc_wkt_protos_genproto",
            actual = "@com_google_protobuf//:cc_wkt_protos_genproto",
        )

        native.bind(
            name = "protobuf_compiler",
            actual = "@com_google_protobuf//:protoc_lib",
        )

        native.bind(
            name = "protobuf_clib",
            actual = "@com_google_protobuf//:protoc_lib",
        )
