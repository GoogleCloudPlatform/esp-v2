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
    name = "token_subscriber_lib",
    srcs = ["token_subscriber.cc"],
    hdrs = ["token_subscriber.h"],
    repository = "@envoy",
    deps = [
        "//api/agent:agent_proto_cc",
        "@envoy//include/envoy/common:time_interface",
        "@envoy//include/envoy/event:dispatcher_interface",
        "@envoy//include/envoy/server:filter_config_interface",
        "@envoy//include/envoy/upstream:cluster_manager_interface",
        "@envoy//source/common/grpc:async_client_lib",
        "@envoy//source/common/grpc:async_client_manager_lib",
        "@envoy//source/common/init:target_lib",
    ],
)

envoy_cc_test(
    name = "token_subscriber_test",
    srcs = ["token_subscriber_test.cc"],
    repository = "@envoy",
    deps = [
        ":token_subscriber_lib",
        "@envoy//source/common/tracing:http_tracer_lib",
        "@envoy//test/mocks/init:init_mocks",
        "@envoy//test/mocks/server:server_mocks",
        "@envoy//test/test_common:utility_lib",
    ],
)

envoy_cc_library(
    name = "filter_state_utils_lib",
    srcs = [
        "filter_state_utils.cc",
    ],
    hdrs = [
        "filter_state_utils.h",
    ],
    repository = "@envoy",
    deps = [
        "@envoy//source/common/router:string_accessor_lib",
        "@envoy//source/exe:envoy_common_lib",
    ],
)

envoy_cc_test(
    name = "filter_state_utils_test",
    srcs = ["filter_state_utils_test.cc"],
    repository = "@envoy",
    deps = [
        ":filter_state_utils_lib",
        "@envoy//source/common/stream_info:filter_state_lib",
        "@envoy//test/test_common:utility_lib",
    ],
)

envoy_cc_library(
    name = "http_header_utils_lib",
    srcs = ["http_header_utils.cc"],
    hdrs = ["http_header_utils.h"],
    repository = "@envoy",
    deps = [
        "@envoy//source/exe:envoy_common_lib",
    ],
)

envoy_cc_library(
    name = "message_counter_lib",
    srcs = ["message_counter.cc"],
    hdrs = ["message_counter.h"],
    repository = "@envoy",
    deps = [
        "@envoy//source/common/buffer:buffer_lib",
        "@envoy//source/exe:envoy_common_lib",
    ],
)

envoy_cc_test(
    name = "message_counter_test",
    srcs = [
        "message_counter_test.cc",
    ],
    repository = "@envoy",
    deps = [
        ":message_counter_lib",
        "@envoy//source/common/buffer:buffer_lib",
        "@envoy//test/common/buffer:utility_lib",
        "@envoy//test/test_common:utility_lib",
    ],
)
