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
    name = "service_control_callback_func_lib",
    hdrs = ["service_control_callback_func.h"],
    repository = "@envoy",
    deps = [
        "//src/api_proxy/service_control:request_builder_lib",
    ],
)

envoy_cc_library(
    name = "service_control_call_interface",
    hdrs = ["service_control_call.h"],
    repository = "@envoy",
    deps = [
        ":service_control_callback_func_lib",
        "//api/envoy/http/service_control:config_proto_cc_proto",
        "@envoy//include/envoy/tracing:http_tracer_interface",
    ],
)

envoy_cc_library(
    name = "handler_interface",
    hdrs = ["handler.h"],
    repository = "@envoy",
    deps = [
        ":filter_stats_lib",
        ":service_control_callback_func_lib",
        "@envoy//include/envoy/http:header_map_interface",
        "@envoy//include/envoy/stream_info:stream_info_interface",
    ],
)

envoy_cc_library(
    name = "config_parser_lib",
    srcs = ["config_parser.cc"],
    hdrs = ["config_parser.h"],
    repository = "@envoy",
    deps = [
        ":service_control_call_interface",
        "@envoy//source/common/protobuf:utility_lib",
    ],
)

envoy_cc_library(
    name = "http_call_lib",
    srcs = ["http_call.cc"],
    hdrs = ["http_call.h"],
    repository = "@envoy",
    deps = [
        "//api/envoy/http/common:base_proto_cc_proto",
        "@envoy//include/envoy/event:deferred_deletable",
        "@envoy//include/envoy/upstream:cluster_manager_interface",
        "@envoy//source/common/common:empty_string",
        "@envoy//source/common/common:enum_to_int",
        "@envoy//source/common/http:headers_lib",
        "@envoy//source/common/http:message_lib",
        "@envoy//source/common/http:utility_lib",
        "@envoy//source/common/tracing:http_tracer_lib",
    ],
)

envoy_cc_library(
    name = "client_cache_lib",
    srcs = ["client_cache.cc"],
    hdrs = [
        "client_cache.h",
    ],
    repository = "@envoy",
    deps = [
        "filter_stats_lib",
        ":http_call_lib",
        ":service_control_callback_func_lib",
        "//api/envoy/http/common:base_proto_cc_proto",
        "//api/envoy/http/service_control:config_proto_cc_proto",
        "@envoy//include/envoy/event:dispatcher_interface",
        "@envoy//include/envoy/upstream:cluster_manager_interface",
        "@envoy//source/common/tracing:http_tracer_lib",
        "@servicecontrol_client_git//:service_control_client_lib",
    ],
)

envoy_cc_test(
    name = "client_cache_test",
    srcs = [
        "client_cache_test.cc",
    ],
    repository = "@envoy",
    deps = [
        ":client_cache_lib",
        ":mocks_lib",
        ":service_control_callback_func_lib",
        "@envoy//source/common/common:empty_string",
        "@envoy//test/mocks/event:event_mocks",
        "@envoy//test/mocks/server:server_mocks",
        "@envoy//test/mocks/stats:stats_mocks",
        "@envoy//test/test_common:utility_lib",
    ],
)

envoy_cc_library(
    name = "service_control_call_impl_lib",
    srcs = ["service_control_call_impl.cc"],
    hdrs = ["service_control_call_impl.h"],
    repository = "@envoy",
    deps = [
        ":client_cache_lib",
        ":service_control_call_interface",
        "//src/api_proxy/service_control:logs_metrics_loader_lib",
        "//src/envoy/token:token_subscriber_factory_lib",
        "@envoy//include/envoy/server:filter_config_interface",
        "@envoy//source/common/common:empty_string",
        "@envoy//source/common/protobuf:utility_lib",
    ],
)

envoy_cc_library(
    name = "handler_impl_lib",
    srcs = [
        "handler_impl.cc",
        "handler_utils.cc",
    ],
    hdrs = [
        "handler_impl.h",
        "handler_utils.h",
    ],
    repository = "@envoy",
    deps = [
        ":config_parser_lib",
        ":handler_interface",
        "//src/envoy/utils:filter_state_utils_lib",
        "//src/envoy/utils:http_header_utils_lib",
        "@envoy//source/common/config:metadata_lib",
        "@envoy//source/common/grpc:common_lib",
        "@envoy//source/common/http:headers_lib",
        "@envoy//source/extensions/filters/http:well_known_names",
        "@envoy//source/extensions/filters/http/grpc_stats:config",
    ],
)

