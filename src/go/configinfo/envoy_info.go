package configinfo

import (
	"time"
)

// EnvoyConfigOptions describes the possible overrides for the service config to envoy config translation.
type EnvoyConfigOptions struct {
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
	NonGCP            bool
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

	EnableTracing bool
}

// DefaultEnvoyConfigOptions returns EnvoyConfigOptions with default values.
//
// The default values are expected to match the default values from the flags.
func DefaultEnvoyConfigOptions() EnvoyConfigOptions {
	return EnvoyConfigOptions{
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
		EnableTracing:                 false,
		EnvoyUseRemoteAddress:         false,
		EnvoyXffNumTrustedHops:        2,
		JwksCacheDurationInS:          300,
		ListenerAddress:               "0.0.0.0",
		ListenerPort:                  8080,
		LogJwtPayloads:                "",
		LogRequestHeaders:             "",
		LogResponseHeaders:            "",
		Node:                          "api_proxy",
		NonGCP:                        false,
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
