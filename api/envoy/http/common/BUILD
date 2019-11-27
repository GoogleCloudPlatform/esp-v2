load("@envoy_api//bazel:api_build_system.bzl", "api_cc_py_proto_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

DEFAULT_VISIBILITY = [
    "//api/envoy/http:__subpackages__",
    "//src/envoy/http:__subpackages__",
    "//src/go:__subpackages__",
    "//tests/utils:__subpackages__",
]

package(default_visibility = DEFAULT_VISIBILITY)

api_cc_py_proto_library(
    name = "base_proto",
    srcs = [
        "base.proto",
    ],
    visibility = DEFAULT_VISIBILITY,
)

go_proto_library(
    name = "base_go_proto",
    importpath = "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/common",
    proto = ":base_proto",
    deps = [
        "@com_envoyproxy_protoc_gen_validate//validate:go_default_library",
    ],
)
