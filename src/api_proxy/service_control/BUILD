

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
