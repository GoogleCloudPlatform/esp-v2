package(
    default_visibility = [
        "//src/envoy:__subpackages__",
    ],
)

load(
    "@envoy//bazel:envoy_build_system.bzl",
    "envoy_cc_fuzz_test",
)

cc_library(
    name = "path_matcher_lib",
    srcs = [
        "http_template.cc",
        "path_matcher.cc",
        "path_matcher_node.cc",
    ],
    hdrs = [
        "http_template.h",
        "path_matcher.h",
        "path_matcher_node.h",
    ],
    deps = [
        "//external:abseil_strings",
    ],
)

cc_test(
    name = "http_template_test",
    size = "small",
    srcs = ["http_template_test.cc"],
    linkopts = [
        "-lm",
        "-lpthread",
    ],
    linkstatic = 1,
    deps = [
        ":path_matcher_lib",
        "@com_google_googletest//:gtest_main",
    ],
)

cc_test(
    name = "path_matcher_test",
    size = "small",
    srcs = ["path_matcher_test.cc"],
    linkopts = [
        "-lm",
        "-lpthread",
    ],
    linkstatic = 1,
    deps = [
        ":path_matcher_lib",
        "@com_google_googletest//:gtest_main",
    ],
)

cc_library(
    name = "variable_binding_utils_lib",
    srcs = [
        "variable_binding_utils.cc",
    ],
    hdrs = [
        "variable_binding_utils.h",
    ],
    deps = [
        ":path_matcher_lib",
        "//external:abseil_flat_hash_map",
        "//external:abseil_strings",
    ],
)

cc_test(
    name = "variable_binding_utils_test",
    size = "small",
    srcs = [
        "variable_binding_utils_test.cc",
    ],
    linkopts = [
        "-lm",
        "-lpthread",
    ],
    linkstatic = 1,
    deps = [
        ":variable_binding_utils_lib",
        "@com_google_googletest//:gtest_main",
    ],
)

envoy_cc_fuzz_test(
    name = "http_template_fuzz_test",
    srcs = ["http_template_fuzz_test.cc"],
    corpus = "//tests/fuzz/corpus:http_template_corpus",
    repository = "@envoy",
    deps = [
        ":path_matcher_lib",
        "//tests/fuzz/structured_inputs:http_template_proto_cc_proto",
        "@envoy//test/fuzz:utility_lib",
        "@envoy//test/test_common:utility_lib",
    ],
)
