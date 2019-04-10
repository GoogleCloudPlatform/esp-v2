load("@envoy_api//bazel:api_build_system.bzl", "api_proto_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_grpc_library")

package(default_visibility = ["//visibility:public"])

api_proto_library(
    name = "agent_proto",
    srcs = ["agent_service.proto"],
    has_services = 1,
)

go_grpc_library(
    name = "agent_service_go_grpc",
    importpath = "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/agent",
    proto = ":agent_proto",
    deps = [
        "@com_github_gogo_protobuf//:gogo_proto_go",
        "@com_github_golang_protobuf//ptypes/duration:go_default_library",
    ],
)
