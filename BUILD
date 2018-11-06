load(
    "@envoy//bazel:envoy_build_system.bzl",
    "envoy_cc_binary",
    "envoy_cc_library",
    "envoy_cc_test",
)

envoy_cc_binary(
    name = "cloudesf-envoy",
    repository = "@envoy",
    deps = [
        "//src/filters/http_filter_example:http_filter_config",
        "@envoy//source/exe:envoy_main_entry_lib",
    ],
)
