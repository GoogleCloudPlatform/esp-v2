// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package startproxy

import "strconv"

type flagKind int

const (
	kindStoreTrue flagKind = iota
	kindStoreFalse
	kindString
	kindOptString
	kindInt
	kindOptInt
	kindAppend
)

type flagDef struct {
	short   string
	longs   []string
	kind    flagKind
	choices []string
	apply   func(a *proxyArgs, val string) error
}

func buildFlagDefs() []*flagDef {
	setOptStr := func(target **string) func(*proxyArgs, string) error {
		return func(_ *proxyArgs, val string) error {
			v := val
			*target = &v
			return nil
		}
	}
	return []*flagDef{
		{
			short: "-s",
			longs: []string{"--service"},
			kind:  kindString,
			apply: func(a *proxyArgs, val string) error { a.Service = val; return nil },
		},
		{
			short: "-v",
			longs: []string{"--version"},
			kind:  kindString,
			apply: func(a *proxyArgs, val string) error { a.Version = val; return nil },
		},
		{
			longs: []string{"--service_json_path"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.ServiceJSONPath)(a, val) },
		},
		{
			short: "-a",
			longs: []string{"--backend"},
			kind:  kindString,
			apply: func(a *proxyArgs, val string) error { a.Backend = val; return nil },
		},
		{
			longs: []string{"--enable_backend_address_override"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.EnableBackendAddressOverride = true; return nil },
		},
		{
			longs: []string{"--listener_port"},
			kind:  kindOptInt,
			apply: func(a *proxyArgs, val string) error {
				n, err := strconv.Atoi(val)
				if err != nil {
					return err
				}
				a.ListenerPort = &n
				return nil
			},
		},
		{
			short: "-N",
			longs: []string{"--status_port", "--admin_port"},
			kind:  kindInt,
			apply: func(a *proxyArgs, val string) error {
				n, err := strconv.Atoi(val)
				if err != nil {
					return err
				}
				a.StatusPort = n
				return nil
			},
		},
		{
			longs: []string{"--ssl_server_cert_path"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.SslServerCertPath)(a, val) },
		},
		{
			longs: []string{"--ssl_server_cipher_suites"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.SslServerCipherSuites)(a, val) },
		},
		{
			longs: []string{"--ssl_server_root_cert_path"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.SslServerRootCertPath)(a, val) },
		},
		{
			longs: []string{"--ssl_backend_client_cert_path"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.SslBackendClientCertPath)(a, val) },
		},
		{
			longs: []string{"--ssl_backend_client_root_certs_file"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.SslBackendClientRootCertsFile)(a, val) },
		},
		{
			longs: []string{"--ssl_backend_client_cipher_suites"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.SslBackendClientCipherSuites)(a, val) },
		},
		{
			longs:   []string{"--ssl_minimum_protocol"},
			kind:    kindOptString,
			choices: []string{"TLSv1.0", "TLSv1.1", "TLSv1.2", "TLSv1.3"},
			apply:   func(a *proxyArgs, val string) error { return setOptStr(&a.SslMinimumProtocol)(a, val) },
		},
		{
			longs:   []string{"--ssl_maximum_protocol"},
			kind:    kindOptString,
			choices: []string{"TLSv1.0", "TLSv1.1", "TLSv1.2", "TLSv1.3"},
			apply:   func(a *proxyArgs, val string) error { return setOptStr(&a.SslMaximumProtocol)(a, val) },
		},
		{
			longs: []string{"--enable_strict_transport_security"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.EnableStrictTransportSecurity = true; return nil },
		},
		{
			longs: []string{"--generate_self_signed_cert"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.GenerateSelfSignedCert = true; return nil },
		},
		{
			short: "-z",
			longs: []string{"--healthz"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.Healthz)(a, val) },
		},
		{
			longs: []string{"--health_check_grpc_backend"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.HealthCheckGrpcBackend = true; return nil },
		},
		{
			longs: []string{"--health_check_grpc_backend_service"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.HealthCheckGrpcBackendService)(a, val) },
		},
		{
			longs: []string{"--health_check_grpc_backend_interval"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.HealthCheckGrpcBackendInterval)(a, val) },
		},
		{
			longs: []string{"--health_check_grpc_backend_no_traffic_interval"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error {
				return setOptStr(&a.HealthCheckGrpcBackendNoTrafficInterval)(a, val)
			},
		},
		{
			longs: []string{"--add_request_header"},
			kind:  kindAppend,
			apply: func(a *proxyArgs, val string) error { a.AddRequestHeader = append(a.AddRequestHeader, val); return nil },
		},
		{
			longs: []string{"--append_request_header"},
			kind:  kindAppend,
			apply: func(a *proxyArgs, val string) error {
				a.AppendRequestHeader = append(a.AppendRequestHeader, val)
				return nil
			},
		},
		{
			longs: []string{"--add_response_header"},
			kind:  kindAppend,
			apply: func(a *proxyArgs, val string) error {
				a.AddResponseHeader = append(a.AddResponseHeader, val)
				return nil
			},
		},
		{
			longs: []string{"--append_response_header"},
			kind:  kindAppend,
			apply: func(a *proxyArgs, val string) error {
				a.AppendResponseHeader = append(a.AppendResponseHeader, val)
				return nil
			},
		},
		{
			longs: []string{"--enable_operation_name_header"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.EnableOperationNameHeader = true; return nil },
		},
		{
			short:   "-R",
			longs:   []string{"--rollout_strategy"},
			kind:    kindString,
			choices: []string{"fixed", "managed"},
			apply:   func(a *proxyArgs, val string) error { a.RolloutStrategy = val; return nil },
		},
		{
			short: "-g",
			longs: []string{"--management"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.Management)(a, val) },
		},
		{
			longs: []string{"--cors_preset"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.CorsPreset)(a, val) },
		},
		{
			longs: []string{"--cors_allow_origin"},
			kind:  kindString,
			apply: func(a *proxyArgs, val string) error { a.CorsAllowOrigin = val; return nil },
		},
		{
			longs: []string{"--cors_allow_origin_regex"},
			kind:  kindString,
			apply: func(a *proxyArgs, val string) error { a.CorsAllowOriginRegex = val; return nil },
		},
		{
			longs: []string{"--cors_allow_methods"},
			kind:  kindString,
			apply: func(a *proxyArgs, val string) error { a.CorsAllowMethods = val; return nil },
		},
		{
			longs: []string{"--cors_allow_headers"},
			kind:  kindString,
			apply: func(a *proxyArgs, val string) error { a.CorsAllowHeaders = val; return nil },
		},
		{
			longs: []string{"--cors_expose_headers"},
			kind:  kindString,
			apply: func(a *proxyArgs, val string) error { a.CorsExposeHeaders = val; return nil },
		},
		{
			longs: []string{"--cors_allow_credentials"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.CorsAllowCredentials = true; return nil },
		},
		{
			longs: []string{"--cors_max_age"},
			kind:  kindString,
			apply: func(a *proxyArgs, val string) error { a.CorsMaxAge = val; return nil },
		},
		{
			longs: []string{"--check_metadata"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.CheckMetadata = true; return nil },
		},
		{
			longs: []string{"--underscores_in_headers"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.UnderscoresInHeaders = true; return nil },
		},
		{
			longs: []string{"--disable_normalize_path"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.DisableNormalizePath = true; return nil },
		},
		{
			longs: []string{"--disable_merge_slashes_in_path"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.DisableMergeSlashesInPath = true; return nil },
		},
		{
			longs: []string{"--disallow_escaped_slashes_in_path"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.DisallowEscapedSlashesInPath = true; return nil },
		},
		{
			longs: []string{"--envoy_use_remote_address"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.EnvoyUseRemoteAddress = true; return nil },
		},
		{
			longs: []string{"--envoy_xff_num_trusted_hops"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.EnvoyXffNumTrustedHops)(a, val) },
		},
		{
			longs: []string{"--envoy_connection_buffer_limit_bytes"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.EnvoyConnectionBufferLimitBytes)(a, val) },
		},
		{
			longs: []string{"--log_request_headers"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.LogRequestHeaders)(a, val) },
		},
		{
			longs: []string{"--log_response_headers"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.LogResponseHeaders)(a, val) },
		},
		{
			longs: []string{"--log_jwt_payloads"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.LogJwtPayloads)(a, val) },
		},
		{
			longs:   []string{"--service_control_network_fail_policy"},
			kind:    kindString,
			choices: []string{"open", "close"},
			apply:   func(a *proxyArgs, val string) error { a.ServiceControlNetworkFailPolicy = val; return nil },
		},
		{
			longs: []string{"--service_control_enable_api_key_uid_reporting"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.ServiceControlEnableAPIKeyUIDReporting = true; return nil },
		},
		{
			longs: []string{"--no-service_control_enable_api_key_uid_reporting"},
			kind:  kindStoreFalse,
			apply: func(a *proxyArgs, _ string) error { a.ServiceControlEnableAPIKeyUIDReporting = false; return nil },
		},
		{
			longs: []string{"--disable_jwks_async_fetch"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.DisableJwksAsyncFetch = true; return nil },
		},
		{
			longs: []string{"--jwks_async_fetch_fast_listener"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.JwksAsyncFetchFastListener = true; return nil },
		},
		{
			longs: []string{"--jwt_cache_size"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.JwtCacheSize)(a, val) },
		},
		{
			longs: []string{"--jwks_cache_duration_in_s"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.JwksCacheDurationInS)(a, val) },
		},
		{
			longs: []string{"--jwks_fetch_num_retries"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.JwksFetchNumRetries)(a, val) },
		},
		{
			longs: []string{"--jwks_fetch_retry_back_off_base_interval_ms"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.JwksFetchRetryBackOffBaseIntervalMs)(a, val) },
		},
		{
			longs: []string{"--jwks_fetch_retry_back_off_max_interval_ms"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.JwksFetchRetryBackOffMaxIntervalMs)(a, val) },
		},
		{
			longs: []string{"--jwt_pad_forward_payload_header"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.JwtPadForwardPayloadHeader = true; return nil },
		},
		{
			longs: []string{"--disable_jwt_audience_service_name_check"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.DisableJwtAudienceServiceNameCheck = true; return nil },
		},
		{
			longs: []string{"--http_request_timeout_s"},
			kind:  kindOptInt,
			apply: func(a *proxyArgs, val string) error {
				n, err := strconv.Atoi(val)
				if err != nil {
					return err
				}
				a.HTTPRequestTimeoutS = &n
				return nil
			},
		},
		{
			longs: []string{"--service_control_url"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.ServiceControlURL)(a, val) },
		},
		{
			longs: []string{"--service_control_check_timeout_ms"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.ServiceControlCheckTimeoutMs)(a, val) },
		},
		{
			longs: []string{"--service_control_quota_timeout_ms"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.ServiceControlQuotaTimeoutMs)(a, val) },
		},
		{
			longs: []string{"--service_control_report_timeout_ms"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.ServiceControlReportTimeoutMs)(a, val) },
		},
		{
			longs: []string{"--service_control_check_retries"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.ServiceControlCheckRetries)(a, val) },
		},
		{
			longs: []string{"--service_control_quota_retries"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.ServiceControlQuotaRetries)(a, val) },
		},
		{
			longs: []string{"--service_control_report_retries"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.ServiceControlReportRetries)(a, val) },
		},
		{
			longs: []string{"--backend_retry_ons"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.BackendRetryOns)(a, val) },
		},
		{
			longs: []string{"--backend_retry_on_status_codes"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.BackendRetryOnStatusCodes)(a, val) },
		},
		{
			longs: []string{"--backend_retry_num"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.BackendRetryNum)(a, val) },
		},
		{
			longs: []string{"--backend_per_try_timeout"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.BackendPerTryTimeout)(a, val) },
		},
		{
			longs: []string{"--access_log"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.AccessLog)(a, val) },
		},
		{
			longs: []string{"--access_log_format"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.AccessLogFormat)(a, val) },
		},
		{
			longs: []string{"--disable_tracing"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.DisableTracing = true; return nil },
		},
		{
			longs: []string{"--tracing_project_id"},
			kind:  kindString,
			apply: func(a *proxyArgs, val string) error { a.TracingProjectID = val; return nil },
		},
		{
			longs: []string{"--tracing_sample_rate"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.TracingSampleRate)(a, val) },
		},
		{
			longs: []string{"--disable_cloud_trace_auto_sampling"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.DisableCloudTraceAutoSampling = true; return nil },
		},
		{
			longs: []string{"--tracing_incoming_context"},
			kind:  kindString,
			apply: func(a *proxyArgs, val string) error { a.TracingIncomingContext = val; return nil },
		},
		{
			longs: []string{"--tracing_outgoing_context"},
			kind:  kindString,
			apply: func(a *proxyArgs, val string) error { a.TracingOutgoingContext = val; return nil },
		},
		{
			longs: []string{"--cloud_trace_url_override"},
			kind:  kindString,
			apply: func(a *proxyArgs, val string) error { a.CloudTraceURLOverride = val; return nil },
		},
		{
			longs: []string{"--non_gcp"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.NonGcp = true; return nil },
		},
		{
			longs: []string{"--service_account_key"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.ServiceAccountKey)(a, val) },
		},
		{
			longs: []string{"--enable_application_default_credentials"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.EnableApplicationDefaultCredentials = true; return nil },
		},
		{
			longs: []string{"--dns_resolver_addresses"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.DNSResolverAddresses)(a, val) },
		},
		{
			longs:   []string{"--backend_dns_lookup_family"},
			kind:    kindOptString,
			choices: []string{"auto", "v4only", "v6only", "v4preferred", "all"},
			apply:   func(a *proxyArgs, val string) error { return setOptStr(&a.BackendDNSLookupFamily)(a, val) },
		},
		{
			longs: []string{"--enable_debug"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.EnableDebug = true; return nil },
		},
		{
			longs: []string{"--transcoding_always_print_primitive_fields"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.TranscodingAlwaysPrintPrimitiveFields = true; return nil },
		},
		{
			longs: []string{"--transcoding_always_print_enums_as_ints"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.TranscodingAlwaysPrintEnumsAsInts = true; return nil },
		},
		{
			longs: []string{"--transcoding_stream_newline_delimited"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.TranscodingStreamNewlineDelimited = true; return nil },
		},
		{
			longs: []string{"--transcoding_case_insensitive_enum_parsing"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.TranscodingCaseInsensitiveEnumParsing = true; return nil },
		},
		{
			longs: []string{"--transcoding_preserve_proto_field_names"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.TranscodingPreserveProtoFieldNames = true; return nil },
		},
		{
			longs: []string{"--transcoding_ignore_query_parameters"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.TranscodingIgnoreQueryParameters)(a, val) },
		},
		{
			longs: []string{"--transcoding_ignore_unknown_query_parameters"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.TranscodingIgnoreUnknownQueryParameters = true; return nil },
		},
		{
			longs: []string{"--transcoding_query_parameters_disable_unescape_plus"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.TranscodingQueryParametersDisableUnescapePlus = true; return nil },
		},
		{
			longs: []string{"--transcoding_match_unregistered_custom_verb"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.TranscodingMatchUnregisteredCustomVerb = true; return nil },
		},
		{
			longs: []string{"--disallow_colon_in_wildcard_path_segment"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.DisallowColonInWildcardPathSegment = true; return nil },
		},
		{
			longs: []string{"--ads_named_pipe"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.AdsNamedPipe)(a, val) },
		},
		{
			longs: []string{"--envoy_extra_config_yaml"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.EnvoyExtraConfigYaml)(a, val) },
		},
		{
			longs: []string{"--enable_response_compression"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.EnableResponseCompression = true; return nil },
		},
		{
			longs: []string{"--enable_backend_routing"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.EnableBackendRouting = true; return nil },
		},
		{
			longs:   []string{"--backend_protocol"},
			kind:    kindOptString,
			choices: []string{"http1", "http2", "grpc"},
			apply:   func(a *proxyArgs, val string) error { return setOptStr(&a.BackendProtocol)(a, val) },
		},
		{
			longs: []string{"--http_port"},
			kind:  kindOptInt,
			apply: func(a *proxyArgs, val string) error {
				n, err := strconv.Atoi(val)
				if err != nil {
					return err
				}
				a.HTTPPort = &n
				return nil
			},
		},
		{
			longs: []string{"--http2_port"},
			kind:  kindOptInt,
			apply: func(a *proxyArgs, val string) error {
				n, err := strconv.Atoi(val)
				if err != nil {
					return err
				}
				a.HTTP2Port = &n
				return nil
			},
		},
		{
			longs: []string{"--ssl_port"},
			kind:  kindOptInt,
			apply: func(a *proxyArgs, val string) error {
				n, err := strconv.Atoi(val)
				if err != nil {
					return err
				}
				a.SslPort = &n
				return nil
			},
		},
		{
			longs: []string{"--dns"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.DNS)(a, val) },
		},
		{
			short: "-t",
			longs: []string{"--tls_mutual_auth"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.TLSMutualAuth = true; return nil },
		},
		{
			longs: []string{"--ssl_protocols"},
			kind:  kindAppend,
			apply: func(a *proxyArgs, val string) error { a.SslProtocols = append(a.SslProtocols, val); return nil },
		},
		{
			longs: []string{"--enable_grpc_backend_ssl"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.EnableGrpcBackendSsl = true; return nil },
		},
		{
			longs: []string{"--grpc_backend_ssl_root_certs_file"},
			kind:  kindString,
			apply: func(a *proxyArgs, val string) error { a.GrpcBackendSslRootCertsFile = val; return nil },
		},
		{
			longs: []string{"--ssl_client_cert_path"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.SslClientCertPath)(a, val) },
		},
		{
			longs: []string{"--ssl_client_root_certs_file"},
			kind:  kindOptString,
			apply: func(a *proxyArgs, val string) error { return setOptStr(&a.SslClientRootCertsFile)(a, val) },
		},
		{
			longs: []string{"--on_serverless"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.OnServerless = true; return nil },
		},
		{
			longs: []string{"--serverless"},
			kind:  kindStoreTrue,
			apply: func(a *proxyArgs, _ string) error { a.ServerlessEntrypoint = true; return nil },
		},
	}
}
