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
    name = "json_struct_lib",
    srcs = ["json_struct.cc"],
    hdrs = ["json_struct.h"],
    repository = "@envoy",
    deps = [
        "@envoy//source/common/buffer:buffer_lib",
        "@envoy//source/common/grpc:status_lib",
        "@envoy//source/common/protobuf:utility_lib",
    ],
)

envoy_cc_test(
    name = "json_struct_test",
    size = "small",
    srcs = [
        "json_struct_test.cc",
    ],
    repository = "@envoy",
    deps = [
        ":json_struct_lib",
        "@envoy//test/test_common:utility_lib",
    ],
)

envoy_cc_library(
    name = "token_subscriber_lib",
    srcs = ["token_subscriber.cc"],
    hdrs = ["token_subscriber.h"],
    repository = "@envoy",
    deps = [
        ":json_struct_lib",
        "//api/envoy/http/common:base_proto_cc",
        "//src/api_proxy/auth:lib",
        "@envoy//include/envoy/common:time_interface",
        "@envoy//include/envoy/event:dispatcher_interface",
        "@envoy//include/envoy/server:filter_config_interface",
        "@envoy//include/envoy/upstream:cluster_manager_interface",
        "@envoy//source/common/common:enum_to_int",
        "@envoy//source/common/http:headers_lib",
        "@envoy//source/common/http:message_lib",
        "@envoy//source/common/http:utility_lib",
        "@envoy//source/common/init:target_lib",
    ],
)

envoy_cc_library(
    name = "iam_token_subscriber_lib",
    srcs = ["iam_token_subscriber.cc"],
    hdrs = ["iam_token_subscriber.h"],
    repository = "@envoy",
    deps = [
        ":json_struct_lib",
        "//api/envoy/http/common:base_proto_cc",
        "//src/api_proxy/auth:lib",
        "@envoy//include/envoy/common:time_interface",
        "@envoy//include/envoy/event:dispatcher_interface",
        "@envoy//include/envoy/server:filter_config_interface",
        "@envoy//include/envoy/upstream:cluster_manager_interface",
        "@envoy//source/common/common:enum_to_int",
        "@envoy//source/common/http:headers_lib",
        "@envoy//source/common/http:message_lib",
        "@envoy//source/common/http:utility_lib",
        "@envoy//source/common/init:target_lib",
    ],
)

envoy_cc_library(
    name = "service_account_token_lib",
    srcs = ["service_account_token.cc"],
    hdrs = ["service_account_token.h"],
    repository = "@envoy",
    deps = [
        ":json_struct_lib",
        "//api/envoy/http/common:base_proto_cc",
        "//src/api_proxy/auth:lib",
        "@envoy//include/envoy/event:dispatcher_interface",
        "@envoy//include/envoy/server:filter_config_interface",
        "@envoy//include/envoy/upstream:cluster_manager_interface",
        "@envoy//source/common/init:target_lib",
    ],
)

envoy_cc_test(
    name = "token_subscriber_test",
    size = "small",
    srcs = ["token_subscriber_test.cc"],
    repository = "@envoy",
    deps = [
        ":token_subscriber_lib",
        "@envoy//test/mocks/init:init_mocks",
        "@envoy//test/mocks/server:server_mocks",
        "@envoy//test/test_common:utility_lib",
    ],
)

envoy_cc_test(
    name = "iam_token_subscriber_test",
    size = "small",
    srcs = ["iam_token_subscriber_test.cc"],
    repository = "@envoy",
    deps = [
        ":iam_token_subscriber_lib",
        "@envoy//test/mocks/init:init_mocks",
        "@envoy//test/mocks/server:server_mocks",
        "@envoy//test/test_common:utility_lib",
    ],
)

envoy_cc_test(
    name = "service_account_token_test",
    size = "small",
    srcs = ["service_account_token_test.cc"],
    repository = "@envoy",
    deps = [
        ":service_account_token_lib",
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
    size = "small",
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
    size = "small",
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
