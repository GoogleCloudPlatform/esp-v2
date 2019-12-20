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
    name = "iam_token_subscriber_proto",
    srcs = ["iam_token_subscriber.proto"],
    deps = [
        "@envoy//test/fuzz:common_proto",
    ],
)

envoy_proto_library(
    name = "backend_routing_filter_proto",
    srcs = ["backend_routing_filter.proto"],
    deps = [
        "//api/envoy/http/backend_routing:config_proto",
        "@envoy//test/fuzz:common_proto",
    ],
)

envoy_proto_library(
    name = "http_template_proto",
    srcs = ["http_template.proto"],
    deps = [
        "@envoy//test/fuzz:common_proto",
    ],
)
