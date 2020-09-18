// Copyright 2020 Google LLC
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

import "fmt"

const (
	// Upstream envoy http filter names.

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
	// JwtAuthn filter.
	JwtAuthn = "envoy.filters.http.jwt_authn"
	// TLSTransportSocket is Envoy TLS Transport Socket name.
	TLSTransportSocket = "envoy.transport_sockets.tls"
	// AccessFileLogger filter name
	AccessFileLogger = "envoy.access_loggers.file"

	// ESPv2 custom http filters.

	// ServiceControl filter.
	ServiceControl = "com.google.espv2.filters.http.service_control"
	// PathMatcher filter.
	PathMatcher = "com.google.espv2.filters.http.path_matcher"
	// BackendAuth filter.
	BackendAuth = "com.google.espv2.filters.http.backend_auth"
	// BackendRouting filter.
	BackendRouting = "com.google.espv2.filters.http.backend_routing"
	// gRPC Metadata Scrubber filter.
	GrpcMetadataScrubber = "com.google.espv2.filters.http.grpc_metadata_scrubber"

	// The metadata server cluster name.
	MetadataServerClusterName = "metadata-cluster"

	// The token agent server cluster name.
	TokenAgentClusterName = "token-agent-cluster"

	// The iam server cluster name.
	IamServerClusterName = "iam-cluster"

	// The service control server cluster name.
	ServiceControlClusterName = "service-control-cluster"

	IngressListenerName  = "ingress_listener"
	LoopbackListenerName = "loopback_listener"
)

// Jwt provider cluster's name will be in form of "jwt-provider-cluster-${JWT_PROVIDER_ADDRESS}".
func JwtProviderClusterName(address string) string {
	return fmt.Sprintf("jwt-provider-cluster-%s", address)
}

// Backend cluster'name will be in form of "backend-cluster-${BACKEND_ADDRESS}"
func BackendClusterName(address string) string {
	return fmt.Sprintf("backend-cluster-%s", address)
}
