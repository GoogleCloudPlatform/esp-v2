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

// The definitions below mirror start_proxy.py's argparse parser: option names,
// order, help text, choices and metavars are identical, so --help output and
// parse errors match the Python script.

// helpDescription is the description printed by --help.
const helpDescription = `
ESPv2 start-up script. This script starts Config Manager and Envoy.

The service name and config ID are optional. If not supplied, the Config Manager
fetches the service name and the config ID from the metadata service as
attributes "service_name" and "service_config_id".

ESPv2 relies on the metadata service to fetch access tokens for Google
services. If you deploy ESPv2 outside of Google Cloud environment, you need
to provide a service account credentials file by setting "creds_key"
environment variable or by passing "-k" flag to this script.
            `

// flagDefs returns the startproxy options bound to the fields of a, in
// start_proxy.py's declaration order (help output and ambiguity errors depend
// on it).
func flagDefs(a *proxyArgs) []*flagDef {
	return []*flagDef{
		{
			names: []string{"-h", "--help"},
			value: helpValue{},
			help:  "show this help message and exit",
		},
		{
			names:   []string{"-s", "--service"},
			value:   stringValue{&a.Service},
			metavar: "SERVICE",
			help: "Set the name of the Endpoints service. If omitted, ESPv2 contacts " +
				"the metadata service to fetch the service name.",
		},
		{
			names:   []string{"-v", "--version"},
			value:   stringValue{&a.Version},
			metavar: "VERSION",
			help: "Set the service config ID of the Endpoints service. It is required " +
				"if the \"fixed\" rollout strategy is used.",
		},
		{
			names:   []string{"--service_json_path"},
			value:   optStringValue{&a.ServiceJSONPath},
			metavar: "SERVICE_JSON_PATH",
			help: "Specify a path for ESPv2 to load the endpoint service config. With " +
				"this flag, ESPv2 will use \"fixed\" rollout strategy and following " +
				"flags will be ignored: --service, --version, and " +
				"--rollout_strategy.",
		},
		{
			names:   []string{"-a", "--backend"},
			value:   stringValue{&a.Backend},
			metavar: "BACKEND",
			help: "Specify the local backend application server address when using " +
				"ESPv2 as a sidecar. Default value is http://127.0.0.1:8082. Follow " +
				"the same format when setting manually. Valid schemes are `http`, " +
				"`https`, `grpc`, and `grpcs`. See the flag " +
				"--enable_backend_address_override for details on how ESPv2 decides " +
				"between using this flag vs using the backend addresses specified " +
				"in the service configuration.",
		},
		{
			names: []string{"--enable_backend_address_override"},
			value: trueValue{&a.EnableBackendAddressOverride},
			help: "Backend addresses can be specified using either the --backend flag " +
				"or the `backend.rule.address` field in the service configuration. " +
				"For OpenAPI users, note the `backend.rule.address` field is set by " +
				"the `address` field in the `x-google-backend` extension. " +
				"`backend.rule.address` is usually specified when routing to " +
				"different backends based on the route. By default, the " +
				"`backend.rule.address` will take priority over the --backend flag " +
				"for each individual operation. Enable this flag if you want the " +
				"--backend flag to take priority instead. This is useful if you are " +
				"developing on a local workstation. Then you use a the same " +
				"production service config but override the backend address via the " +
				"--backend flag for local testing. Note: Only the address will be " +
				"overridden. All other components of `backend.rule` will still " +
				"apply (deadlines, backend auth, path translation, etc).",
		},
		{
			names:   []string{"--listener_port"},
			value:   optIntValue{&a.ListenerPort},
			metavar: "LISTENER_PORT",
			help: "The port to accept downstream connections. It supports HTTP/1.x, " +
				"HTTP/2, and gRPC connections. Default is 8080",
		},
		{
			names:   []string{"-N", "--status_port", "--admin_port"},
			value:   intValue{&a.StatusPort},
			metavar: "STATUS_PORT",
			help: "Enable ESPv2 Envoy admin on this port. Please refer to " +
				"https://www.envoyproxy.io/docs/envoy/latest/operations/admin. By " +
				"default the admin port is disabled.",
		},
		{
			names:   []string{"--ssl_server_cert_path"},
			value:   optStringValue{&a.SslServerCertPath},
			metavar: "SSL_SERVER_CERT_PATH",
			help: "Proxy's server cert path. When configured, ESPv2 only accepts " +
				"HTTP/1.x and HTTP/2 secure connections on listener_port. Requires " +
				"the certificate and key files \"server.crt\" and \"server.key\" " +
				"within this path. Before using this feature, please make sure TLS " +
				"isn't terminated before ESPv2 in your deployment model. In " +
				"general, Cloud Run, GKE(GCLB enforced in ingress) and GCE with " +
				"GCLB configured terminates TLS before ESPv2. If that's the case, " +
				"please don't set up flag.",
		},
		{
			names:   []string{"--ssl_server_cipher_suites"},
			value:   optStringValue{&a.SslServerCipherSuites},
			metavar: "SSL_SERVER_CIPHER_SUITES",
			help: "Cipher suites to use for downstream connections as a " +
				"comma-separated list. Please refer to " +
				"https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/auth/common.proto#auth-tlsparameters",
		},
		{
			names:   []string{"--ssl_server_root_cert_path"},
			value:   optStringValue{&a.SslServerRootCertPath},
			metavar: "SSL_SERVER_ROOT_CERT_PATH",
			help: "The file path of root certificates that ESPv2 uses to verify " +
				"downstream client certificate. If not specified, ESPv2 doesn't " +
				"verify client certificates by default. Before using this feature, " +
				"please make sure mTLS isn't terminated before ESPv2 in your " +
				"deployment model. In general, Cloud Run, GKE with container-native " +
				"load balancing and GCE with GCLB configured terminates mTLS before " +
				"ESPv2. If that's the case, please don't set up flag.",
		},
		{
			names:   []string{"--ssl_backend_client_cert_path"},
			value:   optStringValue{&a.SslBackendClientCertPath},
			metavar: "SSL_BACKEND_CLIENT_CERT_PATH",
			help: "Proxy's client cert path. When configured, ESPv2 enables TLS " +
				"mutual authentication for HTTPS backends. Requires the certificate " +
				"and key files \"client.crt\" and \"client.key\" within this path.",
		},
		{
			names:   []string{"--ssl_backend_client_root_certs_file"},
			value:   optStringValue{&a.SslBackendClientRootCertsFile},
			metavar: "SSL_BACKEND_CLIENT_ROOT_CERTS_FILE",
			help: "The file path of root certificates that ESPv2 uses to verify " +
				"backend server certificate. If not specified, ESPv2 uses " +
				"'/etc/ssl/certs/ca-certificates.crt' by default.",
		},
		{
			names:   []string{"--ssl_backend_client_cipher_suites"},
			value:   optStringValue{&a.SslBackendClientCipherSuites},
			metavar: "SSL_BACKEND_CLIENT_CIPHER_SUITES",
			help: "Cipher suites to use for HTTPS backends as a comma-separated list. " +
				"Please refer to " +
				"https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/auth/common.proto#auth-tlsparameters",
		},
		{
			names:   []string{"--ssl_minimum_protocol"},
			value:   optStringValue{&a.SslMinimumProtocol},
			metavar: "{TLSv1.0,TLSv1.1,TLSv1.2,TLSv1.3}",
			choices: []string{"TLSv1.0", "TLSv1.1", "TLSv1.2", "TLSv1.3"},
			help: "Minimum TLS protocol version for client side connection. Please " +
				"refer to " +
				"https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/auth/cert.proto#common-tls-configuration.",
		},
		{
			names:   []string{"--ssl_maximum_protocol"},
			value:   optStringValue{&a.SslMaximumProtocol},
			metavar: "{TLSv1.0,TLSv1.1,TLSv1.2,TLSv1.3}",
			choices: []string{"TLSv1.0", "TLSv1.1", "TLSv1.2", "TLSv1.3"},
			help: "Maximum TLS protocol version for client side connection. Please " +
				"refer to " +
				"https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/auth/cert.proto#common-tls-configuration.",
		},
		{
			names: []string{"--enable_strict_transport_security"},
			value: trueValue{&a.EnableStrictTransportSecurity},
			help: "Enable HSTS (HTTP Strict Transport Security). " +
				"\"Strict-Transport-Security\" response header with value " +
				"\"max-age=31536000; includeSubdomains;\" is added for all " +
				"responses.",
		},
		{
			names: []string{"--generate_self_signed_cert"},
			value: trueValue{&a.GenerateSelfSignedCert},
			help: "Generate a self-signed certificate and key at start, then store " +
				"them in /tmp/ssl/endpoints/server.crt and " +
				"/tmp/ssl/endponts/server.key. This is useful when only a random " +
				"self-sign cert is needed to serve HTTPS requests. Generated " +
				"certificate will have Common Name \"localhost\" and valid for 10 " +
				"years.",
		},
		{
			names:   []string{"-z", "--healthz"},
			value:   optStringValue{&a.Healthz},
			metavar: "HEALTHZ",
			help: "Define a health checking endpoint on the same ports as the " +
				"application backend. For example, \"-z healthz\" makes ESPv2 " +
				"return code 200 for location \"/healthz\", instead of forwarding " +
				"the request to the backend. Please don't use any paths conflicting " +
				"with your normal requests. Default: not used. If the flag " +
				"\"--heath_check_grpc_backend\" is used, ESPv2 periodically checks " +
				"the backend gRPC Health service, its result will be reflected when " +
				"answering the health check calls.",
		},
		{
			names: []string{"--health_check_grpc_backend"},
			value: trueValue{&a.HealthCheckGrpcBackend},
			help: "If enabled, periodically check gRPC Health service to the backend " +
				"specified by the flag \"--backend\". The backend must use gRPC " +
				"protocol and implement the gRPC Health Checking Protocol defined " +
				"in " +
				"https://github.com/grpc/grpc/blob/master/doc/health-checking.md. " +
				"The health check endpoint enabled by the flag --healthz will " +
				"reflect the backend health check result too.",
		},
		{
			names:   []string{"--health_check_grpc_backend_service"},
			value:   optStringValue{&a.HealthCheckGrpcBackendService},
			metavar: "HEALTH_CHECK_GRPC_BACKEND_SERVICE",
			help: "Specify the service name when calling the backend gRPC Health " +
				"Checking Protocol. It only applied when the flag " +
				"\"--health_check_grpc_backend\" is used. It is optional, if not " +
				"set, default is empty. An empty service name is to query the gRPC " +
				"server overall health status.",
		},
		{
			names:   []string{"--health_check_grpc_backend_interval"},
			value:   optStringValue{&a.HealthCheckGrpcBackendInterval},
			metavar: "HEALTH_CHECK_GRPC_BACKEND_INTERVAL",
			help: "Specify the checking interval and timeout when checking the " +
				"backend gRPC Health service. It only applied when the flag " +
				"\"--health_check_grpc_backend\" is used. Default is 1 second. The " +
				"acceptable format is a sequence of decimal numbers, each with " +
				"optional fraction and a unit suffix, such as \"5s\", \"100ms\" or " +
				"\"2m\". Valid time units are \"m\" for minutes, \"s\" for seconds, " +
				"and \"ms\" for milliseconds.",
		},
		{
			names:   []string{"--health_check_grpc_backend_no_traffic_interval"},
			value:   optStringValue{&a.HealthCheckGrpcBackendNoTrafficInterval},
			metavar: "HEALTH_CHECK_GRPC_BACKEND_NO_TRAFFIC_INTERVAL",
			help: "Specify the checking interval to call the backend gRPC Health " +
				"service when at start up or the backend did not have any traffic. " +
				"Default is 60 seconds. It only applies when the flag " +
				"\"--health_check_grpc_backend\" is used. The acceptable format is " +
				"a sequence of decimal numbers, each with optional fraction and a " +
				"unit suffix, such as \"5s\", \"100ms\" or \"2m\". Valid time units " +
				"are \"m\" for minutes, \"s\" for seconds, and \"ms\" for " +
				"milliseconds.",
		},
		{
			names:   []string{"--add_request_header"},
			value:   appendValue{&a.AddRequestHeader},
			metavar: "ADD_REQUEST_HEADER",
			help: "Add a HTTP header to the request before sent to the upstream " +
				"backend. If the header is already in the request, its value will " +
				"be replaced with the new one. It supports envoy variable defined " +
				"at " +
				"https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_conn_man/headers#custom-request-response-headers. " +
				"This argument can be repeated multiple times to specify multiple " +
				"headers. For example: --add_request_header=key1=value1 " +
				"--add_request_header=key2=value2.",
		},
		{
			names:   []string{"--append_request_header"},
			value:   appendValue{&a.AppendRequestHeader},
			metavar: "APPEND_REQUEST_HEADER",
			help: "Append a HTTP header to the request before sent to the upstream " +
				"backend. If the header is already in the request, the new value " +
				"will be appended. It supports envoy variable defined at " +
				"https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_conn_man/headers#custom-request-response-headers. " +
				"This argument can be repeated multiple times to specify multiple " +
				"headers. For example: --append_request_header=key1=value1 " +
				"--append_request_header=key2=value2.",
		},
		{
			names:   []string{"--add_response_header"},
			value:   appendValue{&a.AddResponseHeader},
			metavar: "ADD_RESPONSE_HEADER",
			help: "Add a HTTP header to the response before sent to the downstream " +
				"client. If the header is already in the response, it will be " +
				"replaced with the new one. It supports envoy variable defined at " +
				"https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_conn_man/headers#custom-request-response-headers. " +
				"This argument can be repeated multiple times to specify multiple " +
				"headers. For example: --add_response_header=key1=value1 " +
				"--add_response_header=key2=value2.",
		},
		{
			names:   []string{"--append_response_header"},
			value:   appendValue{&a.AppendResponseHeader},
			metavar: "APPEND_RESPONSE_HEADER",
			help: "Append a HTTP header to the response before sent to the downstream " +
				"client. If the header is already in the response, the new one will " +
				"be appended. It supports envoy variable defined at " +
				"https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_conn_man/headers#custom-request-response-headers. " +
				"This argument can be repeated multiple times to specify multiple " +
				"headers. For example: --append_response_header=key1=value1 " +
				"--append_response_header=key2=value2.",
		},
		{
			names: []string{"--enable_operation_name_header"},
			value: trueValue{&a.EnableOperationNameHeader},
			help: "When enabled, ESPv2 will attach the operation name of the matched " +
				"route in the request to the backend. The operation name: - For " +
				"OpenAPI: Is derived from the `operationId` field. - For gRPC: Is " +
				"the fully-qualified name of the RPC. The header will be attached " +
				"with key `X-Endpoint-API-Operation-Name`. NOTE: We do NOT " +
				"recommend relying on this feature. There are no guarantees that " +
				"the operation name will remain consistent across multiple service " +
				"configuration IDs. NOTE: If you are using this feature with " +
				"OpenAPI, please only use the last segment of the operation name. " +
				"For example: " +
				"`1.echo_api_endpoints_cloudesf_testing_cloud_goog.EchoHeader` Your " +
				"backend should parse past the last `.` and only use `EchoHeader`. " +
				"NOTE: For OpenAPI, the operation name may not match the " +
				"`operationId` field exactly. For example, some sanitization may " +
				"occur that only allows alphanumeric characters. The exact " +
				"sanitization is internal implementation detail and is not " +
				"guaranteed to be consistent.",
		},
		{
			names:   []string{"-R", "--rollout_strategy"},
			value:   stringValue{&a.RolloutStrategy},
			metavar: "{fixed,managed}",
			choices: []string{"fixed", "managed"},
			help:    "The service config rollout strategy, [fixed|managed], Default value: fixed",
		},
		{
			names:   []string{"-g", "--management"},
			value:   optStringValue{&a.Management},
			metavar: "MANAGEMENT",
			hidden:  true,
		},
		{
			names:   []string{"--cors_preset"},
			value:   optStringValue{&a.CorsPreset},
			metavar: "CORS_PRESET",
			help: "Enables setting of CORS headers. This is useful when using a GRPC " +
				"backend, since a GRPC backend cannot set CORS headers. Specify one " +
				"of available presets to configure CORS response headers in nginx. " +
				"Defaults to no preset and therefore no CORS response headers. If " +
				"no preset is suitable for the use case, use the --nginx_config arg " +
				"to use a custom nginx config file. Available presets: - basic - " +
				"Assumes all location paths have the same CORS policy. Responds to " +
				"preflight OPTIONS requests with an empty 204, and the results of " +
				"preflight are allowed to be cached for up to 20 days (1728000 " +
				"seconds). See descriptions for args --cors_allow_origin, " +
				"--cors_allow_methods, --cors_allow_headers, --cors_expose_headers, " +
				"--cors_allow_credentials for more granular configurations. - " +
				"cors_with_regex - Same as basic preset, except that specifying " +
				"allowed origins in regular expression. See descriptions for args " +
				"--cors_allow_origin_regex, --cors_allow_methods, " +
				"--cors_allow_headers, --cors_expose_headers, " +
				"--cors_allow_credentials for more granular configurations.",
		},
		{
			names:   []string{"--cors_allow_origin"},
			value:   stringValue{&a.CorsAllowOrigin},
			metavar: "CORS_ALLOW_ORIGIN",
			help: "Only works when --cors_preset is 'basic'. Configures the CORS " +
				"header Access-Control-Allow-Origin. Defaults to \"*\" which allows " +
				"all origins.",
		},
		{
			names:   []string{"--cors_allow_origin_regex"},
			value:   stringValue{&a.CorsAllowOriginRegex},
			metavar: "CORS_ALLOW_ORIGIN_REGEX",
			help: "Only works when --cors_preset is 'cors_with_regex'. Configures the " +
				"whitelists of CORS header Access-Control-Allow-Origin with regular " +
				"expression.",
		},
		{
			names:   []string{"--cors_allow_methods"},
			value:   stringValue{&a.CorsAllowMethods},
			metavar: "CORS_ALLOW_METHODS",
			help: "Only works when --cors_preset is in use. Configures the CORS " +
				"header Access-Control-Allow-Methods. Defaults to allow common HTTP " +
				"methods.",
		},
		{
			names:   []string{"--cors_allow_headers"},
			value:   stringValue{&a.CorsAllowHeaders},
			metavar: "CORS_ALLOW_HEADERS",
			help: "Only works when --cors_preset is in use. Configures the CORS " +
				"header Access-Control-Allow-Headers. Defaults to allow common HTTP " +
				"headers.",
		},
		{
			names:   []string{"--cors_expose_headers"},
			value:   stringValue{&a.CorsExposeHeaders},
			metavar: "CORS_EXPOSE_HEADERS",
			help: "Only works when --cors_preset is in use. Configures the CORS " +
				"header Access-Control-Expose-Headers. Defaults to allow common " +
				"response headers.",
		},
		{
			names: []string{"--cors_allow_credentials"},
			value: trueValue{&a.CorsAllowCredentials},
			help: "Only works when --cors_preset is in use. Enable the CORS header " +
				"Access-Control-Allow-Credentials. By default, this header is " +
				"disabled.",
		},
		{
			names:   []string{"--cors_max_age"},
			value:   stringValue{&a.CorsMaxAge},
			metavar: "CORS_MAX_AGE",
			help: "Only works when --cors_preset is in use. Configures the CORS " +
				"header Access-Control-Max-Age. Defaults to 20 days (1728000 " +
				"seconds). The acceptable format is a sequence of decimal numbers, " +
				"each with optional fraction and a unit suffix, such as \"300m\", " +
				"\"1.5h\" or \"2h45m\". Valid time units are \"m\" for minutes, " +
				"\"h\" for hours.",
		},
		{
			names: []string{"--check_metadata"},
			value: trueValue{&a.CheckMetadata},
			help: "Enable fetching service name, service config ID and rollout " +
				"strategy from the metadata service.",
		},
		{
			names: []string{"--underscores_in_headers"},
			value: trueValue{&a.UnderscoresInHeaders},
			help: "Allow headers contain underscores to pass through. By default " +
				"ESPv2 rejects requests that have headers with underscores.",
		},
		{
			names: []string{"--disable_normalize_path"},
			value: trueValue{&a.DisableNormalizePath},
			help: "Disable normalization of the `path` HTTP header according to RFC " +
				"3986. It is recommended to keep this option enabled if your " +
				"backend performs path normalization by default. The following " +
				"table provides examples of the request `path` the backend will " +
				"receive from ESPv2 based on the configuration of this flag. " +
				"----------------------------------------------------------------- " +
				"| Request Path | Without Normalization | With Normalization | " +
				"----------------------------------------------------------------- " +
				"| /hello/../world | Rejected | /world | | /%4A | /%4A | /J | | " +
				"/%4a | /%4a | /J | " +
				"----------------------------------------------------------------- " +
				"By default, ESPv2 will normalize paths. Disable the feature only " +
				"if your traffic is affected by the behavior. Note: Following RFC " +
				"3986, this option does not unescape percent-encoded slash " +
				"characters. See flag `--disallow_escaped_slashes_in_path` to " +
				"enable this non-compliant behavior. Note: Case normalization from " +
				"RFC 3986 is not supported, even if this option is enabled. For " +
				"more details, see: " +
				"https://cloud.google.com/api-gateway/docs/path-templating",
		},
		{
			names: []string{"--disable_merge_slashes_in_path"},
			value: trueValue{&a.DisableMergeSlashesInPath},
			help: "Disable merging of adjacent slashes in the `path` HTTP header. It " +
				"is recommended to keep this option enabled if your backend " +
				"performs merging by default. The following table provides examples " +
				"of the request `path` the backend will receive from ESPv2 based on " +
				"the configuration of this flag. " +
				"----------------------------------------------------------------- " +
				"| Request Path | Without Normalization | With Normalization | " +
				"----------------------------------------------------------------- " +
				"| /hello//world | Rejected | /hello/world | | /hello/// | Rejected " +
				"| /hello | " +
				"----------------------------------------------------------------- " +
				"By default, ESPv2 will merge slashes. Disable the feature only if " +
				"your traffic is affected by the behavior. For more details, see: " +
				"https://cloud.google.com/api-gateway/docs/path-templating",
		},
		{
			names: []string{"--disallow_escaped_slashes_in_path"},
			value: trueValue{&a.DisallowEscapedSlashesInPath},
			help: "Disallows requests with escaped percent-encoded slash characters: " +
				"- %2F or %2f is treated as a / - %5C or %5c is treated as a When " +
				"enabled, the behavior depends on the protocol: - For OpenAPI " +
				"backends, request paths with unescaped percent-encoded slashes " +
				"will be automatically escaped via a redirect. - For gRPC backends, " +
				"request paths with unescaped percent-encoded slashes will be " +
				"rejected (gRPC does not support redirects). This option is **not** " +
				"RFC 3986 compliant, so it is turned off by default. If your " +
				"backend is **not** RFC 3986 compliant and escapes slashes, you " +
				"**must** enable this option in ESPv2. This will prevent against " +
				"path confusion attacks that result in security requirements not " +
				"being enforced. For more details, see: " +
				"https://cloud.google.com/api-gateway/docs/path-templating and " +
				"https://github.com/envoyproxy/envoy/security/advisories/GHSA-4987-27fx-x6cf",
		},
		{
			names: []string{"--envoy_use_remote_address"},
			value: trueValue{&a.EnvoyUseRemoteAddress},
			help: "Envoy HttpConnectionManager configuration, please refer to envoy " +
				"documentation for detailed information.",
		},
		{
			names:   []string{"--envoy_xff_num_trusted_hops"},
			value:   optStringValue{&a.EnvoyXffNumTrustedHops},
			metavar: "ENVOY_XFF_NUM_TRUSTED_HOPS",
			help: "Envoy HttpConnectionManager configuration, please refer to envoy " +
				"documentation for detailed information. The default value is 2 for " +
				"sidecar deployments and 0 for serverless deployments.",
		},
		{
			names:   []string{"--envoy_connection_buffer_limit_bytes"},
			value:   optStringValue{&a.EnvoyConnectionBufferLimitBytes},
			metavar: "ENVOY_CONNECTION_BUFFER_LIMIT_BYTES",
			help: "Configure the maximum amount of data that is buffered for each " +
				"request/response body, in bytes. If not set, default is decided by " +
				"Envoy. " +
				"https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/listener/v3/listener.proto",
		},
		{
			names:   []string{"--log_request_headers"},
			value:   optStringValue{&a.LogRequestHeaders},
			metavar: "LOG_REQUEST_HEADERS",
			help: "Log corresponding request headers through service control, " +
				"separated by comma. Example, when --log_request_headers=foo,bar, " +
				"endpoint log will have request_headers: " +
				"foo=foo_value;bar=bar_value if values are available;",
		},
		{
			names:   []string{"--log_response_headers"},
			value:   optStringValue{&a.LogResponseHeaders},
			metavar: "LOG_RESPONSE_HEADERS",
			help: "Log corresponding response headers through service control, " +
				"separated by comma. Example, when --log_response_headers=foo,bar, " +
				"endpoint log will have response_headers: " +
				"foo=foo_value;bar=bar_value if values are available;",
		},
		{
			names:   []string{"--log_jwt_payloads"},
			value:   optStringValue{&a.LogJwtPayloads},
			metavar: "LOG_JWT_PAYLOADS",
			help: "Log corresponding JWT JSON payload primitive fields through " +
				"service control, separated by comma. Example, when " +
				"--log_jwt_payload=sub,project_id, log will have jwt_payload: " +
				"sub=[SUBJECT];project_id=[PROJECT_ID] if the fields are available. " +
				"The value must be a primitive field, JSON objects and arrays will " +
				"not be logged.",
		},
		{
			names:   []string{"--service_control_network_fail_policy"},
			value:   stringValue{&a.ServiceControlNetworkFailPolicy},
			metavar: "{open,close}",
			choices: []string{"open", "close"},
			help: "Specify the policy to handle the request in case of network " +
				"failures when connecting to Google service control. If it is " +
				"`open`, the request will be allowed, otherwise, it will be " +
				"rejected. Default is `open`.",
		},
		{
			names: []string{"--service_control_enable_api_key_uid_reporting", "--no-service_control_enable_api_key_uid_reporting"},
			value: boolOptionalValue{&a.ServiceControlEnableAPIKeyUIDReporting},
			help:  "Enable when need to report api_key_uid in the telemetry report.",
		},
		{
			names: []string{"--disable_jwks_async_fetch"},
			value: trueValue{&a.DisableJwksAsyncFetch},
			help: "Disable fetching JWKS for JWT authentication to be done before " +
				"processing any requests. If disabled, JWKS fetching is done when " +
				"authenticating the JWT, the fetching will add to the request " +
				"processing latency. Default is enabled.",
		},
		{
			names: []string{"--jwks_async_fetch_fast_listener"},
			value: trueValue{&a.JwksAsyncFetchFastListener},
			help: "Only apply when --disable_jwks_async_fetch flag is not set. This " +
				"flag determines if the envoy will wait for jwks_async_fetch to " +
				"complete before binding the listener port. If false, it will wait. " +
				"Default is false.",
		},
		{
			names:   []string{"--jwt_cache_size"},
			value:   optStringValue{&a.JwtCacheSize},
			metavar: "JWT_CACHE_SIZE",
			help: "Specify JWT cache size, the number of unique JWT tokens in the " +
				"cache. The cache only stores verified good tokens. If 0, JWT cache " +
				"is disabled. It limits the memory usage. The cache used memory is " +
				"roughly (at most 4 KB data + 64 bytes metadata) per token. If not " +
				"specified, the default is 1000, which represents a max memory " +
				"usage of 4.35 MB.",
		},
		{
			names:   []string{"--jwks_cache_duration_in_s"},
			value:   optStringValue{&a.JwksCacheDurationInS},
			metavar: "JWKS_CACHE_DURATION_IN_S",
			help:    "Specify JWT public key cache duration in seconds. The default is 5 minutes.",
		},
		{
			names:   []string{"--jwks_fetch_num_retries"},
			value:   optStringValue{&a.JwksFetchNumRetries},
			metavar: "JWKS_FETCH_NUM_RETRIES",
			help:    "Specify the remote JWKS fetch retry policy's number of retries. default None.",
		},
		{
			names:   []string{"--jwks_fetch_retry_back_off_base_interval_ms"},
			value:   optStringValue{&a.JwksFetchRetryBackOffBaseIntervalMs},
			metavar: "JWKS_FETCH_RETRY_BACK_OFF_BASE_INTERVAL_MS",
			help: "Specify JWKS fetch retry exponential back off base interval in " +
				"milliseconds. default 200 ms if not set",
		},
		{
			names:   []string{"--jwks_fetch_retry_back_off_max_interval_ms"},
			value:   optStringValue{&a.JwksFetchRetryBackOffMaxIntervalMs},
			metavar: "JWKS_FETCH_RETRY_BACK_OFF_MAX_INTERVAL_MS",
			help: "Specify JWKS fetch retry exponential back off maximum interval in " +
				"milliseconds. default 32s if not set.",
		},
		{
			names: []string{"--jwt_pad_forward_payload_header"},
			value: trueValue{&a.JwtPadForwardPayloadHeader},
			help: "For the JWT in request, the JWT payload is forwarded to backend in " +
				"the `X-Endpoint-API-UserInfo` header by default. Normally JWT " +
				"based64 encode doesn\u2019t add padding. If this flag is true, the " +
				"header will be padded.",
		},
		{
			names: []string{"--disable_jwt_audience_service_name_check"},
			value: trueValue{&a.DisableJwtAudienceServiceNameCheck},
			help: "Normally JWT \"aud\" field is checked against audiences specified " +
				"in OpenAPI \"x-google-audiences\" field. This flag changes the " +
				"behaviour when the \"x-google-audiences\" is not specified. When " +
				"the \"x-google-audiences\" is not specified, normally the service " +
				"name is used to check the JWT \"aud\" field. If this flag is true, " +
				"the service name is not used, JWT \"aud\" field will not be " +
				"checked.",
		},
		{
			names:   []string{"--http_request_timeout_s"},
			value:   optIntValue{&a.HTTPRequestTimeoutS},
			metavar: "HTTP_REQUEST_TIMEOUT_S",
			help: "Set the timeout in seconds for all requests made to all external " +
				"services from ESPv2 (ie. Service Management, Instance Metadata " +
				"Server, etc.). This timeout does not apply to requests proxied to " +
				"the backend. Must be > 0 and the default is 30 seconds if not set.",
		},
		{
			names:   []string{"--service_control_url"},
			value:   optStringValue{&a.ServiceControlURL},
			metavar: "SERVICE_CONTROL_URL",
			help: "Set the url of service control server. The default is " +
				"\"https://servicecontrol.googleapis.com\" if not set.",
		},
		{
			names:   []string{"--service_control_check_timeout_ms"},
			value:   optStringValue{&a.ServiceControlCheckTimeoutMs},
			metavar: "SERVICE_CONTROL_CHECK_TIMEOUT_MS",
			help: "Set the timeout in millisecond for service control Check request. " +
				"Must be > 0 and the default is 1000 if not set. Default",
		},
		{
			names:   []string{"--service_control_quota_timeout_ms"},
			value:   optStringValue{&a.ServiceControlQuotaTimeoutMs},
			metavar: "SERVICE_CONTROL_QUOTA_TIMEOUT_MS",
			help: "Set the timeout in millisecond for service control Quota request. " +
				"Must be > 0 and the default is 1000 if not set.",
		},
		{
			names:   []string{"--service_control_report_timeout_ms"},
			value:   optStringValue{&a.ServiceControlReportTimeoutMs},
			metavar: "SERVICE_CONTROL_REPORT_TIMEOUT_MS",
			help: "Set the timeout in millisecond for service control Report request. " +
				"Must be > 0 and the default is 2000 if not set.",
		},
		{
			names:   []string{"--service_control_check_retries"},
			value:   optStringValue{&a.ServiceControlCheckRetries},
			metavar: "SERVICE_CONTROL_CHECK_RETRIES",
			help: "Set the retry times for service control Check request. Must be >= " +
				"0 and the default is 3 if not set.",
		},
		{
			names:   []string{"--service_control_quota_retries"},
			value:   optStringValue{&a.ServiceControlQuotaRetries},
			metavar: "SERVICE_CONTROL_QUOTA_RETRIES",
			help: "Set the retry times for service control Quota request. Must be >= " +
				"0 and the default is 1 if not set.",
		},
		{
			names:   []string{"--service_control_report_retries"},
			value:   optStringValue{&a.ServiceControlReportRetries},
			metavar: "SERVICE_CONTROL_REPORT_RETRIES",
			help: "Set the retry times for service control Report request. Must be >= " +
				"0 and the default is 5 if not set.",
		},
		{
			names:   []string{"--backend_retry_ons"},
			value:   optStringValue{&a.BackendRetryOns},
			metavar: "BACKEND_RETRY_ONS",
			help: "The conditions under which ESPv2 does retry on the backends. One " +
				"or more retryOn conditions can be specified by comma-separated " +
				"list. The default is `reset,connect-failure,refused-stream`. " +
				"Disable retry by setting this flag to empty. All the retryOn " +
				"conditions are defined in the " +
				"x-envoy-retry-on(https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/router_filter#x-envoy-retry-on) " +
				"and " +
				"x-envoy-retry-grpc-on(https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/router_filter#x-envoy-retry-on).",
		},
		{
			names:   []string{"--backend_retry_on_status_codes"},
			value:   optStringValue{&a.BackendRetryOnStatusCodes},
			metavar: "BACKEND_RETRY_ON_STATUS_CODES",
			help: "The list of backend http status codes will be retried, in addition " +
				"to the status codes enabled for retry through other retry policies " +
				"set in `--backend_retry_ons`. The format is a comma-delimited " +
				"String, like \"501, 503\".",
		},
		{
			names:   []string{"--backend_retry_num"},
			value:   optStringValue{&a.BackendRetryNum},
			metavar: "BACKEND_RETRY_NUM",
			help:    "The allowed number of retries. Must be >= 0 and defaults to 1.",
		},
		{
			names:   []string{"--backend_per_try_timeout"},
			value:   optStringValue{&a.BackendPerTryTimeout},
			metavar: "BACKEND_PER_TRY_TIMEOUT",
			help: "The backend timeout per retry attempt. Valid time units are " +
				"\"ns\", \"us\", \"ms\", \"s\", \"m\", \"h\". Please note the " +
				"`deadline` in the `x-google-backend` extension is the total time " +
				"wait for a full response from one request, including all retries. " +
				"If the flag is unspecified, ESPv2 will use the `deadline` in the " +
				"`x-google-backend` extension. Consequently, a request that times " +
				"out will not be retried as the total timeout budget would have " +
				"been exhausted.",
		},
		{
			names:   []string{"--access_log"},
			value:   optStringValue{&a.AccessLog},
			metavar: "ACCESS_LOG",
			help:    "Path to a local file to which the access log entries will be written.",
		},
		{
			names:   []string{"--access_log_format"},
			value:   optStringValue{&a.AccessLogFormat},
			metavar: "ACCESS_LOG_FORMAT",
			help: "String format to specify the format of access log. If unset, the " +
				"following format will be used. " +
				"https://www.envoyproxy.io/docs/envoy/latest/configuration/observability/access_log#default-format-string " +
				"For the detailed format grammar, please refer to the following " +
				"document. " +
				"https://www.envoyproxy.io/docs/envoy/latest/configuration/observability/access_log#format-strings",
		},
		{
			names: []string{"--disable_tracing"},
			value: trueValue{&a.DisableTracing},
			help: "Disable Stackdriver tracing. By default, tracing is enabled with 1 " +
				"out of 1000 requests being sampled. This sampling rate can be " +
				"changed with the --tracing_sample_rate flag.",
		},
		{
			names:   []string{"--tracing_project_id"},
			value:   stringValue{&a.TracingProjectID},
			metavar: "TRACING_PROJECT_ID",
			help:    "The Google project id for Stack driver tracing",
		},
		{
			names:   []string{"--tracing_sample_rate"},
			value:   optStringValue{&a.TracingSampleRate},
			metavar: "TRACING_SAMPLE_RATE",
			help: "Tracing sampling rate from 0.0 to 1.0. By default, 1 out of 1000 " +
				"requests are sampled. Cloud trace can still be enabled from " +
				"request HTTP headers with trace context regardless this flag " +
				"value.",
		},
		{
			names: []string{"--disable_cloud_trace_auto_sampling"},
			value: trueValue{&a.DisableCloudTraceAutoSampling},
			help:  "An alias to override --tracing_sample_rate to 0",
		},
		{
			names:   []string{"--tracing_incoming_context"},
			value:   stringValue{&a.TracingIncomingContext},
			metavar: "TRACING_INCOMING_CONTEXT",
			help: "Comma separated incoming trace contexts " +
				"(traceparent|grpc-trace-bin|x-cloud-trace-context). Note the order " +
				"matters. Default is 'traceparent,x-cloud-trace-context'. See " +
				"official documentation for more details: " +
				"https://cloud.google.com/endpoints/docs/openapi/tracing",
		},
		{
			names:   []string{"--tracing_outgoing_context"},
			value:   stringValue{&a.TracingOutgoingContext},
			metavar: "TRACING_OUTGOING_CONTEXT",
			help: "Comma separated outgoing trace contexts " +
				"(traceparent|grpc-trace-bin|x-cloud-trace-context). Note the order " +
				"matters. Default is 'traceparent,x-cloud-trace-context'. See " +
				"official documentation for more details: " +
				"https://cloud.google.com/endpoints/docs/openapi/tracing",
		},
		{
			names:   []string{"--cloud_trace_url_override"},
			value:   stringValue{&a.CloudTraceURLOverride},
			metavar: "CLOUD_TRACE_URL_OVERRIDE",
			help: "By default, traces will be sent to production Stackdriver Tracing. " +
				"If this is non-empty, ESPv2 will send traces to this gRPC service " +
				"instead. The url must be in gRPC format. " +
				"https://github.com/grpc/grpc/blob/master/doc/naming.md The gRPC " +
				"service must implement the cloud trace v2 RPCs. " +
				"https://github.com/googleapis/googleapis/tree/master/google/devtools/cloudtrace/v2",
		},
		{
			names: []string{"--non_gcp"},
			value: trueValue{&a.NonGcp},
			help: "By default, the proxy tries to talk to GCP metadata server to get " +
				"VM location in the first few requests. Setting this flag to true " +
				"to skip this step. This will also disable the following features: " +
				"- Backend authentication",
		},
		{
			names:   []string{"--service_account_key"},
			value:   optStringValue{&a.ServiceAccountKey},
			metavar: "SERVICE_ACCOUNT_KEY",
			help: "Use the service account key JSON file to access the service " +
				"control and the service management. You can also set " +
				"GOOGLE_APPLICATION_CREDENTIALS environment variable to the " +
				"location of the service account credentials JSON file. If the " +
				"option is omitted, the proxy contacts the metadata service to " +
				"fetch an access token.",
		},
		{
			names: []string{"--enable_application_default_credentials"},
			value: trueValue{&a.EnableApplicationDefaultCredentials},
			help: "Enable application default credentials to communicate with service " +
				"management APIs. Google auth libraries use a strategy called " +
				"Application Default Credentials (ADC) to detect and select " +
				"credentials based on environment or context automatically. For " +
				"more detail, see: https://google.aip.dev/auth/4110.",
		},
		{
			names:   []string{"--dns_resolver_addresses"},
			value:   optStringValue{&a.DNSResolverAddresses},
			metavar: "DNS_RESOLVER_ADDRESSES",
			help: "The addresses of dns resolvers. Each address should be in format " +
				"of IP_ADDR or IP_ADDR:PORT and they are separated by ';'. For the " +
				"IP_ADDR case, the default DNS port 52 will be used. (e.g., " +
				"--dns_resolver_addresses=127.0.0.1;127.0.0.2;127.0.0.3:8000) If " +
				"unset, will use the default resolver configured in " +
				"/etc/resolv.conf.",
		},
		{
			names:   []string{"--backend_dns_lookup_family"},
			value:   optStringValue{&a.BackendDNSLookupFamily},
			metavar: "{auto,v4only,v6only,v4preferred,all}",
			choices: []string{"auto", "v4only", "v6only", "v4preferred", "all"},
			help: "Define the dns lookup family for all backends. The options are " +
				"\"auto\", \"v4only\", \"v6only\", \"v4preferred\", and \"all\". " +
				"The default is \"v4preferred\". \"auto\" is a legacy name, it " +
				"behaves as \"v6preferred\".",
		},
		{
			names: []string{"--enable_debug"},
			value: trueValue{&a.EnableDebug},
			help: "Enables a variety of debug features in both Config Manager and " +
				"Envoy, such as: - Debug level per-request application logs in " +
				"Envoy - Debug level service configuration logs in Config Manager - " +
				"Debug HTTP response headers",
		},
		{
			names: []string{"--transcoding_always_print_primitive_fields"},
			value: trueValue{&a.TranscodingAlwaysPrintPrimitiveFields},
			help: "Whether to always print primitive fields for grpc-json " +
				"transcoding. By default primitive fields with default values will " +
				"be omitted in JSON output. For example, an int32 field set to 0 " +
				"will be omitted. Setting this flag to true will override the " +
				"default behavior and print primitive fields regardless of their " +
				"values. Defaults to false",
		},
		{
			names: []string{"--transcoding_always_print_enums_as_ints"},
			value: trueValue{&a.TranscodingAlwaysPrintEnumsAsInts},
			help: "Whether to always print enums as ints for grpc-json transcoding. " +
				"By default they are rendered as strings. Defaults to false.",
		},
		{
			names: []string{"--transcoding_stream_newline_delimited"},
			value: trueValue{&a.TranscodingStreamNewlineDelimited},
			help: "If true, use new line delimiter to separate response streaming " +
				"messages. If false, all response streaming messages will be " +
				"transcoded into a JSON array.",
		},
		{
			names: []string{"--transcoding_case_insensitive_enum_parsing"},
			value: trueValue{&a.TranscodingCaseInsensitiveEnumParsing},
			help: "Proto enum values are supposed to be in upper cases when used in " +
				"JSON. Set this flag to true if your JSON request uses non " +
				"uppercase enum values.",
		},
		{
			names: []string{"--transcoding_preserve_proto_field_names"},
			value: trueValue{&a.TranscodingPreserveProtoFieldNames},
			help: "Whether to preserve proto field names for grpc-json transcoding. " +
				"By default protobuf will generate JSON field names using the " +
				"json_name option, or lower camel case, in that order. Setting this " +
				"flag will preserve the original field names. Defaults to false",
		},
		{
			names:   []string{"--transcoding_ignore_query_parameters"},
			value:   optStringValue{&a.TranscodingIgnoreQueryParameters},
			metavar: "TRANSCODING_IGNORE_QUERY_PARAMETERS",
			help: "A list of query parameters(separated by comma) to be ignored for " +
				"transcoding method mapping in grpc-json transcoding. By default, " +
				"the transcoder filter will not transcode a request if there are " +
				"any unknown/invalid query parameters.",
		},
		{
			names: []string{"--transcoding_ignore_unknown_query_parameters"},
			value: trueValue{&a.TranscodingIgnoreUnknownQueryParameters},
			help: "Whether to ignore query parameters that cannot be mapped to a " +
				"corresponding protobuf field in grpc-json transcoding. Use this if " +
				"you cannot control the query parameters and do not know them " +
				"beforehand. Otherwise use ignored_query_parameters. Defaults to " +
				"false.",
		},
		{
			names: []string{"--transcoding_query_parameters_disable_unescape_plus"},
			value: trueValue{&a.TranscodingQueryParametersDisableUnescapePlus},
			help: "By default, unescape \"+\" to space when extracting variables in " +
				"the query parameters in grpc-json transcoding. This is to support " +
				"HTML 2.0<https://tools.ietf.org/html/rfc1866#section-8.2.1>. Set " +
				"this flag to true to disable this feature.",
		},
		{
			names: []string{"--transcoding_match_unregistered_custom_verb"},
			value: trueValue{&a.TranscodingMatchUnregisteredCustomVerb},
			help: "According to the http template[1], the custom verb is `\":\" " +
				"LITERAL` at the end of http template. For a request with " +
				"`/foo/bar:baz` and `:baz` is not registered in any url_template, " +
				"here is the behavior change - By default, `:baz` will not be " +
				"treated as custom verb, so it will match `/foo/{x=*}`. - If the " +
				"flag is provided, `:baz` is treated as custom verb, so it will NOT " +
				"match `/foo/{x=*}` since the template doesn't use any custom verb. " +
				"[1](https://github.com/googleapis/googleapis/blob/master/google/api/http.proto#L226-L231)",
		},
		{
			names: []string{"--disallow_colon_in_wildcard_path_segment"},
			value: trueValue{&a.DisallowColonInWildcardPathSegment},
			help: "Whether disallow colon in the url wildcard path segment for route " +
				"match. According to Google http url template spec[1], the literal " +
				"colon cannot be used in url wildcard path segment. This flag isn't " +
				"enabled for backward compatibility. " +
				"[1]https://github.com/googleapis/googleapis/blob/165280d3deea4d225a079eb5c34717b214a5b732/google/api/http.proto#L226-L252",
		},
		{
			names:   []string{"--ads_named_pipe"},
			value:   optStringValue{&a.AdsNamedPipe},
			metavar: "ADS_NAMED_PIPE",
			help: "Unix domain socket to use internally for xDs between config " +
				"manager and envoy. Only change if running multiple ESPv2 instances " +
				"on the same host.",
		},
		{
			names:   []string{"--envoy_extra_config_yaml"},
			value:   optStringValue{&a.EnvoyExtraConfigYaml},
			metavar: "ENVOY_EXTRA_CONFIG_YAML",
			help: "Specify extra bootstrap configuration for Envoy in the form of a " +
				"YAML string. Values will override and merge with any bootstrap " +
				"configuration loaded with '--config-path'. This will not override " +
				"any value which has been set dynamically. For more details see: " +
				"https://www.envoyproxy.io/docs/envoy/latest/operations/cli",
		},
		{
			names: []string{"--enable_response_compression"},
			value: trueValue{&a.EnableResponseCompression},
			help: "Enable both gzip and brotli compression for response data with " +
				"default envoy compression settings. Please see envoy document for " +
				"detail. " +
				"https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/compressor_filter.",
		},
		{
			names: []string{"--enable_backend_routing"},
			value: trueValue{&a.EnableBackendRouting},
			help: "=== DEPRECATED: This flag will automatically be enabled if needed, " +
				"so it does NOT need to be set manually. === Enable ESPv2 to route " +
				"requests according to the \"x-google-backend\" or \"backend\" " +
				"configuration",
		},
		{
			names:   []string{"--backend_protocol"},
			value:   optStringValue{&a.BackendProtocol},
			metavar: "{http1,http2,grpc}",
			choices: []string{"http1", "http2", "grpc"},
			help: "=== DEPRECATED: This flag will automatically be set based on the " +
				"scheme specified in the --backend flag. Overrides are no longer " +
				"needed. === Backend Protocol. Overrides the protocol in --backend. " +
				"Choices: [http1|http2|grpc]. Default value: http1.",
		},
		{
			names:   []string{"--http_port"},
			value:   optIntValue{&a.HTTPPort},
			metavar: "HTTP_PORT",
			help: "This flag is exactly same as --listener_port. It is added for " +
				"backward compatible for ESPv1 and will be deprecated. Please use " +
				"the flag --listener_port.",
		},
		{
			names:   []string{"--http2_port"},
			value:   optIntValue{&a.HTTP2Port},
			metavar: "HTTP2_PORT",
			help: "This flag is exactly same as --listener_port. It is added for " +
				"backward compatible for ESPv1 and will be deprecated. Please use " +
				"the flag --listener_port.",
		},
		{
			names:   []string{"--ssl_port"},
			value:   optIntValue{&a.SslPort},
			metavar: "SSL_PORT",
			help: "This flag added for backward compatible for ESPv1 and will be " +
				"deprecated. Please use the flags --listener_port and " +
				"--ssl_server_cert_path instead. When configured, ESPv2 accepts " +
				"HTTP/1.x and HTTP/2 secure connections on this port, Requires the " +
				"certificate and key files /etc/nginx/ssl/nginx.crt and " +
				"/etc/nginx/ssl/nginx.key",
		},
		{
			names:   []string{"--dns"},
			value:   optStringValue{&a.DNS},
			metavar: "DNS",
			help: "This flag is exactly same as --dns_resolver_addresses. This flag " +
				"is added for backward compatible for ESPv1 and will be deprecated. " +
				"Please use the flag --dns_resolver_addresses instead.",
		},
		{
			names: []string{"-t", "--tls_mutual_auth"},
			value: trueValue{&a.TLSMutualAuth},
			help: "This flag added for backward compatible for ESPv1 and will be " +
				"deprecated. Please use the flag --ssl_backend_client_cert_path " +
				"instead. Enable TLS mutual authentication for HTTPS backends. " +
				"Default value: Not enabled. Please provide the certificate and key " +
				"files /etc/nginx/ssl/backend.crt and /etc/nginx/ssl/backend.key.",
		},
		{
			names:   []string{"--ssl_protocols"},
			value:   appendValue{&a.SslProtocols},
			metavar: "SSL_PROTOCOLS",
			help: "This flag added for backward compatible for ESPv1 and will be " +
				"deprecated. Please use the flag --ssl_minimum_protocol and " +
				"--ssl_maximum_protocol instead. Enable the specified SSL " +
				"protocols. Please refer to " +
				"https://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_protocols. " +
				"The \"ssl_protocols\" argument can be repeated multiple times to " +
				"specify multiple SSL protocols (e.g., --ssl_protocols=TLSv1.1 " +
				"--ssl_protocols=TLSv1.2).",
		},
		{
			names: []string{"--enable_grpc_backend_ssl"},
			value: trueValue{&a.EnableGrpcBackendSsl},
			help: "This flag added for backward compatible for ESPv1 and will be " +
				"deprecated. Enable SSL for gRPC backend. ESPv2 auto enables SSL if " +
				"schema `grpcs` is detected.",
		},
		{
			names:   []string{"--grpc_backend_ssl_root_certs_file"},
			value:   stringValue{&a.GrpcBackendSslRootCertsFile},
			metavar: "GRPC_BACKEND_SSL_ROOT_CERTS_FILE",
			help: "This flag added for backward compatible for ESPv1 and will be " +
				"deprecated. ESPv2 uses `/etc/ssl/certs/ca-certificates.crt` by " +
				"default. The file path for gRPC backend SSL root certificates.",
		},
		{
			names:   []string{"--ssl_client_cert_path"},
			value:   optStringValue{&a.SslClientCertPath},
			metavar: "SSL_CLIENT_CERT_PATH",
			help: "This flag is renamed and deprecated for clarity. Use " +
				"`--ssl_backend_client_cert_path` instead.",
		},
		{
			names:   []string{"--ssl_client_root_certs_file"},
			value:   optStringValue{&a.SslClientRootCertsFile},
			metavar: "SSL_CLIENT_ROOT_CERTS_FILE",
			help: "This flag is renamed and deprecated for clarity. Use " +
				"`--ssl_backend_client_root_certs_file` instead.",
		},
		{
			names: []string{"--on_serverless"},
			value: trueValue{&a.OnServerless},
			help:  "When ESPv2 is started via the serverless image, this is true.",
		},
	}
}
