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
        "//api/envoy/v8/http/grpc_metadata_scrubber:config_proto_cc_proto",
        "@envoy//source/common/grpc:common_lib",
        "@envoy//source/common/http:headers_lib",
        "@envoy//source/extensions/filters/http/common:pass_through_filter_lib",
    ],
)
