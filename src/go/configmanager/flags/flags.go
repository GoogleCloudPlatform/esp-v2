// Copyright 2019 Google LLC
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

// Package flags includes all API producer configurable settings.

package flags

import (
	"flag"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/commonflags"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/glog"
)

var (
	// These flags are kept in sync with options.ConfigGeneratorOptions.
	// When adding or changing default values, update options.DefaultConfigGeneratorOptions.

	// Cors related configurations.
	CorsAllowCredentials = flag.Bool("cors_allow_credentials", false, "whether include the Access-Control-Allow-Credentials header with the value true in responses or not")
	CorsAllowHeaders     = flag.String("cors_allow_headers", "", "set Access-Control-Allow-Headers to the specified HTTP headers")
	CorsAllowMethods     = flag.String("cors_allow_methods", "", "set Access-Control-Allow-Methods to the specified HTTP methods")
	CorsAllowOrigin      = flag.String("cors_allow_origin", "", "set Access-Control-Allow-Origin to a specific origin")
	CorsAllowOriginRegex = flag.String("cors_allow_origin_regex", "", "set Access-Control-Allow-Origin to a regular expression")
	CorsExposeHeaders    = flag.String("cors_expose_headers", "", "set Access-Control-Expose-Headers to the specified headers")
	CorsPreset           = flag.String("cors_preset", "", `enable CORS support, must be either "basic" or "cors_with_regex"`)

	// Backend routing configurations.
	BackendDnsLookupFamily = flag.String("backend_dns_lookup_family", "auto", `Define the dns lookup family for all backends. The options are "auto", "v4only" and "v6only". The default is "auto".`)

	// Envoy specific configurations.
	ClusterConnectTimeout = flag.Duration("cluster_connect_timeout", 20*time.Second, "cluster connect timeout in seconds")

	// Network related configurations.
	BackendAddress       = flag.String("backend_address", "http://127.0.0.1:8082", `The application server URI to which ESPv2 proxies requests.`)
	ListenerAddress      = flag.String("listener_address", "0.0.0.0", "listener socket ip address")
	ServiceManagementURL = flag.String("service_management_url", "https://servicemanagement.googleapis.com", "url of service management server")

	ListenerPort = flag.Int("listener_port", 8080, "listener port")
	Healthz      = flag.String("healthz", "", "path for health check of ESPv2 proxy itself")

	SslServerCertPath  = flag.String("ssl_server_cert_path", "", "Path to the certificate and key that ESPv2 uses to act as a HTTPS server")
	SslClientCertPath  = flag.String("ssl_client_cert_path", "", "Path to the certificate and key that ESPv2 uses to enable TLS mutual authentication for HTTPS backend")
	SslMinimumProtocol = flag.String("ssl_minimum_protocol", "", "Minimum TLS protocol version for Downstream connections.")
	SslMaximumProtocol = flag.String("ssl_maximum_protocol", "", "Maximum TLS protocol version for Downstream connections.")
	RootCertsPath      = flag.String("root_certs_path", util.DefaultRootCAPaths, "Path to the root certificates to make TLS connection.")
	EnableHSTS         = flag.Bool("enable_strict_transport_security", false, "Enable HSTS (HTTP Strict Transport Security).")

	// Flags for non_gcp deployment.
	ServiceAccountKey = flag.String("service_account_key", "", `Use the service account key JSON file to access the service control and the
	service management.  You can also set {creds_key} environment variable to the location of the service account credentials JSON file. If the option is
  omitted, the proxy contacts the metadata service to fetch an access token`)

	// Envoy configurations.
	EnvoyUseRemoteAddress  = flag.Bool("envoy_use_remote_address", false, "Envoy HttpConnectionManager configuration, please refer to envoy documentation for detailed information.")
	EnvoyXffNumTrustedHops = flag.Int("envoy_xff_num_trusted_hops", 2, "Envoy HttpConnectionManager configuration, please refer to envoy documentation for detailed information.")

	LogJwtPayloads = flag.String("log_jwt_payloads", "", `Log corresponding JWT JSON payload primitive fields through service control, separated by comma. Example, when --log_jwt_payload=sub,project_id, log
	will have jwt_payload: sub=[SUBJECT];project_id=[PROJECT_ID] if the fields are available. The value must be a primitive field, JSON objects and arrays will not be logged.`)
	LogRequestHeaders = flag.String("log_request_headers", "", `Log corresponding request headers through service control, separated by comma. Example, when --log_request_headers=
	foo,bar, endpoint log will have request_headers: foo=foo_value;bar=bar_value if values are available;`)
	LogResponseHeaders = flag.String("log_response_headers", "", `Log corresponding response headers through service control, separated by comma. Example, when --log_response_headers=
	foo,bar,endpoint log will have response_headers: foo=foo_value;bar=bar_value if values are available.`)
	MinStreamReportIntervalMs = flag.Uint64("min_stream_report_interval_ms", 0, `Minimum amount of time (milliseconds) between sending intermediate reports on a stream and the default is 10000 if not set.`)

	SuppressEnvoyHeaders = flag.Bool("suppress_envoy_headers", false, `Do not add any additional x-envoy- headers to requests or responses. This only affects the router filter
	generated *x-envoy-* headers, other Envoy filters and the HTTP connection manager may continue to set x-envoy- headers.`)

	ServiceControlNetworkFailOpen = flag.Bool("service_control_network_fail_open", true, ` In case of network failures when connecting to Google service control,
        the requests will be allowed if this flag is on. The default is on.`)

	JwksCacheDurationInS = flag.Int("jwks_cache_duration_in_s", 300, "Specify JWT public key cache duration in seconds. The default is 5 minutes.")

	ScCheckTimeoutMs  = flag.Int("service_control_check_timeout_ms", 0, `Set the timeout in millisecond for service control Check request. Must be > 0 and the default is 1000 if not set.`)
	ScQuotaTimeoutMs  = flag.Int("service_control_quota_timeout_ms", 0, `Set the timeout in millisecond for service control Quota request. Must be > 0 and the default is 1000 if not set.`)
	ScReportTimeoutMs = flag.Int("service_control_report_timeout_ms", 0, `Set the timeout in millisecond for service control Report request. Must be > 0 and the default is 2000 if not set.`)

	ScCheckRetries  = flag.Int("service_control_check_retries", -1, `Set the retry times for service control Check request. Must be >= 0 and the default is 3 if not set.`)
	ScQuotaRetries  = flag.Int("service_control_quota_retries", -1, `Set the retry times for service control Quota request. Must be >= 0 and the default is 1 if not set.`)
	ScReportRetries = flag.Int("service_control_report_retries", -1, `Set the retry times for service control Report request. Must be >= 0 and the default is 5 if not set.`)

	ComputePlatformOverride = flag.String("compute_platform_override", "", "the overridden platform where the proxy is running at")

	// Flags for testing purpose.
	SkipJwtAuthnFilter       = flag.Bool("skip_jwt_authn_filter", false, "skip jwt authn filter, for test purpose")
	SkipServiceControlFilter = flag.Bool("skip_service_control_filter", false, "skip service control filter, for test purpose")

	TranscodingAlwaysPrintPrimitiveFields   = flag.Bool("transcoding_always_print_primitive_fields", false, "Whether to always print primitive fields for grpc-json transcoding")
	TranscodingAlwaysPrintEnumsAsInts       = flag.Bool("transcoding_always_print_enums_as_ints", false, "Whether to always print enums as ints for grpc-json transcoding")
	TranscodingPreserveProtoFieldNames      = flag.Bool("transcoding_preserve_proto_field_names", false, "Whether to preserve proto field names for grpc-json transcoding")
	TranscodingIgnoreQueryParameters        = flag.String("transcoding_ignore_query_parameters", "", "A list of query parameters(separated by comma) to be ignored for transcoding method mapping in grpc-json transcoding.")
	TranscodingIgnoreUnknownQueryParameters = flag.Bool("transcoding_ignore_unknown_query_parameters", false, "Whether to ignore query parameters that cannot be mapped to a corresponding protobuf field in grpc-json transcoding.")
)

