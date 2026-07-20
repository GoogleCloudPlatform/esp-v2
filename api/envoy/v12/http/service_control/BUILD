load("@envoy_api//bazel:api_build_system.bzl", "api_cc_py_proto_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

package(default_visibility = ["//visibility:public"])

api_cc_py_proto_library(
    name = "config_proto",
    srcs = [
        "config.proto",
        "requirement.proto",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//api/envoy/v12/http/common:base_proto",
        "//api/envoy/v12/http/service_control:googleapis_service_proto",
    ],
)

alias(
    name = "googleapis_service_proto",
    actual = "@com_github_googleapis_googleapis//google/api:service_proto",
)

alias(
    name = "googleapis_service_proto_cc_proto",
    actual = "@com_github_googleapis_googleapis//google/api:service_cc_proto",
)

go_proto_library(
    name = "config_go_proto",
    importpath = "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/service_control",
    proto = ":config_proto",
    deps = [
        "//api/envoy/v12/http/common:base_go_proto",
        "@com_envoyproxy_protoc_gen_validate//validate:go_default_library",
        "@com_github_googleapis_googleapis//google/api:serviceconfig_go_proto",
    ],
)
