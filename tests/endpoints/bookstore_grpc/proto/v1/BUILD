load("@rules_cc//cc:defs.bzl", "cc_proto_library")
load("@rules_proto//proto:defs.bzl", "proto_library")
load(
    "@com_github_grpc_grpc//bazel:grpc_build_system.bzl",
    "grpc_proto_library",
    #    "health_proto_descriptor",
)

exports_files([
    "bookstore.proto",
])

proto_library(
    name = "bookstore_v1_proto",
    srcs = [
        "bookstore.proto",
    ],
    deps = [
        "@com_github_googleapis_googleapis//google/api:annotations_proto",
        "@com_google_protobuf//:any_proto",
    ],
)

cc_proto_library(
    name = "bookstore_v1_cc_proto",
    deps = [
        ":bookstore_v1_proto",
    ],
)