envoy_cc_library(
    name = "filter_stats_lib",
    srcs = [
        "filter_stats.cc",
    ],
    hdrs = [
        "filter_stats.h",
    ],
    repository = "@envoy",
    deps = [
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
        ":filter_stats_lib",
        ":handler_interface",
        "@envoy//source/common/grpc:status_lib",
        "@envoy//source/common/http:headers_lib",
        "@envoy//source/exe:envoy_common_lib",
        "@envoy//source/extensions/filters/http/common:pass_through_filter_lib",
    ],
)

envoy_cc_library(
    name = "filter_config_lib",
    hdrs = [
        "filter_config.h",
    ],
    repository = "@envoy",
    deps = [
        ":filter_stats_lib",
        ":handler_impl_lib",
        ":service_control_call_impl_lib",
        "@envoy//source/exe:envoy_common_lib",
    ],
)

envoy_cc_library(
    name = "filter_factory",
    srcs = ["filter_factory.cc"],
    repository = "@envoy",
    visibility = ["//src/envoy:__subpackages__"],
    deps = [
        ":filter_config_lib",
        ":filter_lib",
        "@envoy//source/exe:envoy_common_lib",
    ],
)

envoy_cc_library(
    name = "mocks_lib",
    hdrs = ["mocks.h"],
    repository = "@envoy",
    deps = [
        ":handler_interface",
        ":service_control_call_interface",
    ],
)

envoy_cc_test(
    name = "config_parser_test",
    srcs = [
        "config_parser_test.cc",
    ],
    repository = "@envoy",
    deps = [
        ":config_parser_lib",
        ":mocks_lib",
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
        ":mocks_lib",
        "@envoy//source/common/common:empty_string",
        "@envoy//test/mocks/server:server_mocks",
        "@envoy//test/mocks/stats:stats_mocks",
        "@envoy//test/mocks/tracing:tracing_mocks",
        "@envoy//test/test_common:utility_lib",
    ],
)

envoy_cc_test(
    name = "filter_stats_test",
    srcs = [
        "filter_stats_test.cc",
    ],
    repository = "@envoy",
    deps = [
        ":filter_stats_lib",
        "@envoy//test/mocks/server:server_mocks",
        "@envoy//test/test_common:utility_lib",
    ],
)

envoy_cc_test(
    name = "handler_test",
    srcs = [
        "handler_impl_test.cc",
        "handler_utils_test.cc",
    ],
    repository = "@envoy",
    deps = [
        ":config_parser_lib",
        ":handler_impl_lib",
        ":mocks_lib",
        "@envoy//source/common/common:empty_string",
        "@envoy//test/mocks/server:server_mocks",
        "@envoy//test/mocks/stats:stats_mocks",
        "@envoy//test/mocks/tracing:tracing_mocks",
        "@envoy//test/test_common:simulated_time_system_lib",
        "@envoy//test/test_common:utility_lib",
    ],
)

envoy_cc_test(
    name = "http_call_test",
    srcs = [
        "http_call_test.cc",
    ],
    repository = "@envoy",
    deps = [
        ":http_call_lib",
        ":mocks_lib",
        "@envoy//test/mocks:common_lib",
        "@envoy//test/mocks/event:event_mocks",
        "@envoy//test/mocks/server:server_mocks",
        "@envoy//test/mocks/stats:stats_mocks",
        "@envoy//test/mocks/tracing:tracing_mocks",
        "@envoy//test/test_common:utility_lib",
    ],
)

envoy_cc_fuzz_test(
    name = "service_control_filter_fuzz_test",
    srcs = ["filter_fuzz_test.cc"],
    corpus = "//tests/fuzz/corpus:service_control_filter_corpus",
    repository = "@envoy",
    deps = [
        ":filter_config_lib",
        ":filter_lib",
        "//src/envoy/utils:filter_state_utils_lib",
        "//tests/fuzz/structured_inputs:service_control_filter_proto_cc_proto",
        "@envoy//test/extensions/filters/http/common/fuzz:uber_filter_lib",
        "@envoy//test/fuzz:utility_lib",
        "@envoy//test/mocks/init:init_mocks",
        "@envoy//test/mocks/server:server_mocks",
        "@envoy//test/test_common:utility_lib",
    ],
)
