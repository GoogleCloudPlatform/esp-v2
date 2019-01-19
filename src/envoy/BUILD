load(
    "@envoy//bazel:envoy_build_system.bzl",
    "envoy_cc_binary",
)

package(
    default_visibility = [
        "//src/envoy:__subpackages__",
    ]
)

envoy_cc_binary(
    name = "envoy",
    repository = "@envoy",
    deps = [
        "//src/envoy/http/backend_auth:filter_factory",
        "//src/envoy/http/path_matcher:filter_factory",
        "//src/envoy/http/service_control:filter_factory",
        "@envoy//source/exe:envoy_main_entry_lib",
    ],
)
