// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package options

import (
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"

	commonpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v9/http/common"
)

// ConfigGeneratorOptions describes the possible overrides for the service config to envoy config translation.
// Note that this rename is difficult because it will break managed api gateway team
type ConfigGeneratorOptions struct {
	CommonOptions

	// Cors related configurations.
	CorsAllowCredentials bool
	CorsAllowHeaders     string
	CorsAllowMethods     string
	CorsAllowOrigin      string
	CorsAllowOriginRegex string
	CorsExposeHeaders    string
	CorsMaxAge           time.Duration
	CorsPreset           string

	// Backend routing configurations.
	BackendDnsLookupFamily string

	// Envoy specific configurations.
	ClusterConnectTimeout time.Duration
	StreamIdleTimeout     time.Duration

	// Full URI to the backend: scheme, address/hostname, port
	BackendAddress               string
	EnableBackendAddressOverride bool

	// Network related configurations.
	ListenerAddress                  string
	Healthz                          string
	ServiceManagementURL             string
	ServiceControlURL                string
	ListenerPort                     int
	SslServerCertPath                string
	SslServerCipherSuites            string
	SslMinimumProtocol               string
	SslMaximumProtocol               string
	EnableHSTS                       bool
	SslSidestreamClientRootCertsPath string
	SslBackendClientCertPath         string
	SslBackendClientRootCertsPath    string
	SslBackendClientCipherSuites     string
	DnsResolverAddresses             string

	// Headers manipulation:
	AddRequestHeaders     string
	AppendRequestHeaders  string
	AddResponseHeaders    string
	AppendResponseHeaders string

	// Flags for non_gcp deployment.
	ServiceAccountKey string
	TokenAgentPort    uint

	// Flags for external calls.
	DisableOidcDiscovery    bool
	DependencyErrorBehavior string

	// Flags for testing purpose.
	SkipJwtAuthnFilter       bool
	SkipServiceControlFilter bool

	// Envoy configurations.
	AccessLog       string
	AccessLogFormat string

	EnvoyUseRemoteAddress  bool
	EnvoyXffNumTrustedHops int

	LogJwtPayloads            string
	LogRequestHeaders         string
	LogResponseHeaders        string
	MinStreamReportIntervalMs uint64

	SuppressEnvoyHeaders          bool
	UnderscoresInHeaders          bool
	NormalizePath                 bool
	MergeSlashesInPath            bool
	DisallowEscapedSlashesInPath  bool
	ServiceControlNetworkFailOpen bool
	EnableGrpcForHttp1            bool
	ConnectionBufferLimitBytes    int

	JwksCacheDurationInS int

	ScCheckTimeoutMs  int
	ScQuotaTimeoutMs  int
	ScReportTimeoutMs int

	BackendRetryOns      string
	BackendRetryNum      uint
	BackendPerTryTimeout time.Duration
	ScCheckRetries       int
	ScQuotaRetries       int
	ScReportRetries      int

	ComputePlatformOverride string

	TranscodingAlwaysPrintPrimitiveFields   bool
	TranscodingAlwaysPrintEnumsAsInts       bool
	TranscodingPreserveProtoFieldNames      bool
	TranscodingIgnoreQueryParameters        string
	TranscodingIgnoreUnknownQueryParameters bool
}

// DefaultConfigGeneratorOptions returns ConfigGeneratorOptions with default values.
//
// The default values are expected to match the default values from the flags.
func DefaultConfigGeneratorOptions() ConfigGeneratorOptions {

	return ConfigGeneratorOptions{
		CommonOptions:                    DefaultCommonOptions(),
		BackendDnsLookupFamily:           "auto",
		BackendAddress:                   fmt.Sprintf("http://%s:8082", util.LoopbackIPv4Addr),
		EnableBackendAddressOverride:     false,
		ClusterConnectTimeout:            20 * time.Second,
		StreamIdleTimeout:                util.DefaultIdleTimeout,
		EnvoyXffNumTrustedHops:           2,
		JwksCacheDurationInS:             300,
		ListenerAddress:                  "0.0.0.0",
		ListenerPort:                     8080,
		TokenAgentPort:                   8791,
		DisableOidcDiscovery:             false,
		DependencyErrorBehavior:          commonpb.DependencyErrorBehavior_BLOCK_INIT_ON_ANY_ERROR.String(),
		SslSidestreamClientRootCertsPath: util.DefaultRootCAPaths,
		SslBackendClientRootCertsPath:    util.DefaultRootCAPaths,
		SuppressEnvoyHeaders:             true,
		NormalizePath:                    true,
		MergeSlashesInPath:               true,
		DisallowEscapedSlashesInPath:     false,
		ServiceControlNetworkFailOpen:    true,
		EnableGrpcForHttp1:               true,
		ConnectionBufferLimitBytes:       -1,
		ServiceManagementURL:             "https://servicemanagement.googleapis.com",
		ServiceControlURL:                "https://servicecontrol.googleapis.com",
		BackendRetryNum:                  1,
		BackendRetryOns:                  "reset,connect-failure,refused-stream",
		ScCheckRetries:                   -1,
		ScQuotaRetries:                   -1,
		ScReportRetries:                  -1,
		CorsMaxAge:                       480 * time.Hour,
	}
}
