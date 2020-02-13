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

SERVICE_CONTROL_CLIENT_GIT_SHA = "e9af22b1703d4873a250b9e09a7287d30f8de542"
SERVICE_CONTROL_CLIENT_SHA = "91898d7feffddfe93f76824d3d553f095c24e06c581c3216afcfd69c00b44885"

def service_control_client_repositories(bind = True):
    http_archive(
        name = "servicecontrol_client_git",
        #        sha256 = SERVICE_CONTROL_CLIENT_SHA,
        strip_prefix = "service-control-client-cxx-" + SERVICE_CONTROL_CLIENT_GIT_SHA,
        urls = ["https://github.com/cloudendpoints/service-control-client-cxx/archive/" + SERVICE_CONTROL_CLIENT_GIT_SHA + ".tar.gz"],
        repo_mapping = {"@googleapis_git": "@com_github_googleapis_googleapis"},
    )

    if bind:
        native.bind(
            name = "servicecontrol_client",
            actual = "@servicecontrol_client_git//:service_control_client_lib",
        )

        native.bind(
            name = "quotacontrol",
            actual = "@servicecontrol_client_git//proto:quotacontrol",
        )

        native.bind(
            name = "quotacontrol_genproto",
            actual = "@servicecontrol_client_git//proto:quotacontrol_genproto",
        )
