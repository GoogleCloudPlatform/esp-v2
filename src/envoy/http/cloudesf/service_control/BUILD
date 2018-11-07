

cc_library(
    name = "proto",
    srcs = [
        "proto.cc",
    ],
    hdrs = [
        "info.h",
        "proto.h",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//external:servicecontrol_client",
    ],
)

cc_test(
    name = "proto_test",
    size = "small",
    srcs = [
        "proto_test.cc",
    ],
    data = glob(["testdata/*.golden"]),
    linkstatic = 1,
    deps = [
        ":proto",
        "//external:googletest_main",
    ],
)
