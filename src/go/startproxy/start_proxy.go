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

// Package startproxy implements the ESPv2 startup and process supervision logic,
// replacing docker/generic/start_proxy.py and docker/serverless/env_start_proxy.py.
package startproxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ErrHelpRequested is returned when -h or --help is passed to ParseArgs.
var ErrHelpRequested = errors.New("help requested")

var (
	// BootstrapCmd is the command to generate Envoy bootstrap config.
	BootstrapCmd = "bin/bootstrap"

	// ConfigManagerBin is the location of the Config Manager binary.
	ConfigManagerBin = "bin/configmanager"

	// EnvoyBin is the location of the Envoy binary.
	EnvoyBin = "bin/envoy"

	// ShutdownGracePeriod is how long Run waits for both child processes to
	// exit after forwarding SIGTERM/SIGINT before SIGKILLing any survivors.
	ShutdownGracePeriod = 10 * time.Second

	// notifySignals registers c to receive shutdown signals. Overridable in
	// tests so they can inject signals without signalling the test process.
	notifySignals = func(c chan<- os.Signal) {
		signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	}

	// stopSignals undoes notifySignals.
	stopSignals = func(c chan<- os.Signal) {
		signal.Stop(c)
	}
)

const (
	// HealthCheckPeriod is the health check period for Config Manager and Envoy.
	HealthCheckPeriod = 2 * time.Second

	// DefaultConfigDir is the directory where bootstrap config file is written.
	DefaultConfigDir = "/tmp"

	// BootstrapConfig is the bootstrap config file name.
	BootstrapConfig = "/bootstrap.json"

	// DefaultListenerPort is the default listener port.
	DefaultListenerPort = 8080

	// DefaultBackend is the default backend address.
	DefaultBackend = "http://127.0.0.1:8082"

	// DefaultRolloutStrategy is the default rollout strategy.
	DefaultRolloutStrategy = "fixed"

	// GoogleCredsKey is the Google default application credentials environment variable.
	GoogleCredsKey = "GOOGLE_APPLICATION_CREDENTIALS"

	// ServerlessPlatform is the platform string when running on serverless.
	ServerlessPlatform = "Cloud Run(ESPv2)"

	// ServerlessXffNumTrustedHops is the default XFF num trusted hops on serverless.
	ServerlessXffNumTrustedHops = 0

	// DefaultSelfSignedCertDir is the directory where self-signed certs are written.
	DefaultSelfSignedCertDir = "/tmp/ssl/endpoints"
)

// SelfSignedCertDir can be overridden in unit tests.
var SelfSignedCertDir = DefaultSelfSignedCertDir

// Args holds all parsed command-line arguments for startproxy.
type Args struct {
	Service                                       string
	Version                                       string
	ServiceJSONPath                               *string
	Backend                                       string
	EnableBackendAddressOverride                  bool
	ListenerPort                                  *int
	StatusPort                                    int
	SslServerCertPath                             *string
	SslServerCipherSuites                         *string
	SslServerRootCertPath                         *string
	SslBackendClientCertPath                      *string
	SslBackendClientRootCertsFile                 *string
	SslBackendClientCipherSuites                  *string
	SslMinimumProtocol                            *string
	SslMaximumProtocol                            *string
	EnableStrictTransportSecurity                 bool
	GenerateSelfSignedCert                        bool
	Healthz                                       *string
	HealthCheckGrpcBackend                        bool
	HealthCheckGrpcBackendService                 *string
	HealthCheckGrpcBackendInterval                *string
	HealthCheckGrpcBackendNoTrafficInterval       *string
	AddRequestHeader                              []string
	AppendRequestHeader                           []string
	AddResponseHeader                             []string
	AppendResponseHeader                          []string
	EnableOperationNameHeader                     bool
	RolloutStrategy                               string
	Management                                    *string
	CorsPreset                                    *string
	CorsAllowOrigin                               string
	CorsAllowOriginRegex                          string
	CorsAllowMethods                              string
	CorsAllowHeaders                              string
	CorsExposeHeaders                             string
	CorsAllowCredentials                          bool
	CorsMaxAge                                    string
	CheckMetadata                                 bool
	UnderscoresInHeaders                          bool
	DisableNormalizePath                          bool
	DisableMergeSlashesInPath                     bool
	DisallowEscapedSlashesInPath                  bool
	EnvoyUseRemoteAddress                         bool
	EnvoyXffNumTrustedHops                        *string
	EnvoyConnectionBufferLimitBytes               *string
	LogRequestHeaders                             *string
	LogResponseHeaders                            *string
	LogJwtPayloads                                *string
	ServiceControlNetworkFailPolicy               string
	ServiceControlEnableApiKeyUidReporting        bool
	DisableJwksAsyncFetch                         bool
	JwksAsyncFetchFastListener                    bool
	JwtCacheSize                                  *string
	JwksCacheDurationInS                          *string
	JwksFetchNumRetries                           *string
	JwksFetchRetryBackOffBaseIntervalMs           *string
	JwksFetchRetryBackOffMaxIntervalMs            *string
	JwtPadForwardPayloadHeader                    bool
	DisableJwtAudienceServiceNameCheck            bool
	HttpRequestTimeoutS                           *int
	ServiceControlUrl                             *string
	ServiceControlCheckTimeoutMs                  *string
	ServiceControlQuotaTimeoutMs                  *string
	ServiceControlReportTimeoutMs                 *string
	ServiceControlCheckRetries                    *string
	ServiceControlQuotaRetries                    *string
	ServiceControlReportRetries                   *string
	BackendRetryOns                               *string
	BackendRetryOnStatusCodes                     *string
	BackendRetryNum                               *string
	BackendPerTryTimeout                          *string
	AccessLog                                     *string
	AccessLogFormat                               *string
	DisableTracing                                bool
	TracingProjectId                              string
	TracingSampleRate                             *string
	DisableCloudTraceAutoSampling                 bool
	TracingIncomingContext                        string
	TracingOutgoingContext                        string
	CloudTraceUrlOverride                         string
	NonGcp                                        bool
	ServiceAccountKey                             *string
	EnableApplicationDefaultCredentials           bool
	DnsResolverAddresses                          *string
	BackendDnsLookupFamily                        *string
	EnableDebug                                   bool
	TranscodingAlwaysPrintPrimitiveFields         bool
	TranscodingAlwaysPrintEnumsAsInts             bool
	TranscodingStreamNewlineDelimited             bool
	TranscodingCaseInsensitiveEnumParsing         bool
	TranscodingPreserveProtoFieldNames            bool
	TranscodingIgnoreQueryParameters              *string
	TranscodingIgnoreUnknownQueryParameters       bool
	TranscodingQueryParametersDisableUnescapePlus bool
	TranscodingMatchUnregisteredCustomVerb        bool
	DisallowColonInWildcardPathSegment            bool
	AdsNamedPipe                                  *string
	EnvoyExtraConfigYaml                          *string
	EnableResponseCompression                     bool
	EnableBackendRouting                          bool
	BackendProtocol                               *string
	HttpPort                                      *int
	Http2Port                                     *int
	SslPort                                       *int
	Dns                                           *string
	TlsMutualAuth                                 bool
	SslProtocols                                  []string
	EnableGrpcBackendSsl                          bool
	GrpcBackendSslRootCertsFile                   string
	SslClientCertPath                             *string
	SslClientRootCertsFile                        *string
	OnServerless                                  bool
	ServerlessEntrypoint                          bool
}

