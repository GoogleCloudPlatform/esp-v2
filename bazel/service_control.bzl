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

SERVICE_CONTROL_CLIENT_SHA1 = "8189638f8e2c410010befe5dbd81e267a15e3e17"

def service_control_client_repositories(bind = True):
    git_repository(
        name = "servicecontrol_client_git",
        commit = SERVICE_CONTROL_CLIENT_SHA1,
        remote = "https://github.com/cloudendpoints/service-control-client-cxx.git",
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
