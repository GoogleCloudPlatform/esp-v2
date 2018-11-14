load("@envoy_api//bazel:api_build_system.bzl", "api_proto_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library", "go_grpc_library")

DEFAULT_VISIBILITY = [
    "//api/envoy/http:__subpackages__",
    "//src/envoy/http:__subpackages__",
]

package(default_visibility = DEFAULT_VISIBILITY)

api_proto_library(
    name = "pattern_proto",
    srcs = [
        "pattern.proto",
    ],
    visibility = DEFAULT_VISIBILITY,
)

go_grpc_library(
    name = "pattern_proto_go_proto",
    proto = ":pattern_proto",
    importpath = "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/common",
)
