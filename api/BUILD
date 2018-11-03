package(default_visibility = ["//visibility:public"])

load("@envoy_api//bazel:api_build_system.bzl", "api_proto_library")

api_proto_library(
    name = "service_control_filter_config_proto",
    srcs = ["service_control_filter_config.proto"],
    deps = [
        ":matcher",
    ]
)

api_proto_library(
    name = "matcher",
    srcs = ["matcher.proto"],
)
