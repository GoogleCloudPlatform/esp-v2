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
    name = "token_subscriber_lib",
    srcs = ["token_subscriber.cc"],
    hdrs = ["token_subscriber.h"],
    repository = "@envoy",
    deps = [
        "//api/agent:agent_proto_cc",
        "@envoy//include/envoy/common:time_interface",
        "@envoy//include/envoy/event:dispatcher_interface",
        "@envoy//include/envoy/init:init_interface",
        "@envoy//include/envoy/server:filter_config_interface",
        "@envoy//include/envoy/upstream:cluster_manager_interface",
        "@envoy//source/common/grpc:async_client_lib",
        "@envoy//source/common/grpc:async_client_manager_lib",
    ],
)

envoy_cc_test(
    name = "token_subscriber_test",
    srcs = ["token_subscriber_test.cc"],
    repository = "@envoy",
    deps = [
        ":token_subscriber_lib",
        "@envoy//source/common/tracing:http_tracer_lib",
        "@envoy//test/test_common:utility_lib",
        "@envoy//test/mocks/server:server_mocks",
    ],
)

envoy_cc_library(
    name = "filter_state_utils_lib",
    hdrs = [
        "filter_state_utils.h",
    ],
    srcs = [
        "filter_state_utils.cc",
    ],
    repository = "@envoy",
    deps = [
        "@envoy//source/common/router:string_accessor_lib",
        "@envoy//source/exe:envoy_common_lib",
    ]
)

envoy_cc_test(
    name = "filter_state_utils_test",
    srcs = ["filter_state_utils_test.cc"],
    repository = "@envoy",
    deps = [
        ":filter_state_utils_lib",
        "@envoy//test/test_common:utility_lib",
        "@envoy//source/common/stream_info:filter_state_lib",
    ],
)
envoy_cc_library(
    name = "status_http_code_lib",
    hdrs = ["status_http_code.h"],
    srcs = ["status_http_code.cc"],
    repository = "@envoy",
    deps = [
        "//external:protobuf",
        "@envoy//include/envoy/http:codes_interface",
    ]
)

envoy_cc_library(
    name = "http_header_utils_lib",
    hdrs = ["http_header_utils.h"],
    srcs = ["http_header_utils.cc"],
    repository = "@envoy",
    deps = [
        "@envoy//source/exe:envoy_common_lib",
    ]
)
