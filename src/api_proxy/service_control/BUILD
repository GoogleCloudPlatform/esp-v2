package(
    default_visibility = [
        "//src/api_proxy/service_control:__subpackages__",
    ]
)

cc_library(
    name = "request_builder_lib",
    srcs = ["request_builder.cc"],
    hdrs = [
        "request_info.h",
        "request_builder.h",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//external:servicecontrol_client",
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
