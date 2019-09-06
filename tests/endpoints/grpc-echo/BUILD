load(
    "@envoy//bazel:envoy_build_system.bzl",
    "envoy_cc_binary",
)
load("@com_google_protobuf//:protobuf.bzl", "cc_proto_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

cc_proto_library(
    name = "grpc-test_proto",
    srcs = ["proto/grpc-test.proto"],
    default_runtime = "//external:protobuf",
    protoc = "//external:protoc",
    use_grpc_plugin = True,
    deps = [
        "//external:servicecontrol",
    ],
)

go_proto_library(
    name = "grpc-test_go_proto",
    proto = ":grpc-test.proto",
    deps = [
        "@com_github_googleapis_googleapis//google/api/servicecontrol/v1/service_controller.proto",
    ],
)

genrule(
    name = "grpc-test_descriptor",
    srcs = [
        "@com_github_googleapis_googleapis//:google/rpc/status.proto",
        "@com_github_googleapis_googleapis//:google/api/servicecontrol/v1/service_controller.proto",
        "@com_github_googleapis_googleapis//:google/api/servicecontrol/v1/check_error.proto",
        "@com_github_googleapis_googleapis//:google/api/servicecontrol/v1/distribution.proto",
        "@com_github_googleapis_googleapis//:google/api/servicecontrol/v1/log_entry.proto",
        "@com_github_googleapis_googleapis//:google/api/servicecontrol/v1/metric_value.proto",
        "@com_github_googleapis_googleapis//:google/api/servicecontrol/v1/operation.proto",
        "@com_github_googleapis_googleapis//:google/api/annotations.proto",
        "@com_github_googleapis_googleapis//:google/api/http.proto",
        "@com_github_googleapis_googleapis//:google/logging/type/log_severity.proto",
        "@com_github_googleapis_googleapis//:google/type/money.proto",
        "@com_google_protobuf//:well_known_protos",
        "proto/grpc-test.proto",
    ],
    outs = ["grpc-test.descriptor"],
    cmd = """$(location //external:protoc) $(location :proto/grpc-test.proto) \
        -I$$(dirname $(location @com_github_googleapis_googleapis//:google/rpc/status.proto))/../.. \
        -Iexternal/com_google_protobuf/src \
        -I$$(dirname $(location :proto/grpc-test.proto)) \
        --include_imports --descriptor_set_out=$@""",
    tools = ["//external:protoc"],
)

cc_binary(
    name = "grpc-test-server",
    testonly = 1,
    srcs = ["grpc-test-server.cc"],
    deps = [
        ":grpc-test_proto",
        "@com_github_grpc_grpc//:grpc++",
    ],
)

cc_binary(
    name = "grpc-test-client",
    testonly = 1,
    srcs = ["grpc-test-client.cc"],
    linkopts = ["-lrt"],
    deps = [
        ":client-test-lib",
        ":grpc-test_proto",
        "@envoy//test/test_common:utility_lib",
    ],
)

cc_library(
    name = "client-test-lib",
    testonly = 1,
    srcs = ["client-test-lib.cc"],
    hdrs = ["client-test-lib.h"],
    linkopts = ["-lm"],
    visibility = ["//visibility:private"],
    deps = [
        ":grpc-test_proto",
        "@com_github_grpc_grpc//:grpc++",
        "@envoy//test/test_common:utility_lib",
    ],
)
