
package(default_visibility = ["//visibility:public"])

load(
    "@envoy//bazel:envoy_build_system.bzl",
    "envoy_cc_library",
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
        "//api/envoy/http/service_control:config_proto_cc",
        "//src/api_proxy/service_control:request_builder_lib",
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

