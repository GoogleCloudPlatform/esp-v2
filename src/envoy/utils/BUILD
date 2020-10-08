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
    srcs = [
        "json_struct_test.cc",
    ],
    repository = "@envoy",
    deps = [
        ":json_struct_lib",
        "@envoy//test/test_common:utility_lib",
    ],
)

envoy_cc_fuzz_test(
    name = "json_struct_fuzz_test",
    srcs = ["json_struct_fuzz_test.cc"],
    corpus = "//tests/fuzz/corpus:json_struct_corpus",
    repository = "@envoy",
    deps = [
        ":json_struct_lib",
        "//tests/fuzz/structured_inputs:json_struct_proto_cc_proto",
        "@envoy//test/fuzz:utility_lib",
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
        "@envoy//source/common/common:empty_string",
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
        "@envoy//source/common/common:empty_string",
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
        "@envoy//source/common/common:empty_string",
        "@envoy//source/exe:envoy_common_lib",
    ],
)

envoy_cc_library(
    name = "rc_detail_utils_lib",
    srcs = ["rc_detail_utils.cc"],
    hdrs = ["rc_detail_utils.h"],
    repository = "@envoy",
)

envoy_cc_test(
    name = "http_header_utils_test",
    srcs = ["http_header_utils_test.cc"],
    repository = "@envoy",
    deps = [
        ":http_header_utils_lib",
        "@envoy//source/common/common:empty_string",
        "@envoy//test/test_common:utility_lib",
    ],
)

envoy_cc_test(
    name = "rc_detail_utils_test",
    srcs = ["rc_detail_utils_test.cc"],
    repository = "@envoy",
    deps = [
        "rc_detail_utils_lib",
    ],
)
