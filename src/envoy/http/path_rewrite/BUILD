load(
    "@envoy//bazel:envoy_build_system.bzl",
    "envoy_cc_library",
    "envoy_cc_test",
)

package(
    default_visibility = [
        "//src/envoy:__subpackages__",
    ],
)

envoy_cc_library(
    name = "config_parser_interface",
    hdrs = ["config_parser.h"],
    repository = "@envoy",
)

envoy_cc_library(
    name = "config_parser_lib",
    srcs = ["config_parser_impl.cc"],
    hdrs = [
        "config_parser_impl.h",
    ],
    repository = "@envoy",
    deps = [
        ":config_parser_interface",
        "//api/envoy/v9/http/path_rewrite:config_proto_cc_proto",
        "//src/api_proxy/path_matcher:path_matcher_lib",
        "//src/api_proxy/path_matcher:variable_binding_utils_lib",
        "@envoy//source/common/common:empty_string",
        "@envoy//source/common/common:logger_lib",
    ],
)

envoy_cc_test(
    name = "config_parser_impl_test",
    srcs = [
        "config_parser_impl_test.cc",
    ],
    repository = "@envoy",
    deps = [
        ":config_parser_lib",
        "@envoy//test/test_common:utility_lib",
    ],
)

envoy_cc_library(
    name = "filter_factory",
    srcs = ["filter_factory.cc"],
    repository = "@envoy",
    visibility = ["//src/envoy:__subpackages__"],
    deps = [
        ":config_parser_lib",
        ":filter_lib",
    ],
)

envoy_cc_library(
    name = "filter_lib",
    srcs = [
        "filter.cc",
    ],
    hdrs = [
        "filter.h",
        "filter_config.h",
    ],
    repository = "@envoy",
    deps = [
        ":config_parser_interface",
        "//src/envoy/utils:http_header_utils_lib",
        "//src/envoy/utils:rc_detail_utils_lib",
        "@envoy//include/envoy/stats:stats_interface",
        "@envoy//source/extensions/filters/http/common:pass_through_filter_lib",
    ],
)

envoy_cc_library(
    name = "mocks_lib",
    hdrs = ["mocks.h"],
    repository = "@envoy",
    deps = [
        ":config_parser_interface",
    ],
)

envoy_cc_test(
    name = "filter_test",
    srcs = [
        "filter_test.cc",
    ],
    repository = "@envoy",
    deps = [
        ":filter_lib",
        ":mocks_lib",
        "@envoy//test/mocks/server:server_mocks",
        "@envoy//test/test_common:utility_lib",
    ],
)
