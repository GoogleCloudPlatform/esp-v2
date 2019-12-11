load(
    "@envoy//bazel:envoy_build_system.bzl",
    "envoy_proto_library",
)

package(
    default_visibility = [
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
