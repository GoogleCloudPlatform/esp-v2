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

package utils

import (
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/proto"

	bapb "github.com/GoogleCloudPlatform/api-proxy/src/go/proto/api/envoy/http/backend_auth"
	drpb "github.com/GoogleCloudPlatform/api-proxy/src/go/proto/api/envoy/http/backend_routing"
	pmpb "github.com/GoogleCloudPlatform/api-proxy/src/go/proto/api/envoy/http/path_matcher"
	scpb "github.com/GoogleCloudPlatform/api-proxy/src/go/proto/api/envoy/http/service_control"
	authpb "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	jwtauthnpb "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/jwt_authn/v2alpha"
	routerpb "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/router/v2"
	transcoderpb "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/transcoder/v2"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

// JsonEqual compares two JSON strings after normalizing them.
func JsonEqual(x, y string) bool {
	return NormalizeJson(x) == NormalizeJson(y)
}

// NormalizeJson returns normalized JSON string.
func NormalizeJson(input string) string {
	var jsonObject map[string]interface{}
	json.Unmarshal([]byte(input), &jsonObject)
	outputString, _ := json.Marshal(jsonObject)
	return string(outputString)
}

// Helper to convert Json string to protobuf.Any, for test only.
type FuncResolver func(url string) (proto.Message, error)

func (fn FuncResolver) Resolve(url string) (proto.Message, error) {
	return fn(url)
}

var TestBoostrapResolver = FuncResolver(func(url string) (proto.Message, error) {
	switch url {
	case "type.googleapis.com/google.api.Service":
		return new(confpb.Service), nil
	case "type.googleapis.com/google.api.servicemanagement.v1.ConfigFile":
		return new(smpb.ConfigFile), nil
	case "type.googleapis.com/envoy.config.filter.http.jwt_authn.v2alpha.JwtAuthentication":
		return new(jwtauthnpb.JwtAuthentication), nil
	case "type.googleapis.com/envoy.config.filter.http.transcoder.v2.GrpcJsonTranscoder":
		return new(transcoderpb.GrpcJsonTranscoder), nil
	case "type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager":
		return new(hcmpb.HttpConnectionManager), nil
	case "type.googleapis.com/google.api.envoy.http.path_matcher.FilterConfig":
		return new(pmpb.FilterConfig), nil
	case "type.googleapis.com/google.api.envoy.http.service_control.FilterConfig":
		return new(scpb.FilterConfig), nil
	case "type.googleapis.com/google.api.envoy.http.backend_auth.FilterConfig":
		return new(bapb.FilterConfig), nil
	case "type.googleapis.com/google.api.envoy.http.backend_routing.FilterConfig":
		return new(drpb.FilterConfig), nil
	case "type.googleapis.com/envoy.config.filter.http.router.v2.Router":
		return new(routerpb.Router), nil
	case "type.googleapis.com/envoy.api.v2.auth.UpstreamTlsContext":
		return new(authpb.UpstreamTlsContext), nil
	default:
		return nil, fmt.Errorf("unexpected protobuf.Any with url: %s", url)
	}
})
