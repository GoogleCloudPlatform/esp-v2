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
    name = "filter_lib",
    srcs = [
        "filter.cc",
    ],
    hdrs = [
        "filter.h",
    ],
    repository = "@envoy",
    deps = [
        "//src/envoy/utils:http_header_utils_lib",
        "//src/envoy/utils:rc_detail_utils_lib",
        "@envoy//envoy/stats:stats_interface",
        "@envoy//source/extensions/filters/http/common:pass_through_filter_lib",
    ],
)

envoy_cc_library(
    name = "filter_factory",
    srcs = ["filter_factory.cc"],
    repository = "@envoy",
    deps = [
        ":filter_lib",
        "//api/envoy/v11/http/header_sanitizer:config_proto_cc_proto",
        "@envoy//source/exe:envoy_common_lib",
    ],
)
