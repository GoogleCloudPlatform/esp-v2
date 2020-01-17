# Backend Routing Filter

This filter enables HTTP redirection when sending requests to backends
via Dynamic Routing. Based on the configuration of the backend rules,
this filter overwrites the `:path` header with corresponding remote backend address.

For more information on configuration and usage, see
[Understanding Path Translation](https://cloud.google.com/endpoints/docs/openapi/openapi-extensions#understanding_path_translation).

## Prerequisites

This filter will not function unless the following filters appear earlier in the filter chain:

- [Path Matcher](../path_matcher/README.md)

This filter is designed to strongly integrate with the following filters:

- [Backend Auth](../backend_auth/README.md)

## Configuration

View the [backend routing configuration proto](../../../../api/envoy/http/backend_routing/config.proto)
for inline documentation.