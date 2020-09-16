# ESPv2 extension configuration override.
#
# We do not need all the extensions, so we disable the ones we do not need. This
# reduces binary size, compile time, and attack vectors.

# This list overrides the default list controlling what extensions are compiled
# into envoy.
#
# See https://github.com/envoyproxy/envoy/tree/master/bazel/README.md
# and https://github.com/envoyproxy/envoy/blob/master/source/extensions/extensions_build_config.bzl
EXTENSIONS = {
    # All extensions explicitly referenced by config generator and our tests.
    "envoy.access_loggers.file": "//source/extensions/access_loggers/file:config",
    "envoy.filters.http.cors": "//source/extensions/filters/http/cors:config",
    "envoy.filters.http.grpc_json_transcoder": "//source/extensions/filters/http/grpc_json_transcoder:config",
    "envoy.filters.http.grpc_web": "//source/extensions/filters/http/grpc_web:config",
    "envoy.filters.http.health_check": "//source/extensions/filters/http/health_check:config",
    "envoy.filters.http.jwt_authn": "//source/extensions/filters/http/jwt_authn:config",
    "envoy.filters.http.router": "//source/extensions/filters/http/router:config",
    "envoy.filters.network.http_connection_manager": "//source/extensions/filters/network/http_connection_manager:config",
    "envoy.tracers.opencensus": "//source/extensions/tracers/opencensus:config",

    # Implicitly needed for TLS config.
    "envoy.transport_sockets.raw_buffer": "//source/extensions/transport_sockets/raw_buffer:config",

    # Remaining items are for API Gateway and not covered by our tests. Do not remove.
    "envoy.access_loggers.http_grpc": "//source/extensions/access_loggers/grpc:http_config",
    "envoy.stat_sinks.metrics_service": "//source/extensions/stat_sinks/metrics_service:config",
    "envoy.stat_sinks.statsd": "//source/extensions/stat_sinks/statsd:config",
}
