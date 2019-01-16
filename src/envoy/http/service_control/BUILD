load(
    "@envoy//bazel:envoy_build_system.bzl",
    "envoy_cc_library",
    "envoy_cc_test",
)

package(
    default_visibility = [
        "//src/envoy:__subpackages__",
    ]
)

envoy_cc_library(
    name = "filter_lib",
    srcs = [
        "filter.cc",
        "http_call.cc",
    ],
    hdrs = [
        "filter.h",
        "filter_config.h",
        "http_call.h",
    ],
    repository = "@envoy",
    deps = [
        ":config_parser_lib",
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
    visibility = ["//src/envoy:__subpackages__"]
)

envoy_cc_library(
    name = "config_parser_lib",
    srcs = ["config_parser.cc"],
    hdrs = ["config_parser.h"],
    repository = "@envoy",
    deps = [
        "//api/envoy/http/service_control:config_proto_cc",
        "//src/api_proxy/path_matcher:path_matcher_lib",
        "//src/api_proxy/service_control:request_builder_lib",
        "//src/envoy/utils:token_subscriber_lib",
        "@envoy//source/common/protobuf:utility_lib",
    ],
)

envoy_cc_test(
    name = "config_parser_test",
    srcs = ["config_parser_test.cc"],
    repository = "@envoy",
    deps = [
        ":config_parser_lib",
        "@envoy//test/test_common:utility_lib",
        "@envoy//test/mocks/server:server_mocks",
    ],
)
