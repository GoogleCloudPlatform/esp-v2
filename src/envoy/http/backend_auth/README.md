# Backend Auth Filter

This filter enables proxy-to-service authorization when sending requests to backends
via Dynamic Routing. If authentication is configured inside a backend rule,
this filter overwrites the `Authorization` header with corresponding identity token.

_Note_: this is a pass through filter. If the requested operation is not configured in the
filter config, the request will pass through unmodified.

## Prerequisites

This filter will not function unless the following filters appear earlier in the filter chain:

- [Path Matcher](../path_matcher/README.md)

This filter is designed to strongly integrate with the following filters:

- [Backend Routing](../backend_routing/README.md)

## Configuration

View the [backend auth configuration proto](../../../../api/envoy/v6/http/backend_auth/config.proto)
for inline documentation.

## Statistics

This filter records statistics.

### Counters

- `denied_by_no_token`: Number of API Consumer requests that are denied due to the filter
 missing a token needed for the request.
- `denied_by_no_operation`: Number of API Consumer requests that are denied due to missing filter state.
- `allowed_by_no_configured_rules`: Number of API Consumer requests that are allowed through
 without modification. Occurs when the operation is not configured for auth rewrite.
- `token_added`: Number of API Consumer requests that are allowed through with
 modification for backend authentication.