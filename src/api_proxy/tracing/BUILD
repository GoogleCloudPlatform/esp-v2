licenses(["notice"])

package(default_visibility = ["//visibility:public"])

cc_library(
    name = "trace_context_utils_lib",
    srcs = [
        "trace_context_utils.cc",
    ],
    hdrs = [
        "trace_context_utils.h",
    ],
    deps = [
        "@com_google_absl//absl/strings",
        "@com_google_absl//absl/strings:str_format",
        "@com_google_absl//absl/types:optional",
    ],
)

cc_test(
    name = "trace_context_utils_test",
    srcs = [
        "trace_context_utils_test.cc",
    ],
    deps = [
        ":trace_context_utils_lib",
        "@com_google_googletest//:gtest_main",
    ],
)
