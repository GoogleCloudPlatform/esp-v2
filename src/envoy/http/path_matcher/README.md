# Path Matcher Filter

## Overview

This filter operates on downstream headers to determine the operation name and
map variable bindings for an API request.
It then updates the shared filter state, reducing redundant work in later filters.

State modifications:
- Modifies shared filter state

### Operation Names

In a Google Cloud Endpoints service configuration, each API path is identified by a unique selector.
This is documented in the [path matcher bootstrap configuration test](../../../go/bootstrap/static/testdata/README.md#path-matcherpath_matcher).

This filter matches the request path to an operation (selector) and stores it
in the shared filter state. The results of this match are used the following filters:

- [Backend Auth](../backend_auth/README.md)
- [Backend Routing](../backend_routing/README.md)
- [Service Control](../service_control/README.md)
- [JWT Authn](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/security/jwt_authn_filter)

This selector is used by these filters (instead of the path) for a variety of operations:

- Selecting a JWT provider for client authentication
- Checking if an API key is required
- Rewriting paths for remote backends
- Attached a JWT for backend authentication

### Variable Bindings

In a Google Cloud Endpoints service configuration, certain variables may need to be extracted from a request path.
This is documented in the [path matcher bootstrap configuration test](../../../go/bootstrap/static/testdata/README.md#path-matcherpath_matcher).

This filter extracts variable bindings, transforms them into query parameters,
and stores them in the shared filter state. The result of the transformation is
used by the following filters:

- [Backend Routing](../backend_routing/README.md)

The variable bindings are used for:

- Attaching query parameters to the request

### Method override

If the original request has a `X-HTTP-Method-Override` header, this filter will overwrite
the `:method` header. This will result in all future headers handling the request with
the overridden path.

## Configuration

View the [path matcher configuration proto](../../../../api/envoy/http/path_matcher/config.proto)
for inline documentation.
