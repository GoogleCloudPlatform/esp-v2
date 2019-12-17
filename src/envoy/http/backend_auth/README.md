# Backend Auth Filter

This filter enables proxy-to-service authorization when sending requests to backends
via Dynamic Routing. If authentication is configured inside a backend rule,
this filter overwrites the `Authorization` header with corresponding identity token.

## Prerequisites

This filter will not function unless the following filters appear earlier in the filter chain:

- [Path Matcher](../path_matcher/README.md)

This filter is designed to strongly integrate with the following filters:

- [Backend Routing](../backend_routing/README.md)