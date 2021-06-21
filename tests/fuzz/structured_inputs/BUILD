load(
    "@envoy//bazel:envoy_build_system.bzl",
    "envoy_proto_library",
)

package(
    default_visibility = [
        "//src/api_proxy:__subpackages__",
        "//src/envoy:__subpackages__",
    ],
)

envoy_proto_library(
    name = "json_struct_proto",
    srcs = ["json_struct.proto"],
    deps = [],
)

envoy_proto_library(
    name = "iam_token_info_proto",
    srcs = ["iam_token_info.proto"],
)

envoy_proto_library(
    name = "imds_token_info_proto",
    srcs = ["imds_token_info.proto"],
)

envoy_proto_library(
    name = "http_template_proto",
    srcs = ["http_template.proto"],
    deps = [
        "@envoy//test/fuzz:common_proto",
    ],
)

envoy_proto_library(
    name = "auth_token_proto",
    srcs = ["auth_token.proto"],
    deps = [
        "@envoy//test/fuzz:common_proto",
    ],
)

envoy_proto_library(
    name = "service_control_filter_proto",
    srcs = ["service_control_filter.proto"],
    deps = [
        "//api/envoy/v10/http/service_control:config_proto",
        "@envoy//test/fuzz:common_proto",
    ],
)
