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

// Package flags includes all API producer configurable settings.

package flags

import (
	"flag"
	"time"
)

var (
	// Service Management related configurations. Must be set.
	ServiceName     = flag.String("service", "", "endpoint service name")
	ConfigID        = flag.String("version", "", "initial service config id")
	RolloutStrategy = flag.String("rollout_strategy", "fixed", `service config rollout strategy, must be either "managed" or "fixed"`)
	BackendProtocol = flag.String("backend_protocol", "", `must set as one of "grpc", "http1", "http2"`)
	CheckMetadata   = flag.Bool("check_metadata", false, `enable fetching service name, config ID and rollout strategy from service metadata server`)

	// Cors related configurations.
	CorsPreset           = flag.String("cors_preset", "", `enable CORS support, must be either "basic" or "cors_with_regex"`)
	CorsAllowOrigin      = flag.String("cors_allow_origin", "", "set Access-Control-Allow-Origin to a specific origin")
	CorsAllowOriginRegex = flag.String("cors_allow_origin_regex", "", "set Access-Control-Allow-Origin to a regular expression")

	CorsAllowMethods     = flag.String("cors_allow_methods", "", "set Access-Control-Allow-Methods to the specified HTTP methods")
	CorsAllowHeaders     = flag.String("cors_allow_headers", "", "set Access-Control-Allow-Headers to the specified HTTP headers")
	CorsExposeHeaders    = flag.String("cors_expose_headers", "", "set Access-Control-Expose-Headers to the specified headers")
	CorsAllowCredentials = flag.Bool("cors_allow_credentials", false, "whether include the Access-Control-Allow-Credentials header with the value true in responses or not")

	// Backend routing configurations.
	EnableBackendRouting = flag.Bool("enable_backend_routing", false, `enable apiproxy to route requests according to the "x-google-backend" or "backend" configuration`)

	// Envoy specific configurations.
	ClusterConnectTimeout = flag.Duration("cluster_connect_imeout", 20*time.Second, "cluster connect timeout in seconds")

	// Network related configurations.
	Node                 = flag.String("node", "api_proxy", "envoy node id")
	ListenerAddress      = flag.String("listener_address", "0.0.0.0", "listener socket ip address")
	ClusterAddress       = flag.String("cluster_address", "127.0.0.1", "cluster socket ip address")
	ServiceManagementURL = flag.String("service_management_url", "https://servicemanagement.googleapis.com", "url of service management server")
	MetadataURL          = flag.String("metadata_url", "http://metadata.google.internal/computeMetadata", "url of metadata server")

	DiscoveryPort = flag.Int("discovery_port", 8790, "discovery service port")
	ListenerPort  = flag.Int("listener_port", 8080, "listener port")
	ClusterPort   = flag.Int("cluster_port", 8082, "cluster port")

	// Flags for testing purpose.
	SkipServiceControlFilter = flag.Bool("skip_service_control_filter", false, "skip service control filter, for test purpose")
	SkipJwtAuthnFilter       = flag.Bool("skip_jwt_authn_filter", false, "skip jwt authn filter, for test purpose")

	// Envoy configurations.
	EnvoyUseRemoteAddress  = flag.Bool("envoy_use_remote_address", false, "Envoy HttpConnectionManager configuration, please refer to envoy documentation for detailed information.")
	EnvoyXffNumTrustedHops = flag.Int("envoy_xff_num_trusted_hops", 2, "Envoy HttpConnectionManager configuration, please refer to envoy documentation for detailed information.")

	LogRequestHeaders = flag.String("log_request_headers", "", `Log corresponding request headers through service control, separated by comma. Example, when --log_request_headers=
	foo,bar, endpoint log will have request_headers: foo=foo_value;bar=bar_value if values are available;`)
	LogResponseHeaders = flag.String("log_response_headers", "", `Log corresponding response headers through service control, separated by comma. Example, when --log_response_headers=
	foo,bar,endpoint log will have response_headers: foo=foo_value;bar=bar_value if values are available.`)
)