// DefaultArgs returns an Args struct populated with the default values from start_proxy.py.
func DefaultArgs() *Args {
	return &Args{
		Service:                                "",
		Version:                                "",
		Backend:                                DefaultBackend,
		StatusPort:                             0,
		RolloutStrategy:                        DefaultRolloutStrategy,
		CorsAllowOrigin:                        "*",
		CorsAllowOriginRegex:                   "",
		CorsAllowMethods:                       "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		CorsAllowHeaders:                       "DNT,User-Agent,X-User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization",
		CorsExposeHeaders:                      "Content-Length,Content-Range",
		CorsMaxAge:                             "480h",
		ServiceControlNetworkFailPolicy:        "open",
		ServiceControlEnableApiKeyUidReporting: true,
		TracingProjectId:                       "",
		TracingIncomingContext:                 "",
		TracingOutgoingContext:                 "",
		CloudTraceUrlOverride:                  "",
		GrpcBackendSslRootCertsFile:            "/etc/nginx/trusted-ca-certificates.crt",
	}
}

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
	apply   func(a *Args, val string) error
}

func buildFlagDefs() []*flagDef {
	setOptStr := func(target **string) func(*Args, string) error {
		return func(_ *Args, val string) error {
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
			apply: func(a *Args, val string) error { a.Service = val; return nil },
		},
		{
			short: "-v",
			longs: []string{"--version"},
			kind:  kindString,
			apply: func(a *Args, val string) error { a.Version = val; return nil },
		},
		{
			longs: []string{"--service_json_path"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.ServiceJSONPath)(a, val) },
		},
		{
			short: "-a",
			longs: []string{"--backend"},
			kind:  kindString,
			apply: func(a *Args, val string) error { a.Backend = val; return nil },
		},
		{
			longs: []string{"--enable_backend_address_override"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.EnableBackendAddressOverride = true; return nil },
		},
		{
			longs: []string{"--listener_port"},
			kind:  kindOptInt,
			apply: func(a *Args, val string) error {
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
			apply: func(a *Args, val string) error {
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
			apply: func(a *Args, val string) error { return setOptStr(&a.SslServerCertPath)(a, val) },
		},
		{
			longs: []string{"--ssl_server_cipher_suites"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.SslServerCipherSuites)(a, val) },
		},
		{
			longs: []string{"--ssl_server_root_cert_path"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.SslServerRootCertPath)(a, val) },
		},
		{
			longs: []string{"--ssl_backend_client_cert_path"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.SslBackendClientCertPath)(a, val) },
		},
		{
			longs: []string{"--ssl_backend_client_root_certs_file"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.SslBackendClientRootCertsFile)(a, val) },
		},
		{
			longs: []string{"--ssl_backend_client_cipher_suites"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.SslBackendClientCipherSuites)(a, val) },
		},
		{
			longs:   []string{"--ssl_minimum_protocol"},
			kind:    kindOptString,
			choices: []string{"TLSv1.0", "TLSv1.1", "TLSv1.2", "TLSv1.3"},
			apply:   func(a *Args, val string) error { return setOptStr(&a.SslMinimumProtocol)(a, val) },
		},
		{
			longs:   []string{"--ssl_maximum_protocol"},
			kind:    kindOptString,
			choices: []string{"TLSv1.0", "TLSv1.1", "TLSv1.2", "TLSv1.3"},
			apply:   func(a *Args, val string) error { return setOptStr(&a.SslMaximumProtocol)(a, val) },
		},
		{
			longs: []string{"--enable_strict_transport_security"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.EnableStrictTransportSecurity = true; return nil },
		},
		{
			longs: []string{"--generate_self_signed_cert"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.GenerateSelfSignedCert = true; return nil },
		},
		{
			short: "-z",
			longs: []string{"--healthz"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.Healthz)(a, val) },
		},
		{
			longs: []string{"--health_check_grpc_backend"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.HealthCheckGrpcBackend = true; return nil },
		},
		{
			longs: []string{"--health_check_grpc_backend_service"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.HealthCheckGrpcBackendService)(a, val) },
		},
		{
			longs: []string{"--health_check_grpc_backend_interval"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.HealthCheckGrpcBackendInterval)(a, val) },
		},
		{
			longs: []string{"--health_check_grpc_backend_no_traffic_interval"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.HealthCheckGrpcBackendNoTrafficInterval)(a, val) },
		},
		{
			longs: []string{"--add_request_header"},
			kind:  kindAppend,
			apply: func(a *Args, val string) error { a.AddRequestHeader = append(a.AddRequestHeader, val); return nil },
		},
		{
			longs: []string{"--append_request_header"},
			kind:  kindAppend,
			apply: func(a *Args, val string) error {
				a.AppendRequestHeader = append(a.AppendRequestHeader, val)
				return nil
			},
		},
		{
			longs: []string{"--add_response_header"},
			kind:  kindAppend,
			apply: func(a *Args, val string) error { a.AddResponseHeader = append(a.AddResponseHeader, val); return nil },
		},
		{
			longs: []string{"--append_response_header"},
			kind:  kindAppend,
			apply: func(a *Args, val string) error {
				a.AppendResponseHeader = append(a.AppendResponseHeader, val)
				return nil
			},
		},
		{
			longs: []string{"--enable_operation_name_header"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.EnableOperationNameHeader = true; return nil },
		},
		{
			short:   "-R",
			longs:   []string{"--rollout_strategy"},
			kind:    kindString,
			choices: []string{"fixed", "managed"},
			apply:   func(a *Args, val string) error { a.RolloutStrategy = val; return nil },
		},
		{
			short: "-g",
			longs: []string{"--management"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.Management)(a, val) },
		},
		{
			longs: []string{"--cors_preset"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.CorsPreset)(a, val) },
		},
		{
			longs: []string{"--cors_allow_origin"},
			kind:  kindString,
			apply: func(a *Args, val string) error { a.CorsAllowOrigin = val; return nil },
		},
		{
			longs: []string{"--cors_allow_origin_regex"},
			kind:  kindString,
			apply: func(a *Args, val string) error { a.CorsAllowOriginRegex = val; return nil },
		},
		{
			longs: []string{"--cors_allow_methods"},
			kind:  kindString,
			apply: func(a *Args, val string) error { a.CorsAllowMethods = val; return nil },
		},
		{
			longs: []string{"--cors_allow_headers"},
			kind:  kindString,
			apply: func(a *Args, val string) error { a.CorsAllowHeaders = val; return nil },
		},
		{
			longs: []string{"--cors_expose_headers"},
			kind:  kindString,
			apply: func(a *Args, val string) error { a.CorsExposeHeaders = val; return nil },
		},
		{
			longs: []string{"--cors_allow_credentials"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.CorsAllowCredentials = true; return nil },
		},
		{
			longs: []string{"--cors_max_age"},
			kind:  kindString,
			apply: func(a *Args, val string) error { a.CorsMaxAge = val; return nil },
		},
		{
			longs: []string{"--check_metadata"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.CheckMetadata = true; return nil },
		},
		{
			longs: []string{"--underscores_in_headers"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.UnderscoresInHeaders = true; return nil },
		},
		{
			longs: []string{"--disable_normalize_path"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.DisableNormalizePath = true; return nil },
		},
		{
			longs: []string{"--disable_merge_slashes_in_path"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.DisableMergeSlashesInPath = true; return nil },
		},
		{
			longs: []string{"--disallow_escaped_slashes_in_path"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.DisallowEscapedSlashesInPath = true; return nil },
		},
		{
			longs: []string{"--envoy_use_remote_address"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.EnvoyUseRemoteAddress = true; return nil },
		},
		{
			longs: []string{"--envoy_xff_num_trusted_hops"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.EnvoyXffNumTrustedHops)(a, val) },
		},
		{
			longs: []string{"--envoy_connection_buffer_limit_bytes"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.EnvoyConnectionBufferLimitBytes)(a, val) },
		},
		{
			longs: []string{"--log_request_headers"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.LogRequestHeaders)(a, val) },
		},
		{
			longs: []string{"--log_response_headers"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.LogResponseHeaders)(a, val) },
		},
		{
			longs: []string{"--log_jwt_payloads"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.LogJwtPayloads)(a, val) },
		},
		{
			longs:   []string{"--service_control_network_fail_policy"},
			kind:    kindString,
			choices: []string{"open", "close"},
			apply:   func(a *Args, val string) error { a.ServiceControlNetworkFailPolicy = val; return nil },
		},
		{
			longs: []string{"--service_control_enable_api_key_uid_reporting"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.ServiceControlEnableApiKeyUidReporting = true; return nil },
		},
		{
			longs: []string{"--no-service_control_enable_api_key_uid_reporting"},
			kind:  kindStoreFalse,
			apply: func(a *Args, _ string) error { a.ServiceControlEnableApiKeyUidReporting = false; return nil },
		},
		{
			longs: []string{"--disable_jwks_async_fetch"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.DisableJwksAsyncFetch = true; return nil },
		},
		{
			longs: []string{"--jwks_async_fetch_fast_listener"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.JwksAsyncFetchFastListener = true; return nil },
		},
		{
			longs: []string{"--jwt_cache_size"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.JwtCacheSize)(a, val) },
		},
		{
			longs: []string{"--jwks_cache_duration_in_s"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.JwksCacheDurationInS)(a, val) },
		},
		{
			longs: []string{"--jwks_fetch_num_retries"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.JwksFetchNumRetries)(a, val) },
		},
		{
			longs: []string{"--jwks_fetch_retry_back_off_base_interval_ms"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.JwksFetchRetryBackOffBaseIntervalMs)(a, val) },
		},
		{
			longs: []string{"--jwks_fetch_retry_back_off_max_interval_ms"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.JwksFetchRetryBackOffMaxIntervalMs)(a, val) },
		},
		{
			longs: []string{"--jwt_pad_forward_payload_header"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.JwtPadForwardPayloadHeader = true; return nil },
		},
		{
			longs: []string{"--disable_jwt_audience_service_name_check"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.DisableJwtAudienceServiceNameCheck = true; return nil },
		},
		{
			longs: []string{"--http_request_timeout_s"},
			kind:  kindOptInt,
			apply: func(a *Args, val string) error {
				n, err := strconv.Atoi(val)
				if err != nil {
					return err
				}
				a.HttpRequestTimeoutS = &n
				return nil
			},
		},
		{
			longs: []string{"--service_control_url"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.ServiceControlUrl)(a, val) },
		},
		{
			longs: []string{"--service_control_check_timeout_ms"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.ServiceControlCheckTimeoutMs)(a, val) },
		},
		{
			longs: []string{"--service_control_quota_timeout_ms"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.ServiceControlQuotaTimeoutMs)(a, val) },
		},
		{
			longs: []string{"--service_control_report_timeout_ms"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.ServiceControlReportTimeoutMs)(a, val) },
		},
		{
			longs: []string{"--service_control_check_retries"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.ServiceControlCheckRetries)(a, val) },
		},
		{
			longs: []string{"--service_control_quota_retries"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.ServiceControlQuotaRetries)(a, val) },
		},
		{
			longs: []string{"--service_control_report_retries"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.ServiceControlReportRetries)(a, val) },
		},
		{
			longs: []string{"--backend_retry_ons"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.BackendRetryOns)(a, val) },
		},
		{
			longs: []string{"--backend_retry_on_status_codes"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.BackendRetryOnStatusCodes)(a, val) },
		},
		{
			longs: []string{"--backend_retry_num"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.BackendRetryNum)(a, val) },
		},
		{
			longs: []string{"--backend_per_try_timeout"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.BackendPerTryTimeout)(a, val) },
		},
		{
			longs: []string{"--access_log"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.AccessLog)(a, val) },
		},
		{
			longs: []string{"--access_log_format"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.AccessLogFormat)(a, val) },
		},
		{
			longs: []string{"--disable_tracing"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.DisableTracing = true; return nil },
		},
		{
			longs: []string{"--tracing_project_id"},
			kind:  kindString,
			apply: func(a *Args, val string) error { a.TracingProjectId = val; return nil },
		},
		{
			longs: []string{"--tracing_sample_rate"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.TracingSampleRate)(a, val) },
		},
		{
			longs: []string{"--disable_cloud_trace_auto_sampling"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.DisableCloudTraceAutoSampling = true; return nil },
		},
		{
			longs: []string{"--tracing_incoming_context"},
			kind:  kindString,
			apply: func(a *Args, val string) error { a.TracingIncomingContext = val; return nil },
		},
		{
			longs: []string{"--tracing_outgoing_context"},
			kind:  kindString,
			apply: func(a *Args, val string) error { a.TracingOutgoingContext = val; return nil },
		},
		{
			longs: []string{"--cloud_trace_url_override"},
			kind:  kindString,
			apply: func(a *Args, val string) error { a.CloudTraceUrlOverride = val; return nil },
		},
		{
			longs: []string{"--non_gcp"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.NonGcp = true; return nil },
		},
		{
			longs: []string{"--service_account_key"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.ServiceAccountKey)(a, val) },
		},
		{
			longs: []string{"--enable_application_default_credentials"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.EnableApplicationDefaultCredentials = true; return nil },
		},
		{
			longs: []string{"--dns_resolver_addresses"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.DnsResolverAddresses)(a, val) },
		},
		{
			longs:   []string{"--backend_dns_lookup_family"},
			kind:    kindOptString,
			choices: []string{"auto", "v4only", "v6only", "v4preferred", "all"},
			apply:   func(a *Args, val string) error { return setOptStr(&a.BackendDnsLookupFamily)(a, val) },
		},
		{
			longs: []string{"--enable_debug"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.EnableDebug = true; return nil },
		},
		{
			longs: []string{"--transcoding_always_print_primitive_fields"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.TranscodingAlwaysPrintPrimitiveFields = true; return nil },
		},
		{
			longs: []string{"--transcoding_always_print_enums_as_ints"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.TranscodingAlwaysPrintEnumsAsInts = true; return nil },
		},
		{
			longs: []string{"--transcoding_stream_newline_delimited"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.TranscodingStreamNewlineDelimited = true; return nil },
		},
		{
			longs: []string{"--transcoding_case_insensitive_enum_parsing"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.TranscodingCaseInsensitiveEnumParsing = true; return nil },
		},
		{
			longs: []string{"--transcoding_preserve_proto_field_names"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.TranscodingPreserveProtoFieldNames = true; return nil },
		},
		{
			longs: []string{"--transcoding_ignore_query_parameters"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.TranscodingIgnoreQueryParameters)(a, val) },
		},
		{
			longs: []string{"--transcoding_ignore_unknown_query_parameters"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.TranscodingIgnoreUnknownQueryParameters = true; return nil },
		},
		{
			longs: []string{"--transcoding_query_parameters_disable_unescape_plus"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.TranscodingQueryParametersDisableUnescapePlus = true; return nil },
		},
		{
			longs: []string{"--transcoding_match_unregistered_custom_verb"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.TranscodingMatchUnregisteredCustomVerb = true; return nil },
		},
		{
			longs: []string{"--disallow_colon_in_wildcard_path_segment"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.DisallowColonInWildcardPathSegment = true; return nil },
		},
		{
			longs: []string{"--ads_named_pipe"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.AdsNamedPipe)(a, val) },
		},
		{
			longs: []string{"--envoy_extra_config_yaml"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.EnvoyExtraConfigYaml)(a, val) },
		},
		{
			longs: []string{"--enable_response_compression"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.EnableResponseCompression = true; return nil },
		},
		{
			longs: []string{"--enable_backend_routing"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.EnableBackendRouting = true; return nil },
		},
		{
			longs:   []string{"--backend_protocol"},
			kind:    kindOptString,
			choices: []string{"http1", "http2", "grpc"},
			apply:   func(a *Args, val string) error { return setOptStr(&a.BackendProtocol)(a, val) },
		},
		{
			longs: []string{"--http_port"},
			kind:  kindOptInt,
			apply: func(a *Args, val string) error {
				n, err := strconv.Atoi(val)
				if err != nil {
					return err
				}
				a.HttpPort = &n
				return nil
			},
		},
		{
			longs: []string{"--http2_port"},
			kind:  kindOptInt,
			apply: func(a *Args, val string) error {
				n, err := strconv.Atoi(val)
				if err != nil {
					return err
				}
				a.Http2Port = &n
				return nil
			},
		},
		{
			longs: []string{"--ssl_port"},
			kind:  kindOptInt,
			apply: func(a *Args, val string) error {
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
			apply: func(a *Args, val string) error { return setOptStr(&a.Dns)(a, val) },
		},
		{
			short: "-t",
			longs: []string{"--tls_mutual_auth"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.TlsMutualAuth = true; return nil },
		},
		{
			longs: []string{"--ssl_protocols"},
			kind:  kindAppend,
			apply: func(a *Args, val string) error { a.SslProtocols = append(a.SslProtocols, val); return nil },
		},
		{
			longs: []string{"--enable_grpc_backend_ssl"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.EnableGrpcBackendSsl = true; return nil },
		},
		{
			longs: []string{"--grpc_backend_ssl_root_certs_file"},
			kind:  kindString,
			apply: func(a *Args, val string) error { a.GrpcBackendSslRootCertsFile = val; return nil },
		},
		{
			longs: []string{"--ssl_client_cert_path"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.SslClientCertPath)(a, val) },
		},
		{
			longs: []string{"--ssl_client_root_certs_file"},
			kind:  kindOptString,
			apply: func(a *Args, val string) error { return setOptStr(&a.SslClientRootCertsFile)(a, val) },
		},
		{
			longs: []string{"--on_serverless"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.OnServerless = true; return nil },
		},
		{
			longs: []string{"--serverless"},
			kind:  kindStoreTrue,
			apply: func(a *Args, _ string) error { a.ServerlessEntrypoint = true; return nil },
		},
	}
}

