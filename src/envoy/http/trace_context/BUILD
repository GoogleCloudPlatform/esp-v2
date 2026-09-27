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
        "//api/envoy/v12/http/trace_context:config_proto_cc_proto",
        "//src/api_proxy/tracing:trace_context_utils_lib",
        "@envoy//source/common/http:headers_lib",
        "@envoy//source/common/http:utility_lib",
        "@envoy//source/common/tracing:http_tracer_lib",
        "@envoy//source/extensions/filters/http/common:pass_through_filter_lib",
    ],
)

envoy_cc_library(
    name = "filter_factory",
    srcs = [
        "filter_factory.cc",
    ],
    repository = "@envoy",
    deps = [
        ":filter_lib",
        "//api/envoy/v12/http/trace_context:config_proto_cc_proto",
        "@envoy//source/exe:all_extensions_lib",
        "@envoy//source/extensions/filters/http/common:factory_base_lib",
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
        "//api/envoy/v12/http/trace_context:config_proto_cc_proto",
        "@envoy//test/mocks/http:http_mocks",
        "@envoy//test/mocks/server:server_mocks",
        "@envoy//test/mocks/tracing:tracing_mocks",
        "@envoy//test/test_common:utility_lib",
    ],
)
