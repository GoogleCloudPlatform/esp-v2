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

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	// defaultConfigDir is the directory where bootstrap config file is written.
	defaultConfigDir = "/tmp"

	// bootstrapConfig is the bootstrap config file name.
	bootstrapConfig = "/bootstrap.json"

	// defaultListenerPort is the default listener port.
	defaultListenerPort = 8080

	// defaultBackend is the default backend address.
	defaultBackend = "http://127.0.0.1:8082"

	// defaultRolloutStrategy is the default rollout strategy.
	defaultRolloutStrategy = "fixed"

	// googleCredsEnv is the Google default application credentials environment variable.
	googleCredsEnv = "GOOGLE_APPLICATION_CREDENTIALS"

	// serverlessPlatform is the platform string when running on serverless.
	serverlessPlatform = "Cloud Run(ESPv2)"

	// serverlessXffNumTrustedHops is the default XFF num trusted hops on serverless.
	serverlessXffNumTrustedHops = 0
)

// genBootstrapConf generates the command-line slice to invoke bin/bootstrap.
func (l *launcher) genBootstrapConf(args *proxyArgs) []string {
	cmd := []string{l.bootstrapBin, "--logtostderr"}
	cmd = append(cmd, "--admin_port", strconv.Itoa(args.StatusPort))
	if args.HTTPRequestTimeoutS != nil && *args.HTTPRequestTimeoutS != 0 {
		cmd = append(cmd, "--http_request_timeout_s", strconv.Itoa(*args.HTTPRequestTimeoutS))
	}
	if args.AdsNamedPipe != nil && *args.AdsNamedPipe != "" {
		cmd = append(cmd, "--ads_named_pipe", *args.AdsNamedPipe)
	}
	bootstrapFile := defaultConfigDir + bootstrapConfig
	cmd = append(cmd, bootstrapFile)
	return cmd
}

