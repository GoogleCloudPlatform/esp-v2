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
	// Echo network filter
	Echo = "envoy.filters.network.echo"
	// TLSTransportSocket is Envoy TLS Transport Socket name.
	TLSTransportSocket = "envoy.transport_sockets.tls"
	// AccessFileLogger filter name
	AccessFileLogger = "envoy.access_loggers.file"
	// UpstreamProtocolOptions is the xDS extension name for HTTP options.
	UpstreamProtocolOptions = "envoy.extensions.upstreams.http.v3.HttpProtocolOptions"

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
