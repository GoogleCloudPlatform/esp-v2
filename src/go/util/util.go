// Copyright 2018 Google Cloud Platform Proxy Authors
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

const (
	// HTTP filter names.

	// Buffer HTTP filter
	Buffer = "envoy.buffer"
	// CORS HTTP filter
	CORS = "envoy.cors"
	// GRPCJSONTranscoder HTTP filter
	GRPCJSONTranscoder = "envoy.grpc_json_transcoder"
	// GRPCWeb HTTP filter
	GRPCWeb = "envoy.grpc_web"
	// Router HTTP filter
	Router = "envoy.router"
	// Health checking HTTP filter
	HealthCheck = "envoy.health_check"
	// Echo network filter
	Echo = "envoy.echo"
	// HTTPConnectionManager network filter
	HTTPConnectionManager = "envoy.http_connection_manager"
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

	// JwtPayloadMetadataName is the field name passed into metadata
	JwtPayloadMetadataName = "jwt_payloads"
	// FakeJwksUri used when jwksUri is unavailable
	FakeJwksUri = "http://aaaaaaaaaaaaa.bbbbbbbbbbbbb.cccccccccccc/inaccessible_pkey"

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

	ServiceAccountTokenSuffix   = "/v1/instance/service-accounts/default/token"
	IdentityTokenSuffix         = "/v1/instance/service-accounts/default/identity"
	ProjectIDSuffix             = "/v1/project/project-id"
	ZoneSuffix                  = "/v1/instance/zone"
	OpenIDDiscoveryCfgURLSuffix = "/.well-known/openid-configuration/"

	// The metadata server cluster name.
	MetadataServerClusterName = "metadata-cluster"

	// The service control server cluster name.
	ServiceControlClusterName = "service-control-cluster"

	// Platforms

	GAEFlex = "GAE_FLEX"
	GKE     = "GKE"
	GCE     = "GCE"

	// System Parameter Name
	APIKeyParameterName = "api_key"
)

type BackendProtocol int32

// Backend protocol.
const (
	Unknown BackendProtocol = iota
	HTTP1
	HTTP2
	GRPC
)
