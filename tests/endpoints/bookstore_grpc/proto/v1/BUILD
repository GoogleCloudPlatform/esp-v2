load("@com_google_protobuf//:protobuf.bzl", "cc_proto_library")
load("@rules_proto//proto:defs.bzl", "proto_library")

exports_files([
    "bookstore.proto",
])

proto_library(
    name = "bookstore_v1_proto",
    srcs = ["bookstore.proto"],
    deps = [
        "@com_github_googleapis_googleapis//google/api:annotations_proto",
        "@com_google_protobuf//google/protobuf:any_proto",
    ],
)

cc_proto_library(
    name = "bookstore_v1_cc_proto",
    srcs = ["bookstore.proto"],
    default_runtime = "//external:protobuf",
    protoc = "//external:protoc",
    deps = [
        "//external:servicecontrol",
    ],
)
