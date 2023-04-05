load("@envoy_api//bazel:api_build_system.bzl", "api_cc_py_proto_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

package(default_visibility = ["//visibility:public"])

api_cc_py_proto_library(
    name = "base_proto",
    srcs = [
        "base.proto",
    ],
    visibility = ["//visibility:public"],
)

go_proto_library(
    name = "base_go_proto",
    importpath = "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/common",
    proto = ":base_proto",
    deps = [
        "@com_envoyproxy_protoc_gen_validate//validate:go_default_library",
    ],
)
