load(
    "@envoy//bazel:envoy_build_system.bzl",
    "envoy_cc_library",
    "envoy_cc_test",
)

package(
    default_visibility = [
        "//src/envoy:__subpackages__",
    ],
)

envoy_cc_library(
    name = "filter_factory",
    srcs = ["filter_factory.cc"],
    repository = "@envoy",
    deps = [
        ":filter_lib",
        "@envoy//source/exe:envoy_common_lib",
    ],
)

envoy_cc_library(
    name = "filter_lib",
    srcs = [
        "filter.cc",
    ],
    hdrs = [
        "filter.h",
        "filter_config.h",
    ],
    repository = "@envoy",
    deps = [
        "//api/agent:agent_proto_cc",
        "//api/envoy/http/path_matcher:config_proto_cc",
        "//src/api_proxy/path_matcher:path_matcher_lib",
        "//src/api_proxy/path_matcher:variable_binding_utils_lib",
        "//src/envoy/utils:filter_state_utils_lib",
        "//src/envoy/utils:http_header_utils_lib",
        "@envoy//source/common/protobuf:utility_lib",
        "@envoy//source/exe:envoy_common_lib",
    ],
)
