load( "@envoy_api//bazel:api_build_system.bzl", "api_proto_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library", "go_grpc_library")

SERVICE_CONTROL_VEISIBILITY = [
    "//api/envoy/http/service_control:__subpackages__",
    "//src/envoy/http/service_control:__subpackages__",
]

package(default_visibility = SERVICE_CONTROL_VEISIBILITY)

api_proto_library(
    name = "config_proto",
    srcs = [
        "requirement.proto",
        "config.proto",
    ],
    deps = [
        "//api/envoy/http/common:pattern_proto",
    ],
)

go_proto_library(
    name = "config_proto_go_proto",
    proto = ":config_proto",
    importpath = "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/service_control",
    deps = [
        "//api/envoy/http/common:pattern_proto_go_proto",
        "@com_lyft_protoc_gen_validate//validate:go_default_library",
    ]
)
