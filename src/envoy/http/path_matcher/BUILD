load(
    "@envoy//bazel:envoy_build_system.bzl",
    "envoy_cc_fuzz_test",
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
    deps = [
        ":filter_lib",
        "@envoy//source/exe:envoy_common_lib",
    ],
)

envoy_cc_library(
    name = "filter_lib",
    srcs = [
        "filter.cc",
    ],
    hdrs = [
        "filter.h",
    ],
    repository = "@envoy",
    deps = [
        ":filter_config_lib",
        "//src/envoy/utils:filter_state_utils_lib",
        "//src/envoy/utils:http_header_utils_lib",
        "@envoy//source/common/protobuf:utility_lib",
        "@envoy//source/exe:envoy_common_lib",
        "@envoy//source/extensions/filters/http/common:pass_through_filter_lib",
    ],
)

envoy_cc_library(
    name = "filter_config_lib",
    srcs = ["filter_config.cc"],
    hdrs = ["filter_config.h"],
    repository = "@envoy",
    deps = [
        "//api/envoy/v9/http/path_matcher:config_proto_cc_proto",
        "//src/api_proxy/path_matcher:path_matcher_lib",
        "//src/api_proxy/path_matcher:variable_binding_utils_lib",
        "@envoy//source/common/common:empty_string",
        "@envoy//source/exe:envoy_common_lib",
    ],
)

envoy_cc_test(
    name = "filter_config_test",
    srcs = [
        "filter_config_test.cc",
    ],
    repository = "@envoy",
    deps = [
        ":filter_config_lib",
        "@envoy//source/common/common:empty_string",
        "@envoy//test/mocks/server:server_mocks",
        "@envoy//test/test_common:utility_lib",
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
        "@envoy//source/common/common:empty_string",
        "@envoy//test/mocks/server:server_mocks",
        "@envoy//test/test_common:utility_lib",
    ],
)

envoy_cc_fuzz_test(
    name = "path_matcher_filter_fuzz_test",
    srcs = ["filter_fuzz_test.cc"],
    corpus = "//tests/fuzz/corpus:path_matcher_filter_corpus",
    repository = "@envoy",
    deps = [
        ":filter_lib",
        "//src/envoy/utils:filter_state_utils_lib",
        "//tests/fuzz/structured_inputs:path_matcher_filter_proto_cc_proto",
        "@envoy//test/extensions/filters/http/common/fuzz:uber_filter_lib",
        "@envoy//test/fuzz:utility_lib",
        "@envoy//test/mocks/init:init_mocks",
        "@envoy//test/mocks/server:server_mocks",
        "@envoy//test/test_common:utility_lib",
    ],
)