func resolveOption(defs []*flagDef, name string) (*flagDef, error) {
	if strings.HasPrefix(name, "--") {
		for _, d := range defs {
			for _, l := range d.longs {
				if l == name {
					return d, nil
				}
			}
		}
		// Unambiguous prefix abbreviation matching (argparse allow_abbrev=True).
		var matched []*flagDef
		for _, d := range defs {
			for _, l := range d.longs {
				if strings.HasPrefix(l, name) {
					matched = append(matched, d)
					break
				}
			}
		}
		if len(matched) == 1 {
			return matched[0], nil
		}
		if len(matched) > 1 {
			return nil, fmt.Errorf("ambiguous option: %s", name)
		}
		return nil, fmt.Errorf("unrecognized arguments: %s", name)
	}
	if strings.HasPrefix(name, "-") && len(name) >= 2 {
		for _, d := range defs {
			if d.short == name {
				return d, nil
			}
		}
		return nil, fmt.Errorf("unrecognized arguments: %s", name)
	}
	return nil, fmt.Errorf("unrecognized arguments: %s", name)
}

func isOptionToken(defs []*flagDef, tok string) bool {
	if !strings.HasPrefix(tok, "-") || tok == "-" {
		return false
	}
	optName := tok
	if idx := strings.IndexByte(tok, '='); idx != -1 {
		optName = tok[:idx]
	}
	if d, err := resolveOption(defs, optName); err == nil && d != nil {
		return true
	}
	// Short option with attached argument (e.g. -sfoo).
	if !strings.HasPrefix(tok, "--") && len(tok) > 2 {
		shortPrefix := tok[:2]
		for _, d := range defs {
			if d.short == shortPrefix {
				return true
			}
		}
	}
	return false
}

