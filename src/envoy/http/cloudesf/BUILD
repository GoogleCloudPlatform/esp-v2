
package(default_visibility = ["//visibility:public"])

load(
    "@envoy//bazel:envoy_build_system.bzl",
    "envoy_cc_binary",
    "envoy_cc_library",
    "envoy_cc_test",
)

load(
    "@envoy_api//bazel:api_build_system.bzl",
    "api_proto_library",
)

api_proto_library(
    name = "config_proto",
    srcs = ["config.proto"],
    visibility = ["//visibility:public"],
    require_py = 0,
)

envoy_cc_library(
    name = "filter_lib",
    srcs = [
        "filter.cc",
        "http_call.cc",
        "token_fetcher.cc",
    ],
    hdrs = [
        "filter.h",
        "filter_config.h",
        "http_call.h",
        "token_fetcher.h",
    ],
    repository = "@envoy",
    deps = [
        ":config_proto_cc",
        "//src/envoy/http/cloudesf/service_control:proto",
        "@envoy//source/exe:envoy_common_lib",
    ],
)

envoy_cc_library(
    name = "filter_factory",
    srcs = ["filter_factory.cc"],
    repository = "@envoy",
    deps = [
        ":filter_lib",
        "@envoy//source/exe:envoy_common_lib",
    ],
)

