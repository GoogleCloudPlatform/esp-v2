load("@envoy_api//bazel:api_build_system.bzl", "api_proto_library")

SERVICE_CONTROL_VEISIBILITY = [
    "//api/envoy/http/service_control:__subpackages__",
    "//src/envoy/http/service_control:__subpackages__",
]

package(default_visibility = SERVICE_CONTROL_VEISIBILITY)

api_proto_library(
    name = "requirement_proto",
    srcs = [
        "requirement.proto"
    ],
    visibility = SERVICE_CONTROL_VEISIBILITY,
)

api_proto_library(
    name = "config_proto",
    srcs = ["config.proto"],
    deps = [
        "//api/envoy/http/common:matcher_proto",
        "requirement_proto",
    ],
)
