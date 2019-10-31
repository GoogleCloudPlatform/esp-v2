load("@envoy_api//bazel:api_build_system.bzl", "api_cc_py_proto_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

SERVICE_CONTROL_VISIBILITY = [
    "//api/envoy/http/service_control:__subpackages__",
    "//src/envoy/http/service_control:__subpackages__",
    "//src/go:__subpackages__",
]

package(default_visibility = SERVICE_CONTROL_VISIBILITY)

api_cc_py_proto_library(
    name = "config_proto",
    srcs = [
        "config.proto",
        "requirement.proto",
    ],
    visibility = SERVICE_CONTROL_VISIBILITY,
    deps = [
        "//api/envoy/http/common:base_proto",
    ],
)

go_proto_library(
    name = "config_go_proto",
    importpath = "github.com/GoogleCloudPlatform/api-proxy/src/go/proto/api/envoy/http/service_control",
    proto = ":config_proto",
    deps = [
        "//api/envoy/http/common:base_go_proto",
        "@com_envoyproxy_protoc_gen_validate//validate:go_default_library",
    ],
)
