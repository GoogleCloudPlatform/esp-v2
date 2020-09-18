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
	CorsPreset           string

	// Backend routing configurations.
	BackendDnsLookupFamily string

	// Envoy specific configurations.
	ClusterConnectTimeout time.Duration

	// Full URI to the backend: scheme, address/hostname, port
	BackendAddress string

	// Network related configurations.
	ListenerAddress                  string
	Healthz                          string
	ServiceManagementURL             string
	ServiceControlURL                string
	ListenerPort                     int
	SslServerCertPath                string
	SslMinimumProtocol               string
	SslMaximumProtocol               string
	EnableHSTS                       bool
	SslSidestreamClientRootCertsPath string
	SslBackendClientCertPath         string
	SslBackendClientRootCertsPath    string
	DnsResolverAddresses             string

	// Flags for non_gcp deployment.
	ServiceAccountKey string
	TokenAgentPort    uint

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
	ServiceControlNetworkFailOpen bool
	EnableGrpcForHttp1            bool

	JwksCacheDurationInS int

	ScCheckTimeoutMs  int
	ScQuotaTimeoutMs  int
	ScReportTimeoutMs int

	ScCheckRetries  int
	ScQuotaRetries  int
	ScReportRetries int

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
		ClusterConnectTimeout:            20 * time.Second,
		EnvoyXffNumTrustedHops:           2,
		JwksCacheDurationInS:             300,
		ListenerAddress:                  "0.0.0.0",
		ListenerPort:                     8080,
		TokenAgentPort:                   8791,
		SslSidestreamClientRootCertsPath: util.DefaultRootCAPaths,
		SslBackendClientRootCertsPath:    util.DefaultRootCAPaths,
		SuppressEnvoyHeaders:             true,
		ServiceControlNetworkFailOpen:    true,
		EnableGrpcForHttp1:               true,
		ServiceManagementURL:             "https://servicemanagement.googleapis.com",
		ServiceControlURL:                "https://servicecontrol.googleapis.com",
		ScCheckRetries:                   -1,
		ScQuotaRetries:                   -1,
		ScReportRetries:                  -1,
	}
}
