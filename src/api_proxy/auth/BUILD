load(
    "@envoy//bazel:envoy_build_system.bzl",
    "envoy_basic_cc_library",
    "envoy_cc_fuzz_test",
)

envoy_basic_cc_library(
    name = "auth_token_lib",
    srcs = [
        "auth_token.cc",
    ],
    hdrs = [
        "auth_token.h",
    ],
    visibility = ["//visibility:public"],
    deps = [
        ":grpc_internals_lib",
        "//external:abseil_optional",
    ],
)

envoy_basic_cc_library(
    name = "grpc_internals_lib",
    hdrs = [
        "grpc_internals.h",
    ],
    visibility = ["//visibility:private"],
    deps = [
        "//external:grpc",
        "//external:protobuf",
    ],
)

envoy_cc_fuzz_test(
    name = "auth_token_fuzz_test",
    srcs = ["auth_token_fuzz_test.cc"],
    corpus = "//tests/fuzz/corpus:auth_token_corpus",
    repository = "@envoy",
    deps = [
        ":auth_token_lib",
        "//tests/fuzz/structured_inputs:auth_token_proto_cc_proto",
        "@envoy//test/fuzz:utility_lib",
        "@envoy//test/test_common:utility_lib",
    ],
)