func EnvoyConfigOptionsFromFlags() options.ConfigGeneratorOptions {
	opts := options.ConfigGeneratorOptions{
		CommonOptions:                           commonflags.DefaultCommonOptionsFromFlags(),
		BackendAddress:                          *BackendAddress,
		ComputePlatformOverride:                 *ComputePlatformOverride,
		CorsAllowCredentials:                    *CorsAllowCredentials,
		CorsAllowHeaders:                        *CorsAllowHeaders,
		CorsAllowMethods:                        *CorsAllowMethods,
		CorsAllowOrigin:                         *CorsAllowOrigin,
		CorsAllowOriginRegex:                    *CorsAllowOriginRegex,
		CorsExposeHeaders:                       *CorsExposeHeaders,
		CorsPreset:                              *CorsPreset,
		BackendDnsLookupFamily:                  *BackendDnsLookupFamily,
		ClusterConnectTimeout:                   *ClusterConnectTimeout,
		ListenerAddress:                         *ListenerAddress,
		ServiceManagementURL:                    *ServiceManagementURL,
		ListenerPort:                            *ListenerPort,
		Healthz:                                 *Healthz,
		RootCertsPath:                           *RootCertsPath,
		SslServerCertPath:                       *SslServerCertPath,
		SslClientCertPath:                       *SslClientCertPath,
		SslMinimumProtocol:                      *SslMinimumProtocol,
		SslMaximumProtocol:                      *SslMaximumProtocol,
		EnableHSTS:                              *EnableHSTS,
		ServiceAccountKey:                       *ServiceAccountKey,
		SkipJwtAuthnFilter:                      *SkipJwtAuthnFilter,
		SkipServiceControlFilter:                *SkipServiceControlFilter,
		EnvoyUseRemoteAddress:                   *EnvoyUseRemoteAddress,
		EnvoyXffNumTrustedHops:                  *EnvoyXffNumTrustedHops,
		LogJwtPayloads:                          *LogJwtPayloads,
		LogRequestHeaders:                       *LogRequestHeaders,
		LogResponseHeaders:                      *LogResponseHeaders,
		MinStreamReportIntervalMs:               *MinStreamReportIntervalMs,
		SuppressEnvoyHeaders:                    *SuppressEnvoyHeaders,
		ServiceControlNetworkFailOpen:           *ServiceControlNetworkFailOpen,
		JwksCacheDurationInS:                    *JwksCacheDurationInS,
		ScCheckTimeoutMs:                        *ScCheckTimeoutMs,
		ScQuotaTimeoutMs:                        *ScQuotaTimeoutMs,
		ScReportTimeoutMs:                       *ScReportTimeoutMs,
		ScCheckRetries:                          *ScCheckRetries,
		ScQuotaRetries:                          *ScQuotaRetries,
		ScReportRetries:                         *ScReportRetries,
		TranscodingAlwaysPrintPrimitiveFields:   *TranscodingAlwaysPrintPrimitiveFields,
		TranscodingAlwaysPrintEnumsAsInts:       *TranscodingAlwaysPrintEnumsAsInts,
		TranscodingPreserveProtoFieldNames:      *TranscodingPreserveProtoFieldNames,
		TranscodingIgnoreQueryParameters:        *TranscodingIgnoreQueryParameters,
		TranscodingIgnoreUnknownQueryParameters: *TranscodingIgnoreUnknownQueryParameters,
	}

	glog.Infof("Config Generator options: %+v", opts)
	return opts
}
