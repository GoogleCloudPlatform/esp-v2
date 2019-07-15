load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

ABSEIL_COMMIT = "61c9bf3e3e1c28a4aa6d7f1be4b37fd473bb5529"  # same as Envoy: 2019-06-05
ABSEIL_SHA256 = "7ddf863ddced6fa5bf7304103f9c7aa619c20a2fcf84475512c8d3834b9d14fa"

def abseil_repositories(bind = True):
    http_archive(
        name = "com_google_absl",
        strip_prefix = "abseil-cpp-" + ABSEIL_COMMIT,
        url = "https://github.com/abseil/abseil-cpp/archive/" + ABSEIL_COMMIT + ".tar.gz",
        sha256 = ABSEIL_SHA256,
    )

    if bind:
        native.bind(
            name = "abseil_strings",
            actual = "@com_google_absl//absl/strings:strings",
        )
        native.bind(
            name = "abseil_time",
            actual = "@com_google_absl//absl/time:time",
        )
        native.bind(
            name = "abseil_flat_hash_map",
            actual = "@com_google_absl//absl/container:flat_hash_map",
        )
        native.bind(
          name = "abseil_flat_hash_set",
          actual = "@com_google_absl//absl/container:flat_hash_set",
        )

    _cctz_repositories(bind)

CCTZ_COMMIT = "e19879df3a14791b7d483c359c4acd6b2a1cd96b"
CCTZ_SHA256 = "35d2c6cf7ddef1cf7c1bb054bdf2e8d7778242f6d199591a834c14d224b80c39"

def _cctz_repositories(bind = True):
    http_archive(
        name = "com_googlesource_code_cctz",
        url = "https://github.com/google/cctz/archive/" + CCTZ_COMMIT + ".tar.gz",
        sha256 = CCTZ_SHA256,
    )
