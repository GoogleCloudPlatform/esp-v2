#!/usr/bin/env bash
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
echo "!! Ignore any warnings below..."
mkdir vendor/github.com/envoyproxy/data-plane-api/api

# Force generate envoy/api first, then generate the remaining. Otherwise name conflicts will lead to api not being created
dirs=$(find \
  ./bazel-bin/external/envoy_api/envoy/api \
  ./bazel-bin/external/envoy_api/envoy/config \
  ./bazel-bin/external/envoy_api/envoy/type \
  ./bazel-bin/external/envoy_api/envoy/service \
  ./bazel-bin/external/envoy_api/envoy/data \
  ./bazel-bin/external/envoy_api/envoy/admin \
  -name '*.pb.go' -exec dirname {} \;)
for dir in ${dirs} ; do
  cp -r ${dir} vendor/github.com/envoyproxy/data-plane-api/api || true # Don't exit on errors by always returning "true"
done

# envoy protos dependency
cp -r ./bazel-bin/external/com_github_gogo_protobuf/*/gogo_proto_go%/gogoproto vendor/ || true # Don't exit on errors by always returning "true"
cp -r ./bazel-bin/external/opencensus_proto/opencensus/proto/*/*/*/*%/github.com/ vendor/ || true # Don't exit on errors by always returning "true"