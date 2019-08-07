package(
    default_visibility = [
        "//src/api_proxy/service_control:__subpackages__",
    ],
)

cc_library(
    name = "request_builder_lib",
    srcs = ["request_builder.cc"],
    hdrs = [
        "request_builder.h",
        "request_info.h",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//external:abseil_strings",
        "//external:service_config",
        "//external:servicecontrol_client",
        "//src/api_proxy/utils",
        "@envoy//source/common/common:base64_lib",
    ],
)

cc_test(
    name = "request_builder_test",
    size = "small",
    srcs = [
        "request_builder_test.cc",
    ],
    data = glob(["testdata/*.golden"]),
    linkstatic = 1,
    deps = [
        ":request_builder_lib",
        "//external:googletest_main",
    ],
)

cc_library(
    name = "logs_metrics_loader_lib",
    srcs = ["logs_metrics_loader.cc"],
    hdrs = ["logs_metrics_loader.h"],
    visibility = ["//visibility:public"],
    deps = [
        ":request_builder_lib",
        "//external:googletest_prod",
        "//external:service_config",
    ],
)

cc_test(
    name = "logs_metrics_loader_test",
    size = "small",
    srcs = [
        "logs_metrics_loader_test.cc",
    ],
    linkstatic = 1,
    deps = [
        ":logs_metrics_loader_lib",
        "//external:googletest_main",
    ],
)