// enforceConflictArgs checks for conflicting or missing flags and updates defaults when needed.
func (l *launcher) enforceConflictArgs(args *proxyArgs) error {
	if args.RolloutStrategy == "managed" {
		if args.Version != "" {
			return fmt.Errorf("Flag --version cannot be used if --rollout_strategy=managed.")
		}
		if args.ServiceJSONPath != nil && *args.ServiceJSONPath != "" {
			return fmt.Errorf("Flag -R or --rollout_strategy must be fixed with --service_json_path.")
		}
	} else {
		if args.Version == "" && (args.ServiceJSONPath == nil || *args.ServiceJSONPath == "") {
			return fmt.Errorf("Flag --version is required if --rollout_strategy=fixed.")
		}
	}

	if args.ServiceJSONPath != nil && *args.ServiceJSONPath != "" {
		if args.Service != "" {
			return fmt.Errorf("Flag --service cannot be used together with --service_json_path.")
		}
		if args.Version != "" {
			return fmt.Errorf("Flag --version cannot be used together with --service_json_path.")
		}
	}

	if args.NonGcp {
		if (args.ServiceAccountKey == nil || *args.ServiceAccountKey == "") && !args.EnableApplicationDefaultCredentials {
			return fmt.Errorf("If --non_gcp is specified, --service_account_key or --enable_application_default_credentials has to be specified, or GOOGLE_APPLICATION_CREDENTIALS has to set in os.environ.")
		}
		if args.ServiceAccountKey != nil && *args.ServiceAccountKey != "" && args.EnableApplicationDefaultCredentials {
			return fmt.Errorf("Only one of --service_account_key or --enable_application_default_credentials can be supplied for credentials at once.")
		}
		hasOtelProjectID := otelAttrsHaveProjectID(os.Getenv(otelResourceAttributesEnv))
		if args.TracingProjectID == "" && !hasOtelProjectID {
			if saProjectID := projectIDFromServiceAccountKey(serviceAccountKeyPath(args)); saProjectID != "" {
				args.TracingProjectID = saProjectID
				l.setupOtelResourceAttributes(args)
			}
		}
		if args.TracingProjectID == "" && !hasOtelProjectID {
			// for non gcp case, disable tracing if tracing project id is not provided.
			args.DisableTracing = true
		}
	}

	if (args.AccessLog == nil || *args.AccessLog == "") && (args.AccessLogFormat != nil && *args.AccessLogFormat != "") {
		return fmt.Errorf("Flag --access_log_format has to be used together with --access_log.")
	}

	if args.SslPort != nil && *args.SslPort != 0 && args.SslServerCertPath != nil && *args.SslServerCertPath != "" {
		return fmt.Errorf("Flag --ssl_port is going to be deprecated, please use --ssl_server_cert_path only.")
	}
	if args.TLSMutualAuth && ((args.SslBackendClientCertPath != nil && *args.SslBackendClientCertPath != "") || (args.SslClientCertPath != nil && *args.SslClientCertPath != "")) {
		return fmt.Errorf("Flag --tls_mutual_auth is going to be deprecated, please use --ssl_backend_client_cert_path only.")
	}
	if ((args.SslBackendClientRootCertsFile != nil && *args.SslBackendClientRootCertsFile != "") || (args.SslClientRootCertsFile != nil && *args.SslClientRootCertsFile != "")) && args.EnableGrpcBackendSsl {
		return fmt.Errorf("Flag --enable_grpc_backend_ssl are going to be deprecated, please use --ssl_backend_client_root_certs_file only.")
	}
	if args.GenerateSelfSignedCert && args.SslServerCertPath != nil && *args.SslServerCertPath != "" {
		return fmt.Errorf("Flag --generate_self_signed_cert and --ssl_server_cert_path cannot be used simutaneously.")
	}

	var portFlags []string
	portNum := defaultListenerPort
	if args.HTTPPort != nil && *args.HTTPPort != 0 {
		portFlags = append(portFlags, "--http_port")
		portNum = *args.HTTPPort
	}
	if args.HTTP2Port != nil && *args.HTTP2Port != 0 {
		portFlags = append(portFlags, "--http2_port")
		portNum = *args.HTTP2Port
	}
	if args.ListenerPort != nil && *args.ListenerPort != 0 {
		portFlags = append(portFlags, "--listener_port")
		portNum = *args.ListenerPort
	}
	if args.SslPort != nil && *args.SslPort != 0 {
		portFlags = append(portFlags, "--ssl_port")
		portNum = *args.SslPort
	}

	if len(portFlags) > 1 {
		return fmt.Errorf("Multiple port flags %s are not allowed, use only the --listener_port flag", strings.Join(portFlags, ","))
	} else if portNum < 1024 {
		return fmt.Errorf("Port %d is a privileged port. For security purposes, the ESPv2 container cannot bind to it. Use any port above 1024 instead.", portNum)
	}

	if len(args.SslProtocols) > 0 && ((args.SslMinimumProtocol != nil && *args.SslMinimumProtocol != "") || (args.SslMaximumProtocol != nil && *args.SslMaximumProtocol != "")) {
		return fmt.Errorf("Flag --ssl_protocols is going to be deprecated, please use --ssl_minimum_protocol and --ssl_maximum_protocol.")
	}

	if args.TranscodingIgnoreQueryParameters != nil && *args.TranscodingIgnoreQueryParameters != "" && args.TranscodingIgnoreUnknownQueryParameters {
		return fmt.Errorf("Flag --transcoding_ignore_query_parameters cannot be used together with --transcoding_ignore_unknown_query_parameters.")
	}

	if args.DNSResolverAddresses != nil && *args.DNSResolverAddresses != "" && args.DNS != nil && *args.DNS != "" {
		return fmt.Errorf("Flag --dns_resolver_addresses cannot be used together with together with --dns.")
	}

	if args.SslBackendClientCertPath != nil && *args.SslBackendClientCertPath != "" && args.SslClientCertPath != nil && *args.SslClientCertPath != "" {
		return fmt.Errorf("Flag --ssl_client_cert_path is renamed to --ssl_backend_client_cert_path, only use the latter flag.")
	}

	if args.SslBackendClientRootCertsFile != nil && *args.SslBackendClientRootCertsFile != "" && args.SslClientRootCertsFile != nil && *args.SslClientRootCertsFile != "" {
		return fmt.Errorf("Flag --ssl_client_root_certs_file is renamed to --ssl_backend_client_root_certs_file, only use the latter flag.")
	}

	if args.HealthCheckGrpcBackend && !strings.HasPrefix(args.Backend, "grpc") {
		return fmt.Errorf("Flag --health_check_grpc_backend requires the flag --backend to use grpc scheme.")
	}
	if !args.HealthCheckGrpcBackend && args.HealthCheckGrpcBackendInterval != nil && *args.HealthCheckGrpcBackendInterval != "" {
		return fmt.Errorf("Flag --health_check_grpc_backend_interval requires the flag --health_check_grpc_backend to be used.")
	}
	if !args.HealthCheckGrpcBackend && args.HealthCheckGrpcBackendService != nil && *args.HealthCheckGrpcBackendService != "" {
		return fmt.Errorf("Flag --health_check_grpc_backend_service requires the flag --health_check_grpc_backend to be used.")
	}
	if !args.HealthCheckGrpcBackend && args.HealthCheckGrpcBackendNoTrafficInterval != nil && *args.HealthCheckGrpcBackendNoTrafficInterval != "" {
		return fmt.Errorf("Flag --health_check_grpc_backend_no_traffic_interval requires the flag --health_check_grpc_backend to be used.")
	}

	return nil
}

