package(default_visibility = ["//visibility:public"])

load("@envoy_api//bazel:api_build_system.bzl", "api_proto_library")

api_proto_library(
    name = "requirement_proto",
    srcs = [
        "requirement.proto"
    ],
    visibility = ["//visibility:public"],
)

api_proto_library(
    name = "config_proto",
    srcs = ["config.proto"],
    deps = [
        "//api/envoy/http/common:matcher_proto",
        "requirement_proto",
    ],
)