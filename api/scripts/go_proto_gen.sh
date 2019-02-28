set -e

#TODO(bochun): probably we can programatically generate these.
# Agent proto
bazel build //api/agent:agent_service_go_grpc
mkdir -p src/go/proto/api/agent
cp -f bazel-bin/api/agent/*/agent_service_go_grpc%/cloudesf.googlesource.com/gcpproxy/src/go/proto/api/agent/* src/go/proto/api/agent
# HTTP filter common
bazel build //api/envoy/http/common:pattern_proto_go_proto
mkdir -p src/go/proto/api/envoy/http/common
cp -f bazel-bin/api/envoy/http/common/*/pattern_proto_go_proto%/cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/common/* src/go/proto/api/envoy/http/common
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