// PrintUsage writes the help message for startproxy to w.
func PrintUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: startproxy [-h] [options...]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "options:")
	fmt.Fprintln(w, "  -h, --help                               show this help message and exit")
	defs := buildFlagDefs()
	for _, d := range defs {
		var flagNames []string
		if d.short != "" {
			flagNames = append(flagNames, d.short)
		}
		flagNames = append(flagNames, d.longs...)
		flagsStr := strings.Join(flagNames, ", ")
		if len(d.choices) > 0 {
			flagsStr += fmt.Sprintf(" {%s}", strings.Join(d.choices, ","))
		}
		fmt.Fprintf(w, "  %-40s\n", flagsStr)
	}
}

// ParseArgs parses command-line flags using argparse-compatible rules.
func ParseArgs(argv []string) (*Args, error) {
	args := DefaultArgs()
	defs := buildFlagDefs()

	for i := 0; i < len(argv); i++ {
		rawTok := argv[i]
		tok := rawTok
		if trimmed := strings.TrimSpace(rawTok); strings.HasPrefix(trimmed, "-") {
			tok = trimmed
		}
		if tok == "-h" || tok == "--help" {
			return nil, ErrHelpRequested
		}
		if !strings.HasPrefix(tok, "-") || tok == "-" {
			return nil, fmt.Errorf("unrecognized arguments: %s", rawTok)
		}

		var optName string
		var inlineVal string
		hasInline := false
		if idx := strings.IndexByte(tok, '='); idx != -1 {
			optName = tok[:idx]
			inlineVal = tok[idx+1:]
			hasInline = true
		} else if !strings.HasPrefix(tok, "--") && len(tok) > 2 {
			// Short flag with attached value, e.g. -sbar.
			shortCandidate := tok[:2]
			if d, err := resolveOption(defs, shortCandidate); err == nil && d != nil && d.kind != kindStoreTrue && d.kind != kindStoreFalse {
				optName = shortCandidate
				inlineVal = tok[2:]
				hasInline = true
			} else {
				optName = tok
			}
		} else {
			optName = tok
		}

		def, err := resolveOption(defs, optName)
		if err != nil {
			return nil, err
		}

		if def.kind == kindStoreTrue || def.kind == kindStoreFalse {
			if hasInline {
				return nil, fmt.Errorf("argument %s: ignored explicit argument %q", optName, inlineVal)
			}
			if err := def.apply(args, ""); err != nil {
				return nil, err
			}
			continue
		}

		var val string
		if hasInline {
			val = inlineVal
		} else {
			if i+1 >= len(argv) {
				return nil, fmt.Errorf("argument %s: expected one argument", optName)
			}
			nextTok := argv[i+1]
			if isOptionToken(defs, nextTok) {
				return nil, fmt.Errorf("argument %s: expected one argument", optName)
			}
			val = nextTok
			i++
		}

		if len(def.choices) > 0 {
			validChoice := false
			for _, c := range def.choices {
				if c == val {
					validChoice = true
					break
				}
			}
			if !validChoice {
				return nil, fmt.Errorf("argument %s: invalid choice: %q (choose from %v)", optName, val, def.choices)
			}
		}

		if err := def.apply(args, val); err != nil {
			return nil, fmt.Errorf("argument %s: %v", optName, err)
		}
	}

	return args, nil
}

