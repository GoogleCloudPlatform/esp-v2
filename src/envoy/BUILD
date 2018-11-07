load(
    "@envoy//bazel:envoy_build_system.bzl",
    "envoy_cc_binary",
)

envoy_cc_binary(
    name = "envoy",
    repository = "@envoy",
    deps = [
        "//src/envoy/http/service_control:filter_factory",
        "@envoy//source/exe:envoy_main_entry_lib",
    ],
)
