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
    name = "request_info_lib",
    hdrs = [
        "request_info.h",
    ],
    deps = [
        "@com_google_absl//absl/status",
        "@com_google_absl//absl/types:optional",
    ],
)

envoy_basic_cc_library(
    name = "request_builder_lib",
    srcs = ["request_builder.cc"],
    hdrs = [
        "request_builder.h",
    ],
    visibility = ["//visibility:public"],
    # FIXME: Direct use of envoy function in non-envoy code. Consider copying
    # relevant code to utils to remove this dependency in the future.
    deps = [
        ":request_info_lib",
        "//external:abseil_strings",
        "//src/api_proxy/utils",
        "@com_github_googleapis_googleapis//google/api:service_cc_proto",
        "@envoy//source/common/common:assert_lib",
        "@envoy//source/common/common:base64_lib",
        "@envoy//source/common/grpc:status_lib",
        "@servicecontrol_client_git//:service_control_client_lib",
    ],
)

envoy_basic_cc_library(
    name = "check_response_converter_lib",
    srcs = ["check_response_convert_utils.cc"],
    hdrs = [
        "check_response_convert_utils.h",
    ],
    visibility = ["//visibility:public"],
    deps = [
        ":request_info_lib",
        "//external:abseil_strings",
        "//src/api_proxy/utils",
        "@com_github_googleapis_googleapis//google/api:service_cc_proto",
        "@servicecontrol_client_git//:service_control_client_lib",
    ],
)

envoy_cc_test(
    name = "check_response_converter_test",
    srcs = [
        "check_response_convert_utils_test.cc",
    ],
    repository = "@envoy",
    deps = [
        ":check_response_converter_lib",
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
        "@com_google_absl//absl/status",
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
