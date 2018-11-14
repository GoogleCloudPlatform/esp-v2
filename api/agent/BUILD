package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library", "go_grpc_library")

proto_library(
  name = "agent_service",
  srcs = ["agent_service.proto"],
  deps = [
    "@com_google_protobuf//:duration_proto",
    "@com_google_protobuf//:empty_proto",
  ]
)

go_grpc_library(
  name = "agent_service_go_grpc",
  proto = ":agent_service",
  importpath = "cloudesf.googlesource.com/gcpproxy/src/go/proto/agent",
  deps = [
    "@com_github_gogo_protobuf//:gogo_proto_go",
    "@com_github_golang_protobuf//ptypes/duration:go_default_library",
  ]
)
