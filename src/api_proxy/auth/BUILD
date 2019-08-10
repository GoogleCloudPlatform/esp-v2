load(
    "@envoy//bazel:envoy_build_system.bzl",
    "envoy_basic_cc_library",
)

envoy_basic_cc_library(
    name = "lib",
    srcs = [
        "auth_token.cc",
    ],
    hdrs = [
        "auth_token.h",
    ],
    visibility = ["//visibility:public"],
    deps = [
        ":json_util_lib",
    ],
)

envoy_basic_cc_library(
    name = "json_util_lib",
    srcs = [
        "json_util.cc",
    ],
    hdrs = [
        "grpc_internals.h",
        "json_util.h",
    ],
    visibility = ["//visibility:private"],
    deps = [
        "//external:grpc",
        "//external:protobuf",
    ],
)

cc_test(
    name = "json_util_test",
    size = "small",
    srcs = [
        "json_util_test.cc",
    ],
    linkstatic = 1,
    deps = [
        ":json_util_lib",
        "@com_google_googletest//:gtest_main",
    ],
)