// GenBootstrapConf generates the command-line slice to invoke bin/bootstrap.
func GenBootstrapConf(args *Args) []string {
	cmd := []string{BootstrapCmd, "--logtostderr"}
	cmd = append(cmd, "--admin_port", strconv.Itoa(args.StatusPort))
	if args.HttpRequestTimeoutS != nil && *args.HttpRequestTimeoutS != 0 {
		cmd = append(cmd, "--http_request_timeout_s", strconv.Itoa(*args.HttpRequestTimeoutS))
	}
	if args.AdsNamedPipe != nil && *args.AdsNamedPipe != "" {
		cmd = append(cmd, "--ads_named_pipe", *args.AdsNamedPipe)
	}
	bootstrapFile := DefaultConfigDir + BootstrapConfig
	cmd = append(cmd, bootstrapFile)
	return cmd
}

// EnforceConflictArgs checks for conflicting or missing flags and updates defaults when needed.
func EnforceConflictArgs(args *Args) error {
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
		if args.TracingProjectId == "" {
			args.DisableTracing = true
		}
	}

	if (args.AccessLog == nil || *args.AccessLog == "") && (args.AccessLogFormat != nil && *args.AccessLogFormat != "") {
		return fmt.Errorf("Flag --access_log_format has to be used together with --access_log.")
	}

	if args.SslPort != nil && *args.SslPort != 0 && args.SslServerCertPath != nil && *args.SslServerCertPath != "" {
		return fmt.Errorf("Flag --ssl_port is going to be deprecated, please use --ssl_server_cert_path only.")
	}
	if args.TlsMutualAuth && ((args.SslBackendClientCertPath != nil && *args.SslBackendClientCertPath != "") || (args.SslClientCertPath != nil && *args.SslClientCertPath != "")) {
		return fmt.Errorf("Flag --tls_mutual_auth is going to be deprecated, please use --ssl_backend_client_cert_path only.")
	}
	if ((args.SslBackendClientRootCertsFile != nil && *args.SslBackendClientRootCertsFile != "") || (args.SslClientRootCertsFile != nil && *args.SslClientRootCertsFile != "")) && args.EnableGrpcBackendSsl {
		return fmt.Errorf("Flag --enable_grpc_backend_ssl are going to be deprecated, please use --ssl_backend_client_root_certs_file only.")
	}
	if args.GenerateSelfSignedCert && args.SslServerCertPath != nil && *args.SslServerCertPath != "" {
		return fmt.Errorf("Flag --generate_self_signed_cert and --ssl_server_cert_path cannot be used simutaneously.")
	}

	var portFlags []string
	portNum := DefaultListenerPort
	if args.HttpPort != nil && *args.HttpPort != 0 {
		portFlags = append(portFlags, "--http_port")
		portNum = *args.HttpPort
	}
	if args.Http2Port != nil && *args.Http2Port != 0 {
		portFlags = append(portFlags, "--http2_port")
		portNum = *args.Http2Port
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

	if args.DnsResolverAddresses != nil && *args.DnsResolverAddresses != "" && args.Dns != nil && *args.Dns != "" {
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

// GenerateSelfSignedCertFiles generates a 2048-bit RSA self-signed X.509 certificate
// and private key in certDir (server.crt and server.key), replacing the openssl CLI call.
func GenerateSelfSignedCertFiles(certDir string) error {
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return fmt.Errorf("failed to create cert directory %s: %w", certDir, err)
	}

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA private key: %w", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	now := time.Now()
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "localhost",
		},
		NotBefore:             now,
		NotAfter:              now.Add(3650 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		return fmt.Errorf("failed to create self-signed X.509 certificate: %w", err)
	}

	certPath := filepath.Join(certDir, "server.crt")
	certOut, err := os.OpenFile(certPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open %s for writing: %w", certPath, err)
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return fmt.Errorf("failed to write PEM certificate: %w", err)
	}

	keyPath := filepath.Join(certDir, "server.key")
	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open %s for writing: %w", keyPath, err)
	}
	defer keyOut.Close()
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privKey)}); err != nil {
		return fmt.Errorf("failed to write PEM private key: %w", err)
	}

	return nil
}

