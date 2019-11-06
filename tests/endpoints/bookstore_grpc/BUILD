load("@com_google_protobuf//:protobuf.bzl", "cc_proto_library")

cc_proto_library(
    name = "bookstore_proto",
    srcs = ["proto/bookstore.proto"],
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
        "proto/bookstore.proto",
    ],
    outs = ["bookstore.descriptor"],
    cmd = """$(location //external:protoc) $(location :proto/bookstore.proto) \
        -I$$(dirname $(location @com_github_googleapis_googleapis//:google/rpc/status.proto))/../.. \
        -Iexternal/com_google_protobuf/src \
        -I$$(dirname $(location :proto/bookstore.proto)) \
        --include_imports --descriptor_set_out=$@""",
    tools = ["//external:protoc"],
)