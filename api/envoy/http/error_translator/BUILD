load("@envoy_api//bazel:api_build_system.bzl", "api_cc_py_proto_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

ERROR_TRANSLATOR_VISIBILITY = [
    "//api/envoy/http/error_translator:__subpackages__",
    "//src/envoy/http/error_translator:__subpackages__",
    "//src/go:__subpackages__",
    "//tests/utils:__subpackages__",
    "//tests/fuzz/structured_inputs:__subpackages__",
]

package(default_visibility = ERROR_TRANSLATOR_VISIBILITY)

api_cc_py_proto_library(
    name = "config_proto",
    srcs = [
        "config.proto",
    ],
    visibility = ERROR_TRANSLATOR_VISIBILITY,
)

go_proto_library(
    name = "config_go_proto",
    importpath = "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/error_translator",
    proto = ":config_proto",
    deps = [
        "@com_envoyproxy_protoc_gen_validate//validate:go_default_library",
    ],
)