// GenProxyConfig builds the command-line slice for bin/configmanager.
func GenProxyConfig(args *Args) ([]string, error) {
	if args.ServiceAccountKey == nil {
		if envVal, ok := os.LookupEnv(GoogleCredsKey); ok {
			args.ServiceAccountKey = &envVal
		}
	} else {
		if _, ok := os.LookupEnv(GoogleCredsKey); !ok {
			_ = os.Setenv(GoogleCredsKey, *args.ServiceAccountKey)
		}
	}

	if err := EnforceConflictArgs(args); err != nil {
		return nil, err
	}

	proxyConf := []string{
		ConfigManagerBin,
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
		proxyConf = append(proxyConf, "--envoy_xff_num_trusted_hops", strconv.Itoa(ServerlessXffNumTrustedHops))
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

	if args.HttpPort != nil && *args.HttpPort != 0 {
		proxyConf = append(proxyConf, "--listener_port", strconv.Itoa(*args.HttpPort))
	}
	if args.Http2Port != nil && *args.Http2Port != 0 {
		proxyConf = append(proxyConf, "--listener_port", strconv.Itoa(*args.Http2Port))
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

	if args.TlsMutualAuth {
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
		if err := GenerateSelfSignedCertFiles(SelfSignedCertDir); err != nil {
			return nil, fmt.Errorf("Failed to create self-signed cert: %w", err)
		}
		proxyConf = append(proxyConf, "--ssl_server_cert_path", DefaultSelfSignedCertDir)
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
	if args.HttpRequestTimeoutS != nil && *args.HttpRequestTimeoutS != 0 {
		proxyConf = append(proxyConf, "--http_request_timeout_s", strconv.Itoa(*args.HttpRequestTimeoutS))
	}
	if args.ServiceControlUrl != nil && *args.ServiceControlUrl != "" {
		proxyConf = append(proxyConf, "--service_control_url", *args.ServiceControlUrl)
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
	if args.ServiceControlEnableApiKeyUidReporting {
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
		if args.TracingProjectId != "" {
			proxyConf = append(proxyConf, "--tracing_project_id", args.TracingProjectId)
		}
		if args.TracingIncomingContext != "" {
			proxyConf = append(proxyConf, "--tracing_incoming_context", args.TracingIncomingContext)
		}
		if args.TracingOutgoingContext != "" {
			proxyConf = append(proxyConf, "--tracing_outgoing_context", args.TracingOutgoingContext)
		}
		if args.CloudTraceUrlOverride != "" {
			proxyConf = append(proxyConf, "--tracing_stackdriver_address", args.CloudTraceUrlOverride)
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
		proxyConf = append(proxyConf, "--compute_platform_override", ServerlessPlatform)
	}
	if args.BackendDnsLookupFamily != nil && *args.BackendDnsLookupFamily != "" {
		proxyConf = append(proxyConf, "--backend_dns_lookup_family", *args.BackendDnsLookupFamily)
	}
	if args.DnsResolverAddresses != nil && *args.DnsResolverAddresses != "" {
		proxyConf = append(proxyConf, "--dns_resolver_addresses", *args.DnsResolverAddresses)
	}
	if args.Dns != nil && *args.Dns != "" {
		proxyConf = append(proxyConf, "--dns_resolver_addresses", *args.Dns)
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

// GenEnvoyArgs builds the command-line slice for bin/envoy.
func GenEnvoyArgs(args *Args) []string {
	cmd := []string{
		EnvoyBin,
		"-c", DefaultConfigDir + BootstrapConfig,
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

func startSubprocess(cmdArgs []string) (*exec.Cmd, <-chan error, error) {
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	return cmd, waitCh, nil
}

// Run executes the ESPv2 startup and supervision loop.
func Run(argv []string) int {
	// Check if invoked in serverless mode via --serverless flag or binary name.
	if len(argv) > 0 && argv[0] == "--serverless" {
		flags, port, serveErrPage, err := BuildServerlessFlags()
		if err != nil {
			if serveErrPage && port != "" {
				log.Println(err.Error())
				if serveErr := ServeErrorMsg(port, err.Error()); serveErr != nil {
					log.Printf("Failed to serve error message: %v", serveErr)
				}
				return 1
			}
			log.Printf("Serverless startup error: %v", err)
			return 1
		}
		argv = append(flags, argv[1:]...)
	}

	args, err := ParseArgs(argv)
	if err != nil {
		if errors.Is(err, ErrHelpRequested) {
			PrintUsage(os.Stdout)
			return 0
		}
		fmt.Fprintf(os.Stderr, "startproxy: error: %v\n", err)
		return 1
	}

	if args.ServerlessEntrypoint {
		flags, port, serveErrPage, err := BuildServerlessFlags()
		if err != nil {
			if serveErrPage && port != "" {
				log.Println(err.Error())
				if serveErr := ServeErrorMsg(port, err.Error()); serveErr != nil {
					log.Printf("Failed to serve error message: %v", serveErr)
				}
				return 1
			}
			log.Printf("Serverless startup error: %v", err)
			return 1
		}
		args, err = ParseArgs(flags)
		if err != nil {
			if errors.Is(err, ErrHelpRequested) {
				PrintUsage(os.Stdout)
				return 0
			}
			fmt.Fprintf(os.Stderr, "startproxy: error: %v\n", err)
			return 1
		}
	}

	proxyConf, err := GenProxyConfig(args)
	if err != nil {
		log.Printf("ERROR: %v", err)
		return 1
	}

	// Register for shutdown signals before starting any child so a signal that
	// arrives during startup is forwarded instead of killing startproxy with
	// the default action and orphaning the children.
	sigCh := make(chan os.Signal, 2)
	notifySignals(sigCh)
	defer stopSignals(sigCh)

	fmt.Printf("Starting Config Manager with args: %v\n", proxyConf)
	cmCmd, cmWaitCh, err := startSubprocess(proxyConf)
	if err != nil {
		log.Printf("Failed to start Config Manager: %v", err)
		return 1
	}

	bootstrapCmdArgs := GenBootstrapConf(args)
	fmt.Printf("%v\n", bootstrapCmdArgs)
	bootstrapCmd := exec.Command(bootstrapCmdArgs[0], bootstrapCmdArgs[1:]...)
	bootstrapCmd.Stdout = os.Stdout
	bootstrapCmd.Stderr = os.Stderr
	_ = bootstrapCmd.Run()

	envoyArgs := GenEnvoyArgs(args)
	fmt.Printf("Starting Envoy with args: %v\n", envoyArgs)
	envoyCmd, envoyWaitCh, err := startSubprocess(envoyArgs)
	if err != nil {
		log.Printf("Failed to start Envoy: %v", err)
		if cmCmd.Process != nil {
			_ = cmCmd.Process.Kill()
		}
		return 1
	}

	return supervise(
		&child{name: "Config Manager", cmd: cmCmd, waitCh: cmWaitCh},
		&child{name: "Envoy", cmd: envoyCmd, waitCh: envoyWaitCh},
		sigCh,
	)
}

// child is a supervised subprocess.
type child struct {
	name   string
	cmd    *exec.Cmd
	waitCh <-chan error
	exited bool
}

func (c *child) signal(sig syscall.Signal) {
	if c.exited || c.cmd.Process == nil {
		return
	}
	if sig == syscall.SIGKILL {
		log.Printf("INFO: Killing process: pid=%d", c.cmd.Process.Pid)
		_ = c.cmd.Process.Kill()
		return
	}
	log.Printf("INFO: sending TERM to PID=%d", c.cmd.Process.Pid)
	if err := c.cmd.Process.Signal(sig); err != nil {
		log.Printf("ERROR: error sending TERM to PID=%d continuing", c.cmd.Process.Pid)
	}
}

// supervise monitors Config Manager and Envoy until both have exited.
//
//   - If either child exits unexpectedly, the other is killed immediately.
//   - On SIGTERM/SIGINT, SIGTERM is forwarded to both children and they are
//     given ShutdownGracePeriod to exit on their own; one child finishing its
//     shutdown first must not cause the other to be SIGKILLed mid-shutdown.
//     Survivors are SIGKILLed when the grace period expires.
//
// It always returns 1, matching the behavior of the original start_proxy.py.
func supervise(cm, envoy *child, sigCh <-chan os.Signal) int {
	children := []*child{cm, envoy}
	terminating := false
	var graceTimeout <-chan time.Time

	onExit := func(c *child) (done bool) {
		c.exited = true
		c.waitCh = nil // never selected again
		if terminating {
			log.Printf("INFO: %s exited", c.name)
			return cm.exited && envoy.exited
		}
		other := envoy
		if c == envoy {
			other = cm
		}
		log.Printf("FATAL: %s is down, killing %s process.", c.name, other.name)
		other.signal(syscall.SIGKILL)
		return true
	}

	for {
		select {
		case sig := <-sigCh:
			log.Printf("WARNING: got signal: %v", sig)
			for _, c := range children {
				c.signal(syscall.SIGTERM)
			}
			if !terminating {
				terminating = true
				graceTimeout = time.After(ShutdownGracePeriod)
			}
		case <-cm.waitCh:
			if onExit(cm) {
				return 1
			}
		case <-envoy.waitCh:
			if onExit(envoy) {
				return 1
			}
		case <-graceTimeout:
			log.Printf("WARNING: child processes did not exit within %v of signal", ShutdownGracePeriod)
			for _, c := range children {
				c.signal(syscall.SIGKILL)
			}
			return 1
		}
	}
}
