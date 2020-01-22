# Release 2.2.0 22-01-2020

- Fix bug in support for multiple services (APIs) in one service config
- Update CORS selector display names with path suffix instead of index
- Support `additional_binding` options for gRPC-JSON transcoding
- Fix bug in OpenID Connect Discovery
- Add x509 support for JWT authentication
- Deprecated `--enable_backend_routing` flag; automatically set based on service configuration

# Release 2.1.0 07-01-2020

- Add support for healthz
- Support multiple services(apis) in one service config
- Solve the permission denied when open /etc/endpoints/service.json
- Improve error message if service config is not found
- Handle missing :path or :method headers in service control filter

# Release 2.0.0 17-12-2019