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

// HTTP filter names.
const (
	// Service Control filter.
	ServiceControl = "envoy.filters.http.service_control"
	// JWT Authn filter.
	JwtAuthn = "envoy.filters.http.jwt_authn"
	// APIKey field name in HTTP header map.
	APIKeyHeader = "x-api-key"
	// APIKey param name for HTTP request.
	APIKeyQuery = "key"

	// Supported Http Methods.
	GET    = "GET"
	PUT    = "PUT"
	POST   = "POST"
	DELETE = "DELETE"
	PATCH  = "PATCH"
	CUSTOM = "CUSTOM"
)

type BackendProtocol int32

// Backend protocol.
const (
	Unknown BackendProtocol = iota
	HTTP1
	HTTP2
	GRPC
)
