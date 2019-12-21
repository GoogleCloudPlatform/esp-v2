load("@com_google_protobuf//:protobuf.bzl", "cc_proto_library")

exports_files([
    "bookstore_v2.proto",
])

cc_proto_library(
    name = "bookstore_v2_proto",
    srcs = ["bookstore_v2.proto"],
    default_runtime = "//external:protobuf",
    protoc = "//external:protoc",
    deps = [
        "//external:servicecontrol",
    ],
)
