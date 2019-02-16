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
    name = "filter_lib",
    srcs = [
        "filter.cc",
        "handler.cc",
    ],
    hdrs = [
        "filter.h",
        "filter_config.h",
        "handler.h",
    ],
    repository = "@envoy",
    deps = [
        ":config_parser_lib",
        "@envoy//source/exe:envoy_common_lib",
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
    visibility = ["//src/envoy:__subpackages__"]
)

envoy_cc_library(
    name = "http_call_lib",
    srcs = ["http_call.cc"],
    hdrs = ["http_call.h"],
    repository = "@envoy",
    deps = [
        "//api/envoy/http/service_control:config_proto_cc",
        "@envoy//include/envoy/upstream:cluster_manager_interface",
        "@envoy//source/common/common:enum_to_int",
        "@envoy//source/common/http:headers_lib",
        "@envoy//source/common/http:message_lib",
        "@envoy//source/common/http:utility_lib",
    ],
)

envoy_cc_library(
    name = "client_cache_lib",
    srcs = ["client_cache.cc"],
    hdrs = ["client_cache.h"],
    repository = "@envoy",
    deps = [
        ":http_call_lib",
        "//api/envoy/http/service_control:config_proto_cc",
        "//external:servicecontrol_client",
        "//src/api_proxy/service_control:request_builder_lib",
        "@envoy//include/envoy/event:dispatcher_interface",
        "@envoy//include/envoy/upstream:cluster_manager_interface",
    ],
)

envoy_cc_library(
    name = "config_parser_lib",
    srcs = ["config_parser.cc"],
    hdrs = ["config_parser.h"],
    repository = "@envoy",
    deps = [
        ":client_cache_lib",
        "//api/envoy/http/service_control:config_proto_cc",
        "//src/api_proxy/path_matcher:path_matcher_lib",
        "//src/api_proxy/service_control:request_builder_lib",
        "//src/api_proxy/service_control:logs_metrics_loader_lib",
        "//src/envoy/utils:token_subscriber_lib",
        "@envoy//source/common/protobuf:utility_lib",
    ],
)

envoy_cc_test(
    name = "config_parser_test",
    srcs = ["config_parser_test.cc"],
    repository = "@envoy",
    deps = [
        ":config_parser_lib",
        "@envoy//test/test_common:utility_lib",
        "@envoy//test/mocks/server:server_mocks",
    ],
)
