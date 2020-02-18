load("@rules_proto//proto:defs.bzl", "proto_library")
load(
    "@com_google_googleapis_imports//:imports.bzl",
    "cc_proto_library",
    "cc_grpc_library")

proto_library(
    name = "grpc-test_proto",
    srcs = ["proto/grpc-test.proto"],
    deps = [
        "@com_github_googleapis_googleapis//google/api:annotations_proto",
        "@com_github_googleapis_googleapis//google/api/servicecontrol/v1:servicecontrol_proto",

    ],
)

cc_proto_library(
    name = "grpc-test_cc_proto",
    deps = [

        ":grpc-test_proto",
    ],
)

cc_grpc_library(
    name = "grpc-test_cc_grpc",
    srcs = [":grpc-test_proto"],
    grpc_only = True,
    deps = [":grpc-test_cc_proto"],
)

cc_library(
    name = "client-test-lib",
    testonly = 1,
    srcs = ["client-test-lib.cc"],
    hdrs = ["client-test-lib.h"],
    linkopts = ["-lm"],
    visibility = ["//visibility:private"],
    deps = [
        ":grpc-test_cc_grpc",
        "@com_github_grpc_grpc//:grpc++",
        "@envoy//test/test_common:utility_lib",
    ],
)

cc_binary(
    name = "grpc-test-server",
    testonly = 1,
    srcs = ["grpc-test-server.cc"],
    deps = [
        ":grpc-test_cc_grpc",
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
        ":grpc-test_cc_grpc",
        "@envoy//test/test_common:utility_lib",
    ],
)

genrule(
    name = "grpc-test_descriptor",
    srcs = [
        "@com_github_googleapis_googleapis//google/rpc:status.proto",
        "@com_github_googleapis_googleapis//google/api/servicecontrol/v1:service_controller.proto",
        "@com_github_googleapis_googleapis//google/api/servicecontrol/v1:check_error.proto",
        "@com_github_googleapis_googleapis//google/api/servicecontrol/v1:distribution.proto",
        "@com_github_googleapis_googleapis//google/api/servicecontrol/v1:log_entry.proto",
        "@com_github_googleapis_googleapis//google/api/servicecontrol/v1:metric_value.proto",
        "@com_github_googleapis_googleapis//google/api/servicecontrol/v1:operation.proto",
        "@com_github_googleapis_googleapis//google/api:annotations.proto",
        "@com_github_googleapis_googleapis//google/api:http.proto",
        "@com_github_googleapis_googleapis//google/logging/type:log_severity.proto",
        "@com_github_googleapis_googleapis//google/type:money.proto",
        "@com_google_protobuf//:well_known_protos",
        "proto/grpc-test.proto",
    ],
    outs = ["grpc-test.descriptor"],
    cmd = """$(location //external:protoc) $(location :proto/grpc-test.proto) \
        -I$$(dirname $(location @com_github_googleapis_googleapis//google/rpc:status.proto))/../.. \
        -Iexternal/com_google_protobuf/src \
        -I$$(dirname $(location :proto/grpc-test.proto)) \
        --include_imports --descriptor_set_out=$@""",
    tools = ["//external:protoc"],
)