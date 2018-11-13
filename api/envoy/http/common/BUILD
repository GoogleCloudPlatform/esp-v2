load("@envoy_api//bazel:api_build_system.bzl", "api_proto_library")

DEFAULT_VISIBILITY = [
    "//api/envoy/http:__subpackages__",
    "//src/envoy/http:__subpackages__",
]

package(default_visibility = DEFAULT_VISIBILITY)

api_proto_library(
    name = "matcher_proto",
    srcs = [
        "matcher.proto",
    ],
    visibility = DEFAULT_VISIBILITY,
)
