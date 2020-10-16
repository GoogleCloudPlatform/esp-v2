load("@envoy_api//bazel:api_build_system.bzl", "api_cc_py_proto_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

package(default_visibility = ["//visibility:public"])

api_cc_py_proto_library(
    name = "config_proto",
    srcs = [
        "config.proto",
    ],
    visibility = ["//visibility:public"],
)

go_proto_library(
    name = "config_go_proto",
    importpath = "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v9/http/path_rewrite",
    proto = ":config_proto",
    deps = [
        "@com_envoyproxy_protoc_gen_validate//validate:go_default_library",
    ],
)
