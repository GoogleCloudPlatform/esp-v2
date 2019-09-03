// Copyright 2019 Google Cloud Platform Proxy Authors
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

package configinfo

import (
	"time"
)

// CommonOptions describes the possible overrides used by both the ADS bootstrapper and the config generator.
// By defining all the common options in one struct, we prevent duplicate flag initialization and reduce repeated code.
type CommonOptions struct {
	// Flags for envoy
	AdminPort int

	// Flags for tracing
	EnableTracing             bool
	TracingProjectId          string
	TracingStackdriverAddress string
	TracingSamplingRate       float64
	TracingIncomingContext    string
	TracingOutgoingContext    string

	// Flags for metadata
	NonGCP                 bool
	MetadataFetcherTimeout time.Duration
}

// AdsBootstrapperOptions describes the possible overrides used by the ADS bootstrapper to create the envoy bootstrap config.
type AdsBootstrapperOptions struct {
	CommonOptions

	// Flags for ADS
	AdsConnectTimeout time.Duration
	DiscoveryAddress  string
}

// EnvoyConfigOptions describes the possible overrides for the service config to envoy config translation.
// TODO(nareddyt): This needs to be renamed to ConfigGeneratorOptions in a later CL
// Note that this rename is difficult because it will break managed api gateway team
type EnvoyConfigOptions struct {
	CommonOptions

	// Service Management related configurations. Must be set.
	BackendProtocol string

	// Cors related configurations.
	CorsAllowCredentials bool
	CorsAllowHeaders     string
	CorsAllowMethods     string
	CorsAllowOrigin      string
	CorsAllowOriginRegex string
	CorsExposeHeaders    string
	CorsPreset           string

	// Backend routing configurations.
	EnableBackendRouting   bool
	BackendDnsLookupFamily string

	// Envoy specific configurations.
	ClusterConnectTimeout time.Duration

	// Network related configurations.
	ClusterAddress       string
	ListenerAddress      string
	Node                 string
	ServiceManagementURL string
	ClusterPort          int
	ListenerPort         int

	// Flags for non_gcp deployment.
	ServiceAccountKey string

	// Flags for testing purpose.
	SkipJwtAuthnFilter       bool
	SkipServiceControlFilter bool

	// Envoy configurations.
	EnvoyUseRemoteAddress  bool
	EnvoyXffNumTrustedHops int

	LogJwtPayloads     string
	LogRequestHeaders  string
	LogResponseHeaders string

	SuppressEnvoyHeaders bool

	ServiceControlNetworkFailOpen bool

	JwksCacheDurationInS int

	ScCheckTimeoutMs  int
	ScQuotaTimeoutMs  int
	ScReportTimeoutMs int

	ScCheckRetries  int
	ScQuotaRetries  int
	ScReportRetries int
}

// DefaultCommonOptions returns CommonOptions with default values.
//
// The default values are expected to match the default values from the flags.
func DefaultCommonOptions() CommonOptions {
	return CommonOptions{
		AdminPort:                 8001,
		EnableTracing:             false,
		MetadataFetcherTimeout:    5 * time.Second,
		NonGCP:                    false,
		TracingProjectId:          "",
		TracingStackdriverAddress: "",
		TracingSamplingRate:       0.001,
		TracingIncomingContext:    "",
		TracingOutgoingContext:    "",
	}
}

// DefaultAdsBootstrapperOptions returns AdsBootstrapperOptions with default values.
//
// The default values are expected to match the default values from the flags.
func DefaultAdsBootstrapperOptions() AdsBootstrapperOptions {
	return AdsBootstrapperOptions{
		CommonOptions:     DefaultCommonOptions(),
		AdsConnectTimeout: 10 * time.Second,
		DiscoveryAddress:  "127.0.0.1:8790",
	}
}

// DefaultEnvoyConfigOptions returns EnvoyConfigOptions with default values.
//
// The default values are expected to match the default values from the flags.
func DefaultEnvoyConfigOptions() EnvoyConfigOptions {

	return EnvoyConfigOptions{
		CommonOptions:                 DefaultCommonOptions(),
		BackendDnsLookupFamily:        "auto",
		BackendProtocol:               "", // Required flag with no default
		ClusterAddress:                "127.0.0.1",
		ClusterConnectTimeout:         20 * time.Second,
		ClusterPort:                   8082,
		CorsAllowCredentials:          false,
		CorsAllowHeaders:              "",
		CorsAllowMethods:              "",
		CorsAllowOrigin:               "",
		CorsAllowOriginRegex:          "",
		CorsExposeHeaders:             "",
		CorsPreset:                    "",
		EnableBackendRouting:          false,
		EnvoyUseRemoteAddress:         false,
		EnvoyXffNumTrustedHops:        2,
		JwksCacheDurationInS:          300,
		ListenerAddress:               "0.0.0.0",
		ListenerPort:                  8080,
		LogJwtPayloads:                "",
		LogRequestHeaders:             "",
		LogResponseHeaders:            "",
		Node:                          "api_proxy",
		ServiceAccountKey:             "",
		ServiceControlNetworkFailOpen: true,
		ServiceManagementURL:          "https://servicemanagement.googleapis.com",
		ScCheckRetries:                -1,
		ScCheckTimeoutMs:              0,
		ScQuotaRetries:                -1,
		ScQuotaTimeoutMs:              0,
		ScReportRetries:               -1,
		ScReportTimeoutMs:             0,
		SkipJwtAuthnFilter:            false,
		SkipServiceControlFilter:      false,
		SuppressEnvoyHeaders:          false,
	}
}
