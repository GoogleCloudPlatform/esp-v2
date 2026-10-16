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

// proxyArgs holds all parsed command-line arguments for startproxy.
type proxyArgs struct {
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
	ServiceControlEnableAPIKeyUIDReporting        bool
	DisableJwksAsyncFetch                         bool
	JwksAsyncFetchFastListener                    bool
	JwtCacheSize                                  *string
	JwksCacheDurationInS                          *string
	JwksFetchNumRetries                           *string
	JwksFetchRetryBackOffBaseIntervalMs           *string
	JwksFetchRetryBackOffMaxIntervalMs            *string
	JwtPadForwardPayloadHeader                    bool
	DisableJwtAudienceServiceNameCheck            bool
	HTTPRequestTimeoutS                           *int
	ServiceControlURL                             *string
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
	TracingProjectID                              string
	TracingSampleRate                             *string
	DisableCloudTraceAutoSampling                 bool
	TracingIncomingContext                        string
	TracingOutgoingContext                        string
	CloudTraceURLOverride                         string
	NonGcp                                        bool
	ServiceAccountKey                             *string
	EnableApplicationDefaultCredentials           bool
	DNSResolverAddresses                          *string
	BackendDNSLookupFamily                        *string
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
	HTTPPort                                      *int
	HTTP2Port                                     *int
	SslPort                                       *int
	DNS                                           *string
	TLSMutualAuth                                 bool
	SslProtocols                                  []string
	EnableGrpcBackendSsl                          bool
	GrpcBackendSslRootCertsFile                   string
	SslClientCertPath                             *string
	SslClientRootCertsFile                        *string
	OnServerless                                  bool
	ServerlessEntrypoint                          bool
}

// defaultArgs returns an proxyArgs struct populated with the default values from start_proxy.py.
func defaultArgs() *proxyArgs {
	return &proxyArgs{
		Service:                                "",
		Version:                                "",
		Backend:                                defaultBackend,
		StatusPort:                             0,
		RolloutStrategy:                        defaultRolloutStrategy,
		CorsAllowOrigin:                        "*",
		CorsAllowOriginRegex:                   "",
		CorsAllowMethods:                       "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		CorsAllowHeaders:                       "DNT,User-Agent,X-User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization",
		CorsExposeHeaders:                      "Content-Length,Content-Range",
		CorsMaxAge:                             "480h",
		ServiceControlNetworkFailPolicy:        "open",
		ServiceControlEnableAPIKeyUIDReporting: true,
		TracingProjectID:                       "",
		TracingIncomingContext:                 "",
		TracingOutgoingContext:                 "",
		CloudTraceURLOverride:                  "",
		GrpcBackendSslRootCertsFile:            "/etc/nginx/trusted-ca-certificates.crt",
	}
}
