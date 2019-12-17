# Path Matcher Filter

This filter matches the request path to an operation (selector) and stores it
in the shared filter state. The results of this match are used the following filters:

- [Backend Auth](../backend_auth/README.md)
- [Backend Routing](../backend_routing/README.md)
- [Service Control](../service_control/README.md)

This filter also extracts variable bindings, transforms them into query parameters,
and stores them in the shared filter state. The result of the transformation is
used by the following filters:

- [Backend Routing](../backend_routing/README.md)