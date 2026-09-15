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
    name = "early_header_mutation_lib",
    srcs = ["early_header_mutation.cc"],
    hdrs = ["early_header_mutation.h"],
    repository = "@envoy",
    deps = [
        "//api/envoy/v12/http/trace_context:config_proto_cc_proto",
        "//src/api_proxy/tracing:trace_context_utils_lib",
        "@envoy//envoy/http:early_header_mutation_interface",
        "@envoy//envoy/stream_info:stream_info_interface",
        "@envoy//source/common/common:logger_lib",
        "@envoy//source/common/singleton:const_singleton",
    ],
)

envoy_cc_library(
    name = "early_header_mutation_factory",
    srcs = ["early_header_mutation_factory.cc"],
    repository = "@envoy",
    deps = [
        ":early_header_mutation_lib",
        "//api/envoy/v12/http/trace_context:config_proto_cc_proto",
        "@envoy//envoy/http:early_header_mutation_interface",
        "@envoy//envoy/registry",
        "@envoy//source/common/config:utility_lib",
    ],
)

envoy_cc_test(
    name = "early_header_mutation_test",
    srcs = ["early_header_mutation_test.cc"],
    repository = "@envoy",
    deps = [
        ":early_header_mutation_factory",
        ":early_header_mutation_lib",
        "@envoy//test/mocks/http:http_mocks",
        "@envoy//test/mocks/server:factory_context_mocks",
        "@envoy//test/mocks/stream_info:stream_info_mocks",
        "@envoy//test/test_common:utility_lib",
    ],
)
