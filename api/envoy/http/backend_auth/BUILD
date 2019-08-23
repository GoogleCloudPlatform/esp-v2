load("@envoy_api//bazel:api_build_system.bzl", "api_proto_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

BACKEND_AUTH_VISIBILITY = [
    "//api/envoy/http/backend_auth:__subpackages__",
    "//src/envoy/http/backend_auth:__subpackages__",
    "//src/go:__subpackages__",
]

package(default_visibility = BACKEND_AUTH_VISIBILITY)

api_proto_library(
    name = "config_proto",
    srcs = [
        "config.proto",
    ],
    visibility = BACKEND_AUTH_VISIBILITY,
    deps = [
        "//api/envoy/http/common:base_proto",
    ],
)

go_proto_library(
    name = "config_go_proto",
    importpath = "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/backend_auth",
    proto = ":config_proto",
    deps = [
        "//api/envoy/http/common:base_go_proto",
        "@com_envoyproxy_protoc_gen_validate//validate:go_default_library",
    ],
)
