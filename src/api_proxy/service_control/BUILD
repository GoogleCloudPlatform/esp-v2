load(
    "@envoy//bazel:envoy_build_system.bzl",
    "envoy_basic_cc_library",
    "envoy_cc_test",
)

package(
    default_visibility = [
        "//src/api_proxy/service_control:__subpackages__",
    ],
)

envoy_basic_cc_library(
    name = "request_builder_lib",
    srcs = ["request_builder.cc"],
    hdrs = [
        "request_builder.h",
        "request_info.h",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//external:abseil_strings",
        "//src/api_proxy/utils",
        "@com_github_googleapis_googleapis//google/api:service_cc_proto",
        "@envoy//source/common/common:base64_lib",
        "@servicecontrol_client_git//:service_control_client_lib",
    ],
)

envoy_cc_test(
    name = "check_response_test",
    srcs = [
        "check_response_test.cc",
    ],
    repository = "@envoy",
    deps = [
        ":request_builder_lib",
    ],
)

envoy_cc_test(
    name = "request_builder_test",
    srcs = [
        "request_builder_test.cc",
    ],
    data = glob(["testdata/*.golden"]),
    repository = "@envoy",
    deps = [
        ":request_builder_lib",
    ],
)

envoy_basic_cc_library(
    name = "logs_metrics_loader_lib",
    srcs = ["logs_metrics_loader.cc"],
    hdrs = ["logs_metrics_loader.h"],
    visibility = ["//visibility:public"],
    deps = [
        ":request_builder_lib",
        "@com_github_googleapis_googleapis//google/api:service_cc_proto",
        "@com_google_googletest//:gtest_prod",
    ],
)

envoy_cc_test(
    name = "logs_metrics_loader_test",
    srcs = [
        "logs_metrics_loader_test.cc",
    ],
    repository = "@envoy",
    deps = [
        ":logs_metrics_loader_lib",
    ],
)
