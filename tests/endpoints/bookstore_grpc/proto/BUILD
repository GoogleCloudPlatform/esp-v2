load("@com_google_protobuf//:protobuf.bzl", "cc_proto_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

exports_files([
    "api_descriptor.pb",
])

cc_proto_library(
    name = "bookstore_proto",
    srcs = ["bookstore.proto"],
    default_runtime = "//external:protobuf",
    protoc = "//external:protoc",
    deps = [
        "//external:servicecontrol",
    ],
)

genrule(
    name = "bookstore_descriptor",
    srcs = [
        "@com_github_googleapis_googleapis//:google/rpc/status.proto",
        "@com_github_googleapis_googleapis//:google/api/annotations.proto",
        "@com_github_googleapis_googleapis//:google/api/http.proto",
        "@com_google_protobuf//:well_known_protos",
        "//tests/endpoints/bookstore_grpc/proto/v1:bookstore.proto",
        "//tests/endpoints/bookstore_grpc/proto/v2:bookstore_v2.proto",
    ],
    outs = ["bookstore.descriptor"],
    cmd = """$(location //external:protoc) $(location //tests/endpoints/bookstore_grpc/proto/v1:bookstore.proto) \
        $(location //tests/endpoints/bookstore_grpc/proto/v2:bookstore_v2.proto) \
        -I$$(dirname $(location @com_github_googleapis_googleapis//:google/rpc/status.proto))/../.. \
        -Iexternal/com_google_protobuf/src \
        -I$$(dirname $(location //tests/endpoints/bookstore_grpc/proto/v1:bookstore.proto)) \
        -I$$(dirname $(location //tests/endpoints/bookstore_grpc/proto/v2:bookstore_v2.proto)) \
        --include_imports --descriptor_set_out=$@""",
    tools = ["//external:protoc"],
)
