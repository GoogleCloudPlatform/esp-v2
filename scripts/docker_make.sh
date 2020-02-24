#!/bin/bash

# Copyright 2020 Google LLC

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script will use Prow docker image to make artifacts.
# The intended use case is to build envoy binary to run in the Docker image
# based on python:3.6-buster.
# Details: some developing boxes may have newer libc so their envoy binary
# could not run in image based on python:3.6-buster which has older libc.
#
# To build an envoy binary, run
#
#    scripts/docker_make.sh build-envoy
#

# The latest Prow docker image used for prow build/tests
# It is built from docker/Dockerfile-prow-env
IMAGE=gcr.io/cloudesf-testing/gcpproxy-prow:v20200207-v2.4.0-9-g17334b8-master

docker run --rm -ti -v "${PWD}":/source "${IMAGE}" \
  /bin/bash -lc "cd source && make $*"

