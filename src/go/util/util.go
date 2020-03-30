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

package util

import "time"

const (
	// HTTP filter names.

	// Buffer HTTP filter
	Buffer = "envoy.filters.http.buffer"
	// CORS HTTP filter
	CORS = "envoy.filters.http.cors"
	// GRPCJSONTranscoder HTTP filter
	GRPCJSONTranscoder = "envoy.filters.http.grpc_json_transcoder"
	// GRPCWeb HTTP filter
	GRPCWeb = "envoy.filters.http.grpc_web"
	// Router HTTP filter
	Router = "envoy.filters.http.router"
	// Health checking HTTP filter
	HealthCheck = "envoy.filters.http.health_check"
	// Echo network filter
	Echo = "envoy.filters.network.echo"
	// HTTPConnectionManager network filter
	HTTPConnectionManager = "envoy.filters.network.http_connection_manager"
	// ServiceControl filter.
	ServiceControl = "envoy.filters.http.service_control"
	// JwtAuthn filter.
	JwtAuthn = "envoy.filters.http.jwt_authn"
	// PathMatcher filter.
	PathMatcher = "envoy.filters.http.path_matcher"
	// BackendAuth filter.
	BackendAuth = "envoy.filters.http.backend_auth"
	// BackendRouting filter.
	BackendRouting = "envoy.filters.http.backend_routing"
	// GrpcStats filter name
	GrpcStatsFilterName = "envoy.filters.http.grpc_stats"
	// TLSTransportSocket is Envoy TLS Transport Socket name.
	TLSTransportSocket = "envoy.transport_sockets.tls"
	// DefaultRootCAPaths is the default certs path.
	DefaultRootCAPaths = "/etc/ssl/certs/ca-certificates.crt"

	// JwtPayloadMetadataName is the field name passed into metadata
	JwtPayloadMetadataName = "jwt_payloads"

	// Supported Http Methods.

	GET     = "GET"
	PUT     = "PUT"
	POST    = "POST"
	DELETE  = "DELETE"
	PATCH   = "PATCH"
	OPTIONS = "OPTIONS"
	CUSTOM  = "CUSTOM"

	// Rollout strategy

	FixedRolloutStrategy   = "fixed"
	ManagedRolloutStrategy = "managed"

	// Metadata suffix

	ConfigIDSuffix          = "/v1/instance/attributes/endpoints-service-version"
	GAEServerSoftwareSuffix = "/v1/instance/attributes/gae_server_software"
	KubeEnvSuffix           = "/v1/instance/attributes/kube-env"
	RolloutStrategySuffix   = "/v1/instance/attributes/endpoints-rollout-strategy"
	ServiceNameSuffix       = "/v1/instance/attributes/endpoints-service-name"

	AccessTokenSuffix   = "/v1/instance/service-accounts/default/token"
	IdentityTokenSuffix = "/v1/instance/service-accounts/default/identity"
	ProjectIDSuffix     = "/v1/project/project-id"
	ZoneSuffix          = "/v1/instance/zone"

	// b/147591854: This string must NOT have a trailing slash
	OpenIDDiscoveryCfgURLSuffix = "/.well-known/openid-configuration"

	// The metadata server cluster name.
	MetadataServerClusterName = "metadata-cluster"

	// The iam server cluster name.
	IamServerClusterName = "iam-cluster"

	// The service control server cluster name.
	ServiceControlClusterName = "service-control-cluster"

	// Platforms

	GAEFlex = "GAE_FLEX(ESPv2)"
	GKE     = "GKE(ESPv2)"
	GCE     = "GCE(ESPv2)"

	// System Parameter Name
	ApiKeyParameterName = "api_key"

	// Default response deadline used if user does not specify one in the BackendRule.
	DefaultResponseDeadline = 15 * time.Second

	// A limit configured to reduce resource usage in Envoy's SafeRegex GoogleRE2 matcher.
	// b/148606900: It is safe to set this to a fairly high value.
	// This won't impact resource usage for customers who have short UriTemplates.
	GoogleRE2MaxProgramSize = 1000

	// Default jwt locations
	DefaultJwtHeaderNameAuthorization          = "Authorization"
	DefaultJwtHeaderValuePrefixBearer          = "Bearer "
	DefaultJwtHeaderNameXGoogleIapJwtAssertion = "X-Goog-Iap-Jwt-Assertion"
	DefaultJwtQueryParamAccessToken            = "access_token"

	// Default api key locations
	DefaultApiKeyQueryParamKey    = "key"
	DefaultApiKeyQueryParamApiKey = "api_key"

	// Strict Transport Security header key and value
	HSTSHeaderKey   = "Strict-Transport-Security"
	HSTSHeaderValue = "max-age=31536000; includeSubdomains"
)

type BackendProtocol int32

type GetAccessTokenFunc func() (string, time.Duration, error)
type GetNewRolloutIdFunc func() (string, error)

// Backend protocol.
const (
	UNKNOWN BackendProtocol = iota
	HTTP1
	HTTP2
	GRPC
)