// genProxyConfig builds the command-line slice for bin/configmanager.
func (l *launcher) genProxyConfig(args *proxyArgs) ([]string, error) {
	if args.ServiceAccountKey == nil {
		if envVal, ok := os.LookupEnv(googleCredsEnv); ok {
			args.ServiceAccountKey = &envVal
		}
	} else {
		if _, ok := os.LookupEnv(googleCredsEnv); !ok {
			_ = os.Setenv(googleCredsEnv, *args.ServiceAccountKey)
		}
	}

	if err := l.enforceConflictArgs(args); err != nil {
		return nil, err
	}

	proxyConf := []string{
		l.configManagerBin,
		"--logtostderr",
		"--rollout_strategy", args.RolloutStrategy,
	}

	if !strings.Contains(args.Backend, "://") {
		proxyConf = append(proxyConf, "--backend_address", "http://"+args.Backend)
	} else {
		proxyConf = append(proxyConf, "--backend_address", args.Backend)
	}

	if args.Healthz != nil && *args.Healthz != "" {
		proxyConf = append(proxyConf, "--healthz", *args.Healthz)
	}

	if args.HealthCheckGrpcBackend {
		proxyConf = append(proxyConf, "--health_check_grpc_backend")
		if args.HealthCheckGrpcBackendService != nil && *args.HealthCheckGrpcBackendService != "" {
			proxyConf = append(proxyConf, "--health_check_grpc_backend_service", *args.HealthCheckGrpcBackendService)
		}
		if args.HealthCheckGrpcBackendInterval != nil && *args.HealthCheckGrpcBackendInterval != "" {
			proxyConf = append(proxyConf, "--health_check_grpc_backend_interval", *args.HealthCheckGrpcBackendInterval)
		}
		if args.HealthCheckGrpcBackendNoTrafficInterval != nil && *args.HealthCheckGrpcBackendNoTrafficInterval != "" {
			proxyConf = append(proxyConf, "--health_check_grpc_backend_no_traffic_interval", *args.HealthCheckGrpcBackendNoTrafficInterval)
		}
	}

	if args.EnableDebug {
		proxyConf = append(proxyConf, "--v", "1")
	} else {
		proxyConf = append(proxyConf, "--v", "0")
	}

	if args.EnvoyXffNumTrustedHops != nil && *args.EnvoyXffNumTrustedHops != "" {
		proxyConf = append(proxyConf, "--envoy_xff_num_trusted_hops", *args.EnvoyXffNumTrustedHops)
	} else if args.OnServerless {
		proxyConf = append(proxyConf, "--envoy_xff_num_trusted_hops", strconv.Itoa(serverlessXffNumTrustedHops))
	}

	if args.DisableJwksAsyncFetch {
		proxyConf = append(proxyConf, "--disable_jwks_async_fetch")
	}
	if args.JwksAsyncFetchFastListener {
		proxyConf = append(proxyConf, "--jwks_async_fetch_fast_listener")
	}
	if args.JwtCacheSize != nil && *args.JwtCacheSize != "" {
		proxyConf = append(proxyConf, "--jwt_cache_size", *args.JwtCacheSize)
	}
	if args.JwksCacheDurationInS != nil && *args.JwksCacheDurationInS != "" {
		proxyConf = append(proxyConf, "--jwks_cache_duration_in_s", *args.JwksCacheDurationInS)
	}
	if args.JwksFetchNumRetries != nil && *args.JwksFetchNumRetries != "" {
		proxyConf = append(proxyConf, "--jwks_fetch_num_retries", *args.JwksFetchNumRetries)
	}
	if args.JwksFetchRetryBackOffBaseIntervalMs != nil && *args.JwksFetchRetryBackOffBaseIntervalMs != "" {
		proxyConf = append(proxyConf, "--jwks_fetch_retry_back_off_base_interval_ms", *args.JwksFetchRetryBackOffBaseIntervalMs)
	}
	if args.JwksFetchRetryBackOffMaxIntervalMs != nil && *args.JwksFetchRetryBackOffMaxIntervalMs != "" {
		proxyConf = append(proxyConf, "--jwks_fetch_retry_back_off_max_interval_ms", *args.JwksFetchRetryBackOffMaxIntervalMs)
	}
	if args.JwtPadForwardPayloadHeader {
		proxyConf = append(proxyConf, "--jwt_pad_forward_payload_header")
	}
	if args.DisableJwtAudienceServiceNameCheck {
		proxyConf = append(proxyConf, "--disable_jwt_audience_service_name_check")
	}

	if args.Management != nil && *args.Management != "" {
		proxyConf = append(proxyConf, "--service_management_url", *args.Management)
	}
	if args.LogRequestHeaders != nil && *args.LogRequestHeaders != "" {
		proxyConf = append(proxyConf, "--log_request_headers", *args.LogRequestHeaders)
	}
	if args.LogResponseHeaders != nil && *args.LogResponseHeaders != "" {
		proxyConf = append(proxyConf, "--log_response_headers", *args.LogResponseHeaders)
	}
	if args.LogJwtPayloads != nil && *args.LogJwtPayloads != "" {
		proxyConf = append(proxyConf, "--log_jwt_payloads", *args.LogJwtPayloads)
	}

	if args.HTTPPort != nil && *args.HTTPPort != 0 {
		proxyConf = append(proxyConf, "--listener_port", strconv.Itoa(*args.HTTPPort))
	}
	if args.HTTP2Port != nil && *args.HTTP2Port != 0 {
		proxyConf = append(proxyConf, "--listener_port", strconv.Itoa(*args.HTTP2Port))
	}
	if args.ListenerPort != nil && *args.ListenerPort != 0 {
		proxyConf = append(proxyConf, "--listener_port", strconv.Itoa(*args.ListenerPort))
	}
	if args.SslServerCertPath != nil && *args.SslServerCertPath != "" {
		proxyConf = append(proxyConf, "--ssl_server_cert_path", *args.SslServerCertPath)
	}
	if args.SslServerRootCertPath != nil && *args.SslServerRootCertPath != "" {
		proxyConf = append(proxyConf, "--ssl_server_root_cert_path", *args.SslServerRootCertPath)
	}
	if args.SslPort != nil && *args.SslPort != 0 {
		proxyConf = append(proxyConf, "--ssl_server_cert_path", "/etc/nginx/ssl")
		proxyConf = append(proxyConf, "--listener_port", strconv.Itoa(*args.SslPort))
	}

	if args.SslBackendClientCertPath != nil && *args.SslBackendClientCertPath != "" {
		proxyConf = append(proxyConf, "--ssl_backend_client_cert_path", *args.SslBackendClientCertPath)
	}
	if args.SslClientCertPath != nil && *args.SslClientCertPath != "" {
		proxyConf = append(proxyConf, "--ssl_backend_client_cert_path", *args.SslClientCertPath)
	}

	if args.EnableGrpcBackendSsl && args.GrpcBackendSslRootCertsFile != "" {
		proxyConf = append(proxyConf, "--ssl_backend_client_root_certs_path", args.GrpcBackendSslRootCertsFile)
	}
	if args.SslBackendClientRootCertsFile != nil && *args.SslBackendClientRootCertsFile != "" {
		proxyConf = append(proxyConf, "--ssl_backend_client_root_certs_path", *args.SslBackendClientRootCertsFile)
	}
	if args.SslClientRootCertsFile != nil && *args.SslClientRootCertsFile != "" {
		proxyConf = append(proxyConf, "--ssl_backend_client_root_certs_path", *args.SslClientRootCertsFile)
	}

	if args.SslServerCipherSuites != nil && *args.SslServerCipherSuites != "" {
		proxyConf = append(proxyConf, "--ssl_server_cipher_suites", *args.SslServerCipherSuites)
	}
	if args.SslBackendClientCipherSuites != nil && *args.SslBackendClientCipherSuites != "" {
		proxyConf = append(proxyConf, "--ssl_backend_client_cipher_suites", *args.SslBackendClientCipherSuites)
	}

	if args.TLSMutualAuth {
		proxyConf = append(proxyConf, "--ssl_backend_client_cert_path", "/etc/nginx/ssl")
	}

	if args.SslMinimumProtocol != nil && *args.SslMinimumProtocol != "" {
		proxyConf = append(proxyConf, "--ssl_minimum_protocol", *args.SslMinimumProtocol)
	}
	if args.SslMaximumProtocol != nil && *args.SslMaximumProtocol != "" {
		proxyConf = append(proxyConf, "--ssl_maximum_protocol", *args.SslMaximumProtocol)
	}
	if len(args.SslProtocols) > 0 {
		sorted := make([]string, len(args.SslProtocols))
		copy(sorted, args.SslProtocols)
		sort.Strings(sorted)
		proxyConf = append(proxyConf, "--ssl_minimum_protocol", sorted[0])
		proxyConf = append(proxyConf, "--ssl_maximum_protocol", sorted[len(sorted)-1])
	}

	if len(args.AddRequestHeader) > 0 {
		proxyConf = append(proxyConf, "--add_request_headers", strings.Join(args.AddRequestHeader, ";"))
	}
	if len(args.AppendRequestHeader) > 0 {
		proxyConf = append(proxyConf, "--append_request_headers", strings.Join(args.AppendRequestHeader, ";"))
	}
	if len(args.AddResponseHeader) > 0 {
		proxyConf = append(proxyConf, "--add_response_headers", strings.Join(args.AddResponseHeader, ";"))
	}
	if len(args.AppendResponseHeader) > 0 {
		proxyConf = append(proxyConf, "--append_response_headers", strings.Join(args.AppendResponseHeader, ";"))
	}

	if args.EnableOperationNameHeader {
		proxyConf = append(proxyConf, "--enable_operation_name_header")
	}
	if args.EnableResponseCompression {
		proxyConf = append(proxyConf, "--enable_response_compression")
	}

	if args.GenerateSelfSignedCert {
		log.Println("Generating self-signed certificate...")
		if err := generateSelfSignedCert(l.certDir); err != nil {
			return nil, fmt.Errorf("Failed to create self-signed cert: %w", err)
		}
		proxyConf = append(proxyConf, "--ssl_server_cert_path", defaultSelfSignedCertDir)
	}

	if args.EnableStrictTransportSecurity {
		proxyConf = append(proxyConf, "--enable_strict_transport_security")
	}

	if args.Service != "" {
		proxyConf = append(proxyConf, "--service", args.Service)
	}
	if args.Version != "" {
		proxyConf = append(proxyConf, "--service_config_id", args.Version)
	}
	if args.HTTPRequestTimeoutS != nil && *args.HTTPRequestTimeoutS != 0 {
		proxyConf = append(proxyConf, "--http_request_timeout_s", strconv.Itoa(*args.HTTPRequestTimeoutS))
	}
	if args.ServiceControlURL != nil && *args.ServiceControlURL != "" {
		proxyConf = append(proxyConf, "--service_control_url", *args.ServiceControlURL)
	}
	if args.ServiceControlCheckRetries != nil && *args.ServiceControlCheckRetries != "" {
		proxyConf = append(proxyConf, "--service_control_check_retries", *args.ServiceControlCheckRetries)
	}
	if args.ServiceControlQuotaRetries != nil && *args.ServiceControlQuotaRetries != "" {
		proxyConf = append(proxyConf, "--service_control_quota_retries", *args.ServiceControlQuotaRetries)
	}
	if args.ServiceControlReportRetries != nil && *args.ServiceControlReportRetries != "" {
		proxyConf = append(proxyConf, "--service_control_report_retries", *args.ServiceControlReportRetries)
	}
	if args.ServiceControlCheckTimeoutMs != nil && *args.ServiceControlCheckTimeoutMs != "" {
		proxyConf = append(proxyConf, "--service_control_check_timeout_ms", *args.ServiceControlCheckTimeoutMs)
	}
	if args.ServiceControlQuotaTimeoutMs != nil && *args.ServiceControlQuotaTimeoutMs != "" {
		proxyConf = append(proxyConf, "--service_control_quota_timeout_ms", *args.ServiceControlQuotaTimeoutMs)
	}
	if args.ServiceControlReportTimeoutMs != nil && *args.ServiceControlReportTimeoutMs != "" {
		proxyConf = append(proxyConf, "--service_control_report_timeout_ms", *args.ServiceControlReportTimeoutMs)
	}

	if args.ServiceControlNetworkFailPolicy == "close" {
		proxyConf = append(proxyConf, "--service_control_network_fail_open=false")
	}
	if args.ServiceControlEnableAPIKeyUIDReporting {
		proxyConf = append(proxyConf, "--service_control_enable_api_key_uid_reporting")
	}
	if args.ServiceJSONPath != nil && *args.ServiceJSONPath != "" {
		proxyConf = append(proxyConf, "--service_json_path", *args.ServiceJSONPath)
	}
	if args.CheckMetadata {
		proxyConf = append(proxyConf, "--check_metadata")
	}
	if args.UnderscoresInHeaders {
		proxyConf = append(proxyConf, "--underscores_in_headers")
	}
	if args.DisableNormalizePath {
		proxyConf = append(proxyConf, "--normalize_path=false")
	}
	if args.DisableMergeSlashesInPath {
		proxyConf = append(proxyConf, "--merge_slashes_in_path=false")
	}
	if args.DisallowEscapedSlashesInPath {
		proxyConf = append(proxyConf, "--disallow_escaped_slashes_in_path")
	}

	if args.BackendRetryOns != nil && *args.BackendRetryOns != "" {
		proxyConf = append(proxyConf, "--backend_retry_ons", *args.BackendRetryOns)
	}
	if args.BackendRetryOnStatusCodes != nil && *args.BackendRetryOnStatusCodes != "" {
		proxyConf = append(proxyConf, "--backend_retry_on_status_codes", *args.BackendRetryOnStatusCodes)
	}
	if args.BackendRetryNum != nil && *args.BackendRetryNum != "" {
		proxyConf = append(proxyConf, "--backend_retry_num", *args.BackendRetryNum)
	}
	if args.BackendPerTryTimeout != nil && *args.BackendPerTryTimeout != "" {
		proxyConf = append(proxyConf, "--backend_per_try_timeout", *args.BackendPerTryTimeout)
	}

	if args.AccessLog != nil && *args.AccessLog != "" {
		proxyConf = append(proxyConf, "--access_log", *args.AccessLog)
	}
	if args.AccessLogFormat != nil && *args.AccessLogFormat != "" {
		proxyConf = append(proxyConf, "--access_log_format", *args.AccessLogFormat)
	}

	if args.DisableTracing {
		proxyConf = append(proxyConf, "--disable_tracing")
	} else {
		if args.TracingProjectID != "" {
			proxyConf = append(proxyConf, "--tracing_project_id", args.TracingProjectID)
		}
		if args.TracingIncomingContext != "" {
			proxyConf = append(proxyConf, "--tracing_incoming_context", args.TracingIncomingContext)
		}
		if args.TracingOutgoingContext != "" {
			proxyConf = append(proxyConf, "--tracing_outgoing_context", args.TracingOutgoingContext)
		}
		if args.CloudTraceURLOverride != "" {
			proxyConf = append(proxyConf, "--tracing_stackdriver_address", args.CloudTraceURLOverride)
		}
		if args.DisableCloudTraceAutoSampling {
			proxyConf = append(proxyConf, "--tracing_sample_rate", "0")
		} else if args.TracingSampleRate != nil && *args.TracingSampleRate != "" {
			proxyConf = append(proxyConf, "--tracing_sample_rate", *args.TracingSampleRate)
		}
	}

	if args.TranscodingAlwaysPrintPrimitiveFields {
		proxyConf = append(proxyConf, "--transcoding_always_print_primitive_fields")
	}
	if args.TranscodingAlwaysPrintEnumsAsInts {
		proxyConf = append(proxyConf, "--transcoding_always_print_enums_as_ints")
	}
	if args.TranscodingStreamNewlineDelimited {
		proxyConf = append(proxyConf, "--transcoding_stream_newline_delimited")
	}
	if args.TranscodingCaseInsensitiveEnumParsing {
		proxyConf = append(proxyConf, "--transcoding_case_insensitive_enum_parsing")
	}
	if args.TranscodingPreserveProtoFieldNames {
		proxyConf = append(proxyConf, "--transcoding_preserve_proto_field_names")
	}
	if args.TranscodingIgnoreQueryParameters != nil && *args.TranscodingIgnoreQueryParameters != "" {
		proxyConf = append(proxyConf, "--transcoding_ignore_query_parameters", *args.TranscodingIgnoreQueryParameters)
	}
	if args.TranscodingIgnoreUnknownQueryParameters {
		proxyConf = append(proxyConf, "--transcoding_ignore_unknown_query_parameters")
	}
	if args.TranscodingQueryParametersDisableUnescapePlus {
		proxyConf = append(proxyConf, "--transcoding_query_parameters_disable_unescape_plus")
	}
	if args.TranscodingMatchUnregisteredCustomVerb {
		proxyConf = append(proxyConf, "--transcoding_match_unregistered_custom_verb")
	}
	if args.DisallowColonInWildcardPathSegment {
		proxyConf = append(proxyConf, "--disallow_colon_in_wildcard_path_segment")
	}

	if args.OnServerless {
		proxyConf = append(proxyConf, "--compute_platform_override", serverlessPlatform)
	}
	if args.BackendDNSLookupFamily != nil && *args.BackendDNSLookupFamily != "" {
		proxyConf = append(proxyConf, "--backend_dns_lookup_family", *args.BackendDNSLookupFamily)
	}
	if args.DNSResolverAddresses != nil && *args.DNSResolverAddresses != "" {
		proxyConf = append(proxyConf, "--dns_resolver_addresses", *args.DNSResolverAddresses)
	}
	if args.DNS != nil && *args.DNS != "" {
		proxyConf = append(proxyConf, "--dns_resolver_addresses", *args.DNS)
	}
	if args.EnvoyUseRemoteAddress {
		proxyConf = append(proxyConf, "--envoy_use_remote_address")
	}

	if args.CorsPreset != nil && *args.CorsPreset != "" {
		proxyConf = append(proxyConf,
			"--cors_preset", *args.CorsPreset,
			"--cors_allow_origin", args.CorsAllowOrigin,
			"--cors_allow_origin_regex", args.CorsAllowOriginRegex,
			"--cors_allow_methods", args.CorsAllowMethods,
			"--cors_allow_headers", args.CorsAllowHeaders,
			"--cors_expose_headers", args.CorsExposeHeaders,
			"--cors_max_age", args.CorsMaxAge,
		)
		if args.CorsAllowCredentials {
			proxyConf = append(proxyConf, "--cors_allow_credentials")
		}
	}

	if args.EnableApplicationDefaultCredentials {
		proxyConf = append(proxyConf, "--enable_application_default_credentials")
	}
	if args.ServiceAccountKey != nil && *args.ServiceAccountKey != "" {
		proxyConf = append(proxyConf, "--service_account_key", *args.ServiceAccountKey)
	}
	if args.NonGcp {
		proxyConf = append(proxyConf, "--non_gcp")
	}
	if args.EnableDebug {
		proxyConf = append(proxyConf, "--suppress_envoy_headers=false")
	}
	if args.EnvoyConnectionBufferLimitBytes != nil && *args.EnvoyConnectionBufferLimitBytes != "" {
		proxyConf = append(proxyConf, "--connection_buffer_limit_bytes", *args.EnvoyConnectionBufferLimitBytes)
	}
	if args.EnableBackendAddressOverride {
		proxyConf = append(proxyConf, "--enable_backend_address_override")
	}
	if args.AdsNamedPipe != nil && *args.AdsNamedPipe != "" {
		proxyConf = append(proxyConf, "--ads_named_pipe", *args.AdsNamedPipe)
	}

	return proxyConf, nil
}

// genEnvoyArgs builds the command-line slice for bin/envoy.
func (l *launcher) genEnvoyArgs(args *proxyArgs) []string {
	cmd := []string{
		l.envoyBin,
		"-c", defaultConfigDir + bootstrapConfig,
		"--disable-hot-restart",
		"--log-format %L%m%d %T.%e %t %@] [%t][%n]%v",
		"--log-format-escaped",
	}
	if args.EnableDebug {
		cmd = append(cmd, "-l debug", "--component-log-level upstream:info,main:info")
	}
	if args.EnvoyExtraConfigYaml != nil && *args.EnvoyExtraConfigYaml != "" {
		cmd = append(cmd, fmt.Sprintf("--config-yaml %s", *args.EnvoyExtraConfigYaml))
	}
	return cmd
}
