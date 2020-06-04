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
    name = "filter_factory",
    srcs = ["filter_factory.cc"],
    repository = "@envoy",
    visibility = ["//src/envoy:__subpackages__"],
    deps = [
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
        "filter_config_impl.h",
    ],
    repository = "@envoy",
    deps = [
        ":config_parser_lib",
        "//api/envoy/http/backend_auth:config_proto_cc_proto",
        "//src/envoy/utils:filter_state_utils_lib",
        "@envoy//source/extensions/filters/http/common:pass_through_filter_lib",
    ],
)

envoy_cc_library(
    name = "config_parser_lib",
    srcs = ["config_parser_impl.cc"],
    hdrs = [
        "config_parser.h",
        "config_parser_impl.h",
    ],
    repository = "@envoy",
    deps = [
        "//api/envoy/http/backend_auth:config_proto_cc_proto",
        "//src/envoy/token:token_subscriber_factory_lib",
        "@envoy//source/common/common:assert_lib",
    ],
)

envoy_cc_test(
    name = "config_parser_impl_test",
    size = "small",
    srcs = [
        "config_parser_impl_test.cc",
    ],
    repository = "@envoy",
    deps = [
        ":config_parser_lib",
        ":filter_lib",
        ":mocks_lib",
        "//src/envoy/token:mocks_lib",
        "@envoy//source/common/common:empty_string",
        "@envoy//test/mocks/http:http_mocks",
        "@envoy//test/mocks/server:server_mocks",
        "@envoy//test/test_common:utility_lib",
    ],
)

envoy_cc_library(
    name = "filter_config_interface",
    hdrs = ["filter_config.h"],
    repository = "@envoy",
)

envoy_cc_library(
    name = "config_parser_interface",
    hdrs = ["config_parser.h"],
    repository = "@envoy",
    deps = [
        "@envoy//source/common/common:empty_string",
    ],
)

envoy_cc_library(
    name = "mocks_lib",
    hdrs = ["mocks.h"],
    repository = "@envoy",
    deps = [
        ":config_parser_interface",
        ":filter_config_interface",
    ],
)

envoy_cc_test(
    name = "filter_test",
    size = "small",
    srcs = [
        "filter_test.cc",
    ],
    repository = "@envoy",
    deps = [
        ":filter_lib",
        ":mocks_lib",
        "@envoy//source/common/common:empty_string",
        "@envoy//test/mocks/http:http_mocks",
        "@envoy//test/mocks/server:server_mocks",
        "@envoy//test/test_common:utility_lib",
    ],
)
