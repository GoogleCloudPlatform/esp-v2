# Release 2.13.0 08-07-2020

- Fix a rare use-after-free by creating FilterStats in client_cache (#212)
- Support api config versioning:  add v6 to api folder name and package name (#210)

# Release 2.12.0 24-06-2020

- Send error message as JSON format in response (#206)
- Add consumer-type and consumer-number headers (#200)
- Increase IMDS access token timeout to 30s (#198)
- More stats for Backend Auth, handle rejections properly. (#191)
- Add flag to control production prefix in generated headers (#184)

# Release 2.11.0 03-06-2020

- Fix grpc-web: move grpc-web filter in front of transcoder (#176)
- Migrate Envoy configs from api/v2 to api/v3 (#169, #175, #174)
- Ensure all consumer/producer errors are logged in stats (#166)
- Update envoy to 2020-05-26 (#165)
- Handle errors and test stats in Backend Routing (#164)
- Implement `denied_consumer_quota` and `denied_producer_error` stats (#163)

# Release 2.10.0 18-05-2020

- Implement `denied_consumer_blocked` and `denied_consumer_error` stats (#156)
- Implement `denied_control_plane_fault` (#155)
- Add statistical counter for check/allocateQuota/report call status (#151)
- Add latency in ServiceControl statistics (#146)

# Release 2.9.0 30-04-2020

- Forward `Authorization` header in JWT Authn filter (#141)
- Add flag: `dns_resolver_address` (#133)
- Add flags: `access_log` and `access_log_format` (#129)
- Covert non-5xx sidestream errors to 500 Internal Server Error (#122)
- Add flag: `underscores_in_headers` (#119)

# Release 2.8.0 07-04-2020

- Apply retry and `network_fail_open` for failed server response properly (#110)
- Align `--cloud_trace_url_override` with ESPv1 (#107)
- Support `generate_self_signed_certificate` (#105)
- Support websocket (#102)
- TLS support for grpcs backend (#95)

# Release 2.7.0 25-03-2020

- Add `X-Forwarded-Authorization` header (#90)
- Detect rolloutId change from Google Service Control (#83)
- Support custom JWT locations (#44)
- Add missing logEntries for report (#68)
- Support strict transport security (#77)
- Add mTLS support for upstream connection (#52)
- Add flag `--disable_cloud_trace_auto_sampling` (#81)
- Add flags for print options in JSON transcoder (#57)
- Add flags for unknown query parameters in JSON transcoder… (#79)
- Add flags for Admin control (#67)
- Add flags `ssl_minimum_protocol` and `ssl_maximum_protocol` for downstream (#58)
- Suppress envoy debug headers when `--enable_debug=false` (#82)

# Release 2.6.0 04-03-2020

- Signal ready only when all tokens are successfully retried (#37)
- Include SA email in fetched iam id token (#40)
- Add TLS support for downstream (#32)
- Support HTTP protocols (#36)
- Remove backend protocol flag (#31)
- Support auto binding for grpc transcoding (#33)

# Release 2.5.0 19-02-2020

- Auto-generate JWT Audience if no audiences are set in service config
- Retry after failed token fetch
- Handle `disable_auth` in endpoint service config
- Add `listener_port` and `http2_port` flags

# Release 2.4.0 06-02-2020

- Fix a crash on race condition during config change.
- Fix CORS bug by adding a CORS route for Envoy cors filter
- Support root path / in service config.
- Increase max program size limit for regex matching in Envoy
- Support http, https, grpc and grpcs scheme in BackendRule address
- Use IAM with delegation in backend auth and service control
- Support deadlines in BackendRule
- Add trace events when checking service control cache

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
