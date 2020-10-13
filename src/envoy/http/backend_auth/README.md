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

View the [backend auth configuration proto](../../../../api/envoy/v9/http/backend_auth/config.proto)
for inline documentation.

## Statistics

This filter records statistics.

### Counters

- `denied_by_no_token`: Number of API Consumer requests that are denied due to the filter
 missing a token needed for the request. Two possible causes: 1) the `jwt_audience` specified in the
 route entry perFilterConfig for this filter PerRouteFilerConfig is not in the `jwt_audience_list` in
 the FilterConfig. 2) fails to fetch ID token.
- `denied_by_no_route`: Number of API Consumer requests that are denied due to not route configurated.
- `allowed_by_auth_not_required`: Number of API Consumer requests that are allowed without sending ID
 token to the backend.
- `token_added`: Number of API Consumer requests that are allowed through with
 modification for backend authentication.
