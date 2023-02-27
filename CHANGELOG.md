# Release 2.42.0 27-02-2023

- Add flag --disable_jwt_audience_service_name_check (#779)
- update go version to 1.18 (#780)

# Release 2.41.0 11-01-2023

- Increase ServiceControl Check cache duration (#773)
- Bug Fix for not using custom SA in flag "--service_account_key" for tracing (#772)
- Bug Fix for writing trace_id in the endpoint log when trace is disabled (#769)
- Add flag "--client_ip_from_forwarded_header" to enable extracting client IP from forward header (#764)
- Add keep-alive for upstream http2 connection for 30s interval and 10s timeout (#763)

# Release 2.40.0 05-12-2022

- Add two transcoding flags: "transcoding_stream_newline_delimited" and "transcoding_case_insensitive_enum_parsing" (#760)
- Enable jwt cache by default for jwt authentication. (#759)
- Update envoy to the top of the tree at 11/30/2022 (#758)
- Increase ID token refetch buffer (#757)
- Update to Envoy 1.24.0 (#755)
- Add two jwt_authn flags: "jwt_cache_size" and "jwks_async_fetch_fast_listener" (#753)

# Release 2.39.0 18-10-2022

- Add a flag "--backend_cluster_maximum_requests (#736)
- fix build_envoy_image failure (#737)
- Update gcloud_build_image script to be more safe. (#730)
- Enforce the default http rule (#725)
- Add option TranscodingRejectCollision for transcoding. Default is false. (#723)

# Release 2.38.0 26-07-2022

- Upgrade envoy to v1.23.0. (#720)
- Update to latest service_control_client (#713)

# Release 2.37.0 16-06-2022

- Added two new options to flag backend_dns_lookup_family and changed its default to "v4preferred"  (#705)
- Update base alpine image to 3.16 (#703)
- Update service-control-client to 05/31/2022 (#697)

# Release 2.36.0 25-04-2022

- Upgraded Envoy to 1.22.0 (#684)
- Added response gzip,brotli compression (#675)

# Release 2.35.0 22-03-2022

- Update Envoy to v1.21.1 (#670)
- Remove envoy runtime flag preserve_downstream_scheme (#667)
- Correctly escape user-provided regex paths (#664)
- Expose the `--config-yaml` envoy flag  (#662)

# Release 2.34.0 01-02-2022

- Expose flag `--ads_named_pipe` (#658)
- Update Envoy to v1.21.0 (#653)
- Support url templates with variables without wildcard (#654)

# Release 2.33.0 06-01-2022

- Support http.rules in the service config for gRPC transcoding (#640)
- Update help text for flag `--enable_strict_transport_security` (#642)
- Disallow colon in url wildcard path segment for route match (#639)
- Update docker base image to use alpine:3.15 (#638)

# Release 2.32.0 04-11-2021

- Support health check gRPC backend (#629)
- Support unescape plus in grpc transcoding (#630)
- Skip google discovery API during config generation. (#632)

# Release 2.31.0 19-10-2021

- Update Envoy to v1.20.0 (#625)
- Add openssl to the base alpine image (#623)
- Remove expired DST_Root_CA_X3.crt root ca (#617)
- Use alpine as base image (#611)
- Update gcloud_build_image again to support GAR (#613)
- Update application log format (#608)

# Release 2.30.3 15-09-2021

- Add X-User-Agent as default cors_allow_headers (#598)
- Add jwt_pad_forward_payload_header flag (#593)

# Release 2.30.2 01-09-2021

- Update Envoy to top of tree at 2021-08-24 (Envoy SHA `6f2726`) (#588)
- Improve gcsrunner remote dependency handling (#586)

# Release 2.30.1 12-08-2021

- Fix basic CORS with default cors_allow_origin=* (#579)

# Release 2.30.0 03-08-2021

- Update service_control_client_cxx with improved quota cache (#573)
- Unify route match policy with CORS filter (#575)
- Add jwks fetch retry flags for jwt authentication (#564)
- Update Envoy to 1.19.0 (#572)

# Release 2.29.1 14-07-2021

- Support downstream mTLS (#560)
- Support backendRetryOnStatusCodes (#554)

# Release 2.29.0 30-06-2021

- Restored the old way of setting scheme header according to upstream connection security level (#546)

# Release 2.28.0 15-06-2021
- Add flag to enable operation name header (#535)
- Enable jwks async fetch by default (#534)
- Update envoy to 06/09/2021, revert the breaking change of padding the forward jwt payload header(#532)

# Release 2.27.0 01-06-2021
- Fix overhead latency calculation for backend timeout (#505)
- API Gateway: Reduce backoff initial latency. (#520)

# Release 2.26.1 19-05-2021

No changes involve Cloud Endpoints users.

# Release 2.26.0 13-05-2021

- Upgrade to Envoy v1.18.3
- Add path normalization options for CVE-2021-29492 (#511)
- Add flag `--cors_max_age` to support set Access-Control-Max-Age response header (#502)
- Add perTryTimeout for doing retry when the upstream times out (#509)
- Support for "eu" zone via -z in gcloud_build_image (#490)
- Propagate trace context headers to Service Control Check (#487)

# Release 2.25.0 23-02-2021

- Add flags to add extra headers (#480)

# Release 2.24.0 01-02-2021

- Respond with HTTP 400 when required headers are omitted in CORS preflight request (#468)
- Allow backend address override (#464)
- Propagate trace ID to correlate access logs and traces (#463)

# Release 2.23.0 13-01-2021

- Automatically configure stream idle timeouts (#457)
- Add 405 directResponse in router (#451)
- Ensure service-config related errors are actionable (#450)
- Remove warning with empty requestTypeName (#448)
- Align behavior of `X-Forwarded-Authorization` and `X-Endpoint-API-UserInfo` headers (#447)
- Revamp status codes in access log (#444)

# Release 2.22.0 15-12-2020

- Enable fallback to x-cloud-trace-context by default (#439)
- backend retry config options (#436)
- Handle trailing backslash for match paths (#435) (#440)

# Release 2.21.0 02-12-2020

- Fix request header size (#425)
- Update envoy to jwt clock_skew change (#420)
- Support AuthenticationRule.allow_without_credential (#419)
- Enable traceparent trace context propagation by default (#416)
- Envoy changes to use DependencyErrorBehavior in TokenSubscriber (#406)

# Release 2.20.0 05-11-2020

- ConfigMgr changes: Jwt_authn use per-route config, remove path_matcher (#403)
- Use syntax parsing to generate route match instead of regex (#394)
- Replace snakeName with jsonName using syntax parsing (#393)
- Change configmgr to replace backend_routing with path_rewrite filter (#388)
- Switch ads to unix domain socket (#386)
- Path rewrite filter: add envoy filter related files. (#384)
- Add path_rewrite filter config and config_parser (#380)
- Change backend_auth to use per-route config (#376)
- Add flags to specify `cipher_suites` for TLS certificate (#379)
- Add option to disable OpenID Connect Discovery (#378)
- Fix path extraction for auto-gen cors methods (#377)
- Use per-route config for ServiceControl filter (#368)
- Support for complicated http_template in envoy route match (#370)
- Automatically disable backend auth on non-GCP platforms (#367)
- Add response code detail in service control report (#349)
- Support BackendRule deadline configuration in sidecar mode (#364)
- Enable route config for local backend in sidecar mode (#358)
- Increase API version to v9 (#360)

# Release 2.19.0 29-09-2020

- b/169095072: Fix path matcher misleading error message (#350)
- Deprecated flag --service_control_network_fail_open (#348)
- Add flag `--connection_buffer_limit_bytes` (#344)
- Enable gRPC when downstream is HTTP/1.1 (#336)
- Add grpc_metadata_scrubber filter (#328)

# Release 2.18.0 16-09-2020

- Add name prefix for backend/jwtProvider cluster (#330)
- Split specifying root certs for backend vs sidestream SSL clients (#325)
- Fix reading the remote client IP when deploying ESPv2 on Cloud Run (#318)
- Rich access logging of http request information (#316)
- Support IP in backend address (#323)
- For Cloud Run, report location with region instead of zone (#314)

# Release 2.17.0 02-09-2020

- Support GCP deployment with service account key (#308)
- Add default location in Report call for non_gcp (#307)
- Set API version in autogenerated CORS methods (#311)
- Better naming of healthz and cors operations (#302)
- Add ApiKeyState into report log entry (#305)
- Fix error with reporting invalid API Keys (#295)
- Remove grpc_stats filter (#299)
- Cleanup un-used metrics in report (#296, #300)
- Reduce noise of Envoy logs with `--enable_debug` (#293)

# Release 2.16.0 18-08-2020

- Add more port restrictions for usability (#281)
- Add retry mechanism on callgoogleapis when it comes into 429 (#278)

# Release 2.15.0 11-08-2020

- Support wildcards in envoy route matching with dynamic routing (#262)
- For local backend address, use HTTP as default schema (#263)
- Fix tracing sample rate (#249)
- Set `x-envoy-original-path` in backend routing filter for access logging (#241)
- Add api_key in filter_state for access logging (#233)
- Support ESP versions in `gcloud_build_image` (#229)

# Release 2.14.0 20-07-2020

- Config versioning v6 to v7 (#226)
- Move snake to json segment mapping to per-operation instead of global (#218)

# Release 2.13.0 08-07-2020

- Update envoy to 7/6/2020 (#217)
- Fix a rare use-after-free by creating FilterStats in client_cache (#212)
- Support api config versioning: add v6 to api folder name and package name (#210)

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
