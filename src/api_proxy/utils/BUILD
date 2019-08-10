sh_binary(
    name = "version_macro_template",
    srcs = [
        "version_macro_template.sh",
    ],
)

genrule(
    name = "version_header_gen",
    srcs = [
        ":version",
    ],
    outs = [
        "version_macro.h",
    ],
    cmd = "$(location :version_macro_template) $(location :version) > $@",
    tools = [
        ":version_macro_template",
    ],
)

cc_library(
    name = "utils",
    srcs = [
        "version.cc",
    ],
    hdrs = [
        "version.h",
        "version_macro.h",
    ],
    visibility = ["//:__subpackages__"],
)

cc_test(
    name = "version_test",
    size = "small",
    srcs = [
        "version_test.cc",
    ],
    linkopts = [
        "-lpthread",
    ],
    deps = [
        ":utils",
        "@com_google_googletest//:gtest_main",
    ],
)
