load(
    "@envoy//bazel:envoy_build_system.bzl",
    "envoy_basic_cc_library",
    "envoy_cc_test",
)

sh_binary(
    name = "version_macro_template",
    srcs = [
        "version_macro_template.sh",
    ],
)

genrule(
    name = "version_header_gen",
    srcs = [
        "//:version",
    ],
    outs = [
        "version_macro.h",
    ],
    cmd = "$(location :version_macro_template) $(location //:version) > $@",
    tools = [
        ":version_macro_template",
    ],
)

envoy_basic_cc_library(
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

envoy_cc_test(
    name = "version_test",
    srcs = [
        "version_test.cc",
    ],
    repository = "@envoy",
    deps = [
        ":utils",
    ],
)
