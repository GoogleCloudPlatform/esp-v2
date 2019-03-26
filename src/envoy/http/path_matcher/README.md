# Path Matcher Filter

This filter matches the request path to an operation (selector) and stores it
in dynamic metadata. This metadata used by the following other filters:

* Service Control
* Backend Auth
* Dynamic Router

If a match is not found, it rejects the request.

This filter is required by Config Manager.
