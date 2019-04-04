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
    name = "filter_factory",
    srcs = ["filter_factory.cc"],
    repository = "@envoy",
    deps = [
        ":filter_lib",
    ],
    visibility = ["//src/envoy:__subpackages__"]
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
        ":config_parser_lib",
        "//api/envoy/http/backend_auth:config_proto_cc",
        "//src/envoy/utils:filter_state_utils_lib",
    ],
)

envoy_cc_library(
    name = "config_parser_lib",
    srcs = ["config_parser.cc"],
    hdrs = ["config_parser.h"],
    repository = "@envoy",
    deps = [
        "//api/envoy/http/backend_auth:config_proto_cc",
        "//src/envoy/utils:token_subscriber_lib",
    ],
)