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
    name = "config_parser_lib",
    srcs = ["config_parser_impl.cc"],
    hdrs = [
        "config_parser.h",
        "config_parser_impl.h",
    ],
    repository = "@envoy",
    deps = [
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
