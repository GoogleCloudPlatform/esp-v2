#!/usr/bin/env bash

# Fail on any error.
set -e

rm -rf src/go/proto
rm -rf vendor/github.com/envoyproxy/data-plane-api/api
rm -rf vendor/gogoproto
rm -rf vendor/github.com/census-instrumentation/opencensus-proto/gen-go

#TODO(bochun): probably we can programatically generate these.
# HTTP filter common
bazel build //api/envoy/http/common:base_proto_go_proto
mkdir -p src/go/proto/api/envoy/http/common
cp -f bazel-bin/api/envoy/http/common/*/base_proto_go_proto%/cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/common/* src/go/proto/api/envoy/http/common
# HTTP filter service_control
bazel build //api/envoy/http/service_control:config_proto_go_proto
mkdir -p src/go/proto/api/envoy/http/service_control
cp -f bazel-bin/api/envoy/http/service_control/*/config_proto_go_proto%/cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/service_control/* src/go/proto/api/envoy/http/service_control
# HTTP filter path_matcher
bazel build //api/envoy/http/path_matcher:config_proto_go_proto
mkdir -p src/go/proto/api/envoy/http/path_matcher
cp -f bazel-bin/api/envoy/http/path_matcher/*/config_proto_go_proto%/cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/path_matcher/* src/go/proto/api/envoy/http/path_matcher
# HTTP filter backend_auth
bazel build //api/envoy/http/backend_auth:config_proto_go_proto
mkdir -p src/go/proto/api/envoy/http/backend_auth
cp -f bazel-bin/api/envoy/http/backend_auth/*/config_proto_go_proto%/cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/backend_auth/* src/go/proto/api/envoy/http/backend_auth
# HTTP filter backend_routing
bazel build //api/envoy/http/backend_routing:config_proto_go_proto
mkdir -p src/go/proto/api/envoy/http/backend_routing
cp -f bazel-bin/api/envoy/http/backend_routing/*/config_proto_go_proto%/cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/backend_routing/* src/go/proto/api/envoy/http/backend_routing

# envoy protos
bazel build @envoy_api//envoy/...
mkdir vendor/github.com/envoyproxy/data-plane-api/api

# ----------------------------------------------------------------------------
# Copy generated go_proto files into `vendor/`.
# This section of code contains a few workarounds for issues with the protos in envoy_api...
#
# In the envoy_api repo, all the proto definitions are in a nested folder structure.
# However, after go_protos are generated with bazel, the protos assume the folder structure is flat.
#   Example) Imports inside generated go_protos directly reference:
#   `github.com/envoyproxy/data-plane-api/api/${single-proto-name}/`
# So all the go_protos from the nested folder structure must be copied into a flat structure,
#   with a single folder for each go_proto that matches the name of the proto definition.
# This is handled by the `find` and the loop over directories.
#
# Inside the envoy_api repository, some protos in different locations have the same name.
#   Example) AccessLog is in `config/filter/accesslog/v2/` and `data/accesslog/v2/`
#   These are two completely different protos with the same name.
# When copied to a flat folder structure, these names conflict and possibly overwrite each other.
# Therefore, we prioritize the ordering of the imports below to copy the ones we care about.
# Using `cp -n`, we can ignore the duplicated second go_proto that is lower priority.
# ----------------------------------------------------------------------------
echo "Copying files from 'bazel-bin/' to 'vendor/github.com/envoyproxy/data-plane-api/api/' in priority ordering"
dirs=$(find \
  ./bazel-bin/external/envoy_api/envoy/api \
  ./bazel-bin/external/envoy_api/envoy/config/filter \
  ./bazel-bin/external/envoy_api/envoy/config/ \
  ./bazel-bin/external/envoy_api/envoy/type \
  ./bazel-bin/external/envoy_api/envoy/service \
  ./bazel-bin/external/envoy_api/envoy/data \
  ./bazel-bin/external/envoy_api/envoy/admin \
  -name '*.pb.go' -exec dirname {} \;)
for dir in ${dirs} ; do
  cp -nr ${dir} vendor/github.com/envoyproxy/data-plane-api/api
done

# envoy protos dependency
cp -nr ./bazel-bin/external/com_github_gogo_protobuf/*/gogo_proto_go%/gogoproto vendor/
cp -nr ./bazel-bin/external/opencensus_proto/opencensus/proto/*/*/*/*%/github.com/ vendor/