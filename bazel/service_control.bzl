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

SERVICE_CONTROL_CLIENT_GIT_SHA = "d76546c466a11199c73d6aee2d7bb4f9b286df96"
SERVICE_CONTROL_CLIENT_SHA = "11434c32020608b71ab04c07c780a4c234037a0b0ab92d9bdbd895e570ab8ab9"

def service_control_client_repositories(bind = True):
    http_archive(
        name = "servicecontrol_client_git",
        sha256 = SERVICE_CONTROL_CLIENT_SHA,
        strip_prefix = "service-control-client-cxx-" + SERVICE_CONTROL_CLIENT_GIT_SHA,  # 2021-06-09
        urls = ["https://github.com/qiwzhang/service-control-client-cxx/archive/" + SERVICE_CONTROL_CLIENT_GIT_SHA + ".tar.gz"],
        #TODO(taoxuy): remove this mapping once Envoy googleapis_git is updated to use the version with servicecontrol_proto change
        repo_mapping = {"@googleapis_git": "@com_github_googleapis_googleapis"},
    )
