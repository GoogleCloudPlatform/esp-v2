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
	// Service Control filter.
	ServiceControl = "envoy.filters.http.service_control"
	// JWT Authn filter.
	JwtAuthn = "envoy.filters.http.jwt_authn"
	// Path Matcher filter.
	PathMatcher = "envoy.filters.http.path_matcher"
	// Backend Auth filter.
	BackendAuth = "envoy.filters.http.backend_auth"

	// Supported Http Methods.
	GET     = "GET"
	PUT     = "PUT"
	POST    = "POST"
	DELETE  = "DELETE"
	PATCH   = "PATCH"
	OPTIONS = "OPTIONS"
	CUSTOM  = "CUSTOM"

	// Clusters
	TokenCluster = "ads_cluster"

	// Rollout strategy
	FixedRolloutStrategy   = "fixed"
	ManagedRolloutStrategy = "managed"

	// Metadata suffix
	ConfigIDSuffix          = "/v1/instance/attributes/endpoints-service-version"
	GAEServerSoftwareSuffix = "/v1/instance/attributes/gae_server_software"
	KubeEnvSuffix           = "/v1/instance/attributes/kube-env"
	RolloutStrategySuffix   = "/v1/instance/attributes/endpoints-rollout-strategy"
	ServiceNameSuffix       = "/v1/instance/attributes/endpoints-service-name"

	ServiceAccountTokenSuffix = "/v1/instance/service-accounts/default/token"
	ProjectIDSuffix           = "/v1/instance/project/project-id"
	ZoneSuffix                = "/v1/instance/zone"

	// Platforms
	GAEFlex = "GAE_FLEX"
	GKE     = "GKE"
	GCE     = "GCE"
)

type BackendProtocol int32

// Backend protocol.
const (
	Unknown BackendProtocol = iota
	HTTP1
	HTTP2
	GRPC
)
