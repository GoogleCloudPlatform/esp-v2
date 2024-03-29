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
    name = "iam_token_info_lib",
    srcs = ["iam_token_info.cc"],
    hdrs = ["iam_token_info.h"],
    repository = "@envoy",
    deps = [
        ":token_info_lib",
        "//external:protobuf",
        "//src/envoy/utils:json_struct_lib",
        "@envoy//source/common/common:empty_string",
        "@envoy//source/common/http:headers_lib",
        "@envoy//source/common/http:message_lib",
        "@envoy//source/common/http:utility_lib",
    ],
)

envoy_cc_library(
    name = "imds_token_info_lib",
    srcs = ["imds_token_info.cc"],
    hdrs = ["imds_token_info.h"],
    repository = "@envoy",
    deps = [
        ":token_info_lib",
        "//src/envoy/utils:json_struct_lib",
        "@envoy//source/common/http:headers_lib",
        "@envoy//source/common/http:message_lib",
        "@envoy//source/common/http:utility_lib",
    ],
)

envoy_cc_library(
    name = "token_subscriber_lib",
    srcs = ["token_subscriber.cc"],
    hdrs = ["token_subscriber.h"],
    repository = "@envoy",
    deps = [
        ":token_info_lib",
        "//api/envoy/v12/http/common:base_proto_cc_proto",
        "@envoy//envoy/common:time_interface",
        "@envoy//envoy/event:dispatcher_interface",
        "@envoy//envoy/server:filter_config_interface",
        "@envoy//envoy/upstream:cluster_manager_interface",
        "@envoy//source/common/common:assert_lib",
        "@envoy//source/common/common:enum_to_int",
        "@envoy//source/common/http:headers_lib",
        "@envoy//source/common/http:message_lib",
        "@envoy//source/common/http:utility_lib",
        "@envoy//source/common/init:target_lib",
    ],
)

envoy_cc_library(
    name = "token_info_lib",
    hdrs = ["token_info.h"],
    repository = "@envoy",
    deps = [
        "@envoy//source/common/http:message_lib",
    ],
)

envoy_cc_library(
    name = "token_subscriber_factory_interface",
    hdrs = ["token_subscriber_factory.h"],
    repository = "@envoy",
    deps = [
        ":iam_token_info_lib",
        ":imds_token_info_lib",
        ":token_subscriber_lib",
        "//api/envoy/v12/http/common:base_proto_cc_proto",
    ],
)

envoy_cc_library(
    name = "mocks_lib",
    hdrs = ["mocks.h"],
    repository = "@envoy",
    deps = [
        ":token_info_lib",
        ":token_subscriber_factory_interface",
        "//api/envoy/v12/http/common:base_proto_cc_proto",
    ],
)

envoy_cc_library(
    name = "token_subscriber_factory_lib",
    hdrs = ["token_subscriber_factory_impl.h"],
    repository = "@envoy",
    deps = [
        ":iam_token_info_lib",
        ":imds_token_info_lib",
        ":token_subscriber_factory_interface",
        ":token_subscriber_lib",
        "//api/envoy/v12/http/common:base_proto_cc_proto",
    ],
)

envoy_cc_fuzz_test(
    name = "imds_token_info_fuzz_test",
    srcs = ["imds_token_info_fuzz_test.cc"],
    corpus = "//tests/fuzz/corpus:imds_token_info_corpus",
    repository = "@envoy",
    deps = [
        ":imds_token_info_lib",
        "//tests/fuzz/structured_inputs:imds_token_info_proto_cc_proto",
        "@envoy//test/fuzz:utility_lib",
        "@envoy//test/test_common:utility_lib",
    ],
)

envoy_cc_test(
    name = "imds_token_info_test",
    srcs = ["imds_token_info_test.cc"],
    repository = "@envoy",
    deps = [
        ":imds_token_info_lib",
    ],
)

envoy_cc_fuzz_test(
    name = "iam_token_info_fuzz_test",
    srcs = ["iam_token_info_fuzz_test.cc"],
    corpus = "//tests/fuzz/corpus:iam_token_info_corpus",
    repository = "@envoy",
    deps = [
        ":iam_token_info_lib",
        "//tests/fuzz/structured_inputs:iam_token_info_proto_cc_proto",
        "@envoy//test/fuzz:utility_lib",
        "@envoy//test/test_common:utility_lib",
    ],
)

envoy_cc_test(
    name = "iam_token_info_test",
    srcs = ["iam_token_info_test.cc"],
    repository = "@envoy",
    deps = [
        ":iam_token_info_lib",
        "@envoy//source/common/common:empty_string",
        "@envoy//test/test_common:utility_lib",
    ],
)

envoy_cc_test(
    name = "token_subscriber_test",
    srcs = ["token_subscriber_test.cc"],
    repository = "@envoy",
    deps = [
        ":mocks_lib",
        ":token_subscriber_lib",
        "//api/envoy/v12/http/common:base_proto_cc_proto",
        "@envoy//test/mocks/init:init_mocks",
        "@envoy//test/mocks/server:server_mocks",
        "@envoy//test/test_common:utility_lib",
    ],
)
