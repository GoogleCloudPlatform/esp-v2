exports_files([
    "api_descriptor.pb",
])

genrule(
    name = "bookstore_descriptor",
    srcs = [
        "@com_google_googleapis//google/rpc:status.proto",
        "@com_google_googleapis//google/api:annotations.proto",
        "@com_google_googleapis//google/api:http.proto",
        "@com_google_protobuf//:well_known_protos",
        "//tests/endpoints/bookstore_grpc/proto/v1:bookstore.proto",
        "//tests/endpoints/bookstore_grpc/proto/v2:bookstore_v2.proto",
        "@com_github_grpc_grpc//src/proto/grpc/health/v1:health.proto",
    ],
    outs = ["bookstore.descriptor"],
    cmd = """$(location //external:protoc) $(location //tests/endpoints/bookstore_grpc/proto/v1:bookstore.proto) \
         $(location //tests/endpoints/bookstore_grpc/proto/v2:bookstore_v2.proto) \
        -I$$(dirname $(location @com_google_googleapis//google/rpc:status.proto))/../.. \
        -I$$(dirname $(location @com_github_grpc_grpc//src/proto/grpc/health/v1:health.proto))/../../.. \
        -Iexternal/com_google_protobuf/src \
        -I$$(dirname $(location //tests/endpoints/bookstore_grpc/proto/v1:bookstore.proto)) \
        -I$$(dirname $(location //tests/endpoints/bookstore_grpc/proto/v2:bookstore_v2.proto)) \
        --include_imports --descriptor_set_out=$@""",
    tools = ["//external:protoc"],
)
