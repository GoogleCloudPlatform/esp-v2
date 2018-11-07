workspace(name = "gcpproxy")

load(
     "//:repositories.bzl",
     "all_dependencies",
)
all_dependencies()
  
bind(
    name = "boringssl_crypto",
    actual = "//external:ssl",
)

# use the istio forked one with a hack for issue:
# https://github.com/envoyproxy/envoy/issues/4924
ENVOY_SHA = "a0b180dd3e8f81478d399fe2812e24a478b083f4"

http_archive(
    name = "envoy",
    strip_prefix = "envoy-" + ENVOY_SHA,
    url = "https://github.com/istio/envoy/archive/" + ENVOY_SHA + ".zip",
)

load("@envoy//bazel:repositories.bzl", "envoy_dependencies")
envoy_dependencies()

load("@envoy//bazel:cc_configure.bzl", "cc_configure")
cc_configure()

load("@envoy_api//bazel:repositories.bzl", "api_dependencies")
api_dependencies()

load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains")
go_rules_dependencies()
go_register_toolchains()

load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")
git_repository(
    name = "org_pubref_rules_protobuf",
    commit = "563b674a2ce6650d459732932ea2bc98c9c9a9bf",  # Nov 28, 2017 (bazel 0.8.0 support)
    remote = "https://github.com/pubref/rules_protobuf",
)
