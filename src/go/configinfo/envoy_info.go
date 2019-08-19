package configinfo

import (
	"time"

	"cloudesf.googlesource.com/gcpproxy/src/go/flags"
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

// EnvoyConfigOptionsFromFlags returns a new EnvoyConfigOptions derived from passed in flag values.
func EnvoyConfigOptionsFromFlags() EnvoyConfigOptions {
	return EnvoyConfigOptions{
		BackendProtocol:               *flags.BackendProtocol,
		CorsAllowCredentials:          *flags.CorsAllowCredentials,
		CorsAllowHeaders:              *flags.CorsAllowHeaders,
		CorsAllowMethods:              *flags.CorsAllowMethods,
		CorsAllowOrigin:               *flags.CorsAllowOrigin,
		CorsAllowOriginRegex:          *flags.CorsAllowOriginRegex,
		CorsExposeHeaders:             *flags.CorsExposeHeaders,
		CorsPreset:                    *flags.CorsPreset,
		EnableBackendRouting:          *flags.EnableBackendRouting,
		BackendDnsLookupFamily:        *flags.BackendDnsLookupFamily,
		ClusterConnectTimeout:         *flags.ClusterConnectTimeout,
		ClusterAddress:                *flags.ClusterAddress,
		ListenerAddress:               *flags.ListenerAddress,
		Node:                          *flags.Node,
		ServiceManagementURL:          *flags.ServiceManagementURL,
		ClusterPort:                   *flags.ClusterPort,
		ListenerPort:                  *flags.ListenerPort,
		NonGCP:                        *flags.NonGCP,
		ServiceAccountKey:             *flags.ServiceAccountKey,
		SkipJwtAuthnFilter:            *flags.SkipJwtAuthnFilter,
		SkipServiceControlFilter:      *flags.SkipServiceControlFilter,
		EnvoyUseRemoteAddress:         *flags.EnvoyUseRemoteAddress,
		EnvoyXffNumTrustedHops:        *flags.EnvoyXffNumTrustedHops,
		LogJwtPayloads:                *flags.LogJwtPayloads,
		LogRequestHeaders:             *flags.LogRequestHeaders,
		LogResponseHeaders:            *flags.LogResponseHeaders,
		SuppressEnvoyHeaders:          *flags.SuppressEnvoyHeaders,
		ServiceControlNetworkFailOpen: *flags.ServiceControlNetworkFailOpen,
		JwksCacheDurationInS:          *flags.JwksCacheDurationInS,
		ScCheckTimeoutMs:              *flags.ScCheckTimeoutMs,
		ScQuotaTimeoutMs:              *flags.ScQuotaTimeoutMs,
		ScReportTimeoutMs:             *flags.ScReportTimeoutMs,
		ScCheckRetries:                *flags.ScCheckRetries,
		ScQuotaRetries:                *flags.ScQuotaRetries,
		ScReportRetries:               *flags.ScReportRetries,
		EnableTracing:                 *flags.EnableTracing,
	}
}
