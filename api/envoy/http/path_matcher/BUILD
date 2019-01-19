load( "@envoy_api//bazel:api_build_system.bzl", "api_proto_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library", "go_grpc_library")

PATH_MATCHER_VISIBILITY = [
    "//api/envoy/http/path_matcher:__subpackages__",
    "//src/envoy/http/path_matcher:__subpackages__",
]

package(default_visibility = PATH_MATCHER_VISIBILITY)

api_proto_library(
    name = "config_proto",
    srcs = [
        "config.proto",
    ],
    deps = [
        "//api/envoy/http/common:pattern_proto",
    ],
    visibility = PATH_MATCHER_VISIBILITY,
)

go_proto_library(
    name = "config_proto_go_proto",
    proto = ":config_proto",
    importpath = "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/path_matcher",
    deps = [
        "//api/envoy/http/common:pattern_proto_go_proto",
        "@com_lyft_protoc_gen_validate//validate:go_default_library",
    ]
)
