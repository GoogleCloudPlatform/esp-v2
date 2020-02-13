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

GOOGLEAPIS_BUILD_FILE = """
exports_files([
        "google/rpc/status.proto",
        "google/api/servicecontrol/v1/service_controller.proto",
        "google/api/servicecontrol/v1/check_error.proto",
        "google/api/servicecontrol/v1/distribution.proto",
        "google/api/servicecontrol/v1/log_entry.proto",
        "google/api/servicecontrol/v1/metric_value.proto",
        "google/api/servicecontrol/v1/operation.proto",
        "google/api/annotations.proto",
        "google/api/http.proto",
        "google/logging/type/log_severity.proto",
        "google/type/money.proto",
]
)
"""

def googleapis_repositories(bind = True):

    http_archive(
        name = "com_github_googleapis_googleapis",
        build_file_content = GOOGLEAPIS_BUILD_FILE,
        strip_prefix = "googleapis-602153361a1f309e1c1b7aba4ad69948aae1015c",  # Oct 22, 2019
        url = "https://github.com/TAOXUY/googleapis/archive/602153361a1f309e1c1b7aba4ad69948aae1015c.tar.gz",
    )

    if bind:
        native.bind(
            name = "servicecontrol",
            actual = "@com_github_googleapis_googleapis//google/api/servicecontrol/v1:servicecontrol_cc_proto",
        )

        native.bind(
            name = "service_config",
            actual = "@com_github_googleapis_googleapis//google/api:service_cc_proto",
        )
