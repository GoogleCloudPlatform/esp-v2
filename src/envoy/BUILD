load(
    "@envoy//bazel:envoy_build_system.bzl",
    "envoy_cc_binary",
)

package(
    default_visibility = ["//visibility:public"],
)

alias(
    name = "backend_auth",
    actual = "//src/envoy/http/backend_auth:filter_factory",
)

alias(
    name = "grpc_metadata_scrubber",
    actual = "//src/envoy/http/grpc_metadata_scrubber:filter_factory",
)

alias(
    name = "header_sanitizer",
    actual = "//src/envoy/http/header_sanitizer:filter_factory",
)

alias(
    name = "path_rewrite",
    actual = "//src/envoy/http/path_rewrite:filter_factory",
)

alias(
    name = "service_control",
    actual = "//src/envoy/http/service_control:filter_factory",
)

alias(
    name = "main",
    actual = "@envoy//source/exe:envoy_main_entry_lib",
)

envoy_cc_binary(
    name = "envoy",
    repository = "@envoy",
    deps = [
        ":backend_auth",
        ":grpc_metadata_scrubber",
        ":header_sanitizer",
        ":main",
        ":path_rewrite",
        ":service_control",
    ],
)
