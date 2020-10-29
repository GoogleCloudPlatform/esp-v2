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
    compilers = [
        "@io_bazel_rules_go//proto:go_proto",
        "@envoy_api//bazel:pgv_plugin_go",
    ],
    importpath = "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v9/http/grpc_metadata_scrubber",
    proto = ":config_proto",
    deps = [
        "@com_envoyproxy_protoc_gen_validate//validate:go_default_library",
        "@com_github_golang_protobuf//ptypes:go_default_library",
    ],
)
