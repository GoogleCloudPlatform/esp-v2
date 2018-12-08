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
	ServiceName     = flag.String("service_name", "", "endpoint service name")
	ConfigID        = flag.String("config_id", "", "initial service config id")
	BackendProtocol = flag.String("backend_protocol", "", `must set as one of "grpc", "http1", "http2"`)

	// Envoy specific configurations.
	ClusterConnectTimeout = flag.Duration("cluster_connect_imeout", 20*time.Second, "cluster connect timeout in seconds")

	// Network related configurations.
	Node                 = flag.String("node", "api_proxy", "envoy node id")
	ListenerAddress      = flag.String("listener_address", "0.0.0.0", "listener socket ip address")
	ClusterAddress       = flag.String("cluster_address", "127.0.0.1", "cluster socket ip address")
	ServiceManagementURL = flag.String("service_management_url", "https://servicemanagement.googleapis.com", "url of service management server")
	MetadataUrl          = flag.String("metadata_url", "http://metadata.google.internal/computeMetadata", "url of metadata server")

	DiscoveryPort = flag.Int("discovery_port", 8790, "discovery service port")
	ListenerPort  = flag.Int("listener_port", 8080, "listener port")
	ClusterPort   = flag.Int("cluster_port", 8082, "cluster port")

	// Flags for testing purpose.
	SkipServiceControlFilter = flag.Bool("skip_service_control_filter", false, "skip service control filter, for test purpose")
	SkipJwtAuthnFilter       = flag.Bool("skip_jwt_authn_filter", false, "skip jwt authn filter, for test purpose")
)
