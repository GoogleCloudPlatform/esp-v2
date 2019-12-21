load("@com_google_protobuf//:protobuf.bzl", "cc_proto_library")

exports_files([
    "bookstore.proto",
])

cc_proto_library(
    name = "bookstore_v1_proto",
    srcs = ["bookstore.proto"],
    default_runtime = "//external:protobuf",
    protoc = "//external:protoc",
    deps = [
        "//external:servicecontrol",
    ],
)
