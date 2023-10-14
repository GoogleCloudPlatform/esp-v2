load("@rules_cc//cc:defs.bzl", "cc_proto_library")
load("@rules_proto//proto:defs.bzl", "proto_library")

exports_files([
    "bookstore_v2.proto",
])

proto_library(
    name = "bookstore_v2_proto",
    srcs = ["bookstore_v2.proto"],
    deps = [
        "@com_github_googleapis_googleapis//google/api:annotations_proto",
    ],
)

cc_proto_library(
    name = "bookstore_v2_cc_proto",
    deps = [
        ":bookstore_v2_proto",
    ],
)
