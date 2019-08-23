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

load(":googleapis.bzl", "googleapis_repositories")
load(":service_control.bzl", "service_control_client_repositories")
load(":bazel_rules_python.bzl", "bazel_rules_python_repositories")

def service_control_repositories():
    googleapis_repositories()
    service_control_client_repositories()
    bazel_rules_python_repositories()
