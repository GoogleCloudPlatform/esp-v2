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

import (
	"fmt"
	"io"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"

	bapb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/backend_auth"
	drpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/backend_routing"
	pmpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/path_matcher"
	scpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/service_control"
	authpb "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	filepb "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v2"
	gspb "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/grpc_stats/v2alpha"
	jwtpb "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/jwt_authn/v2alpha"
	routerpb "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/router/v2"
	transcoderpb "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/transcoder/v2"
	hcmpbv2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tlspb "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"

	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	servicecontrolpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

// Helper to convert Json string to protobuf.Any.
type FuncResolver func(url string) (proto.Message, error)

func (fn FuncResolver) Resolve(url string) (proto.Message, error) {
	return fn(url)
}

var Resolver = FuncResolver(func(url string) (proto.Message, error) {
	switch url {
	case "type.googleapis.com/google.api.servicemanagement.v1.ConfigFile":
		return new(smpb.ConfigFile), nil
	case "type.googleapis.com/google.api.HttpRule":
		return new(annotationspb.HttpRule), nil
	case "type.googleapis.com/google.protobuf.BoolValue":
		return new(wrapperspb.BoolValue), nil
	case "type.googleapis.com/google.protobuf.StringValue":
		return new(wrapperspb.StringValue), nil
	case "type.googleapis.com/google.protobuf.BytesValue":
		return new(wrapperspb.BytesValue), nil
	case "type.googleapis.com/google.protobuf.DoubleValue":
		return new(wrapperspb.DoubleValue), nil
	case "type.googleapis.com/google.protobuf.FloatValue":
		return new(wrapperspb.FloatValue), nil
	case "type.googleapis.com/google.protobuf.Int64Value":
		return new(wrapperspb.Int64Value), nil
	case "type.googleapis.com/google.protobuf.UInt64Value":
		return new(wrapperspb.UInt64Value), nil
	case "type.googleapis.com/google.protobuf.Int32Value":
		return new(wrapperspb.Int32Value), nil
	case "type.googleapis.com/google.protobuf.UInt32Value":
		return new(wrapperspb.UInt32Value), nil
	case "type.googleapis.com/google.api.Service":
		return new(confpb.Service), nil
	case "type.googleapis.com/envoy.config.filter.http.grpc_stats.v2alpha.FilterConfig":
		return new(gspb.FilterConfig), nil
	case "type.googleapis.com/envoy.config.filter.http.transcoder.v2.GrpcJsonTranscoder":
		return new(transcoderpb.GrpcJsonTranscoder), nil
	case "type.googleapis.com/envoy.config.filter.http.jwt_authn.v2alpha.JwtAuthentication":
		return new(jwtpb.JwtAuthentication), nil
	case "type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager":
		return new(hcmpbv2.HttpConnectionManager), nil
	case "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager":
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
	case "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext":
		return new(tlspb.UpstreamTlsContext), nil
	case "type.googleapis.com/envoy.api.v2.auth.UpstreamTlsContext":
		return new(authpb.UpstreamTlsContext), nil
	case "type.googleapis.com/envoy.api.v2.auth.DownstreamTlsContext":
		return new(authpb.DownstreamTlsContext), nil
	case "type.googleapis.com/envoy.config.accesslog.v2.FileAccessLog":
		return new(filepb.FileAccessLog), nil
	default:
		return nil, fmt.Errorf("unexpected protobuf.Any with url: %s", url)
	}
})

// UnmarshalBytesToPbMessage converts bytes to corresponding pb message.
var UnmarshalBytesToPbMessage = func(input []byte, output proto.Message) error {
	switch t := output.(type) {
	case *confpb.Service:
		if err := proto.Unmarshal(input, output.(*confpb.Service)); err != nil {
			return fmt.Errorf("fail to unmarshal %T: %v", t, err)
		}
	case *smpb.ListServiceRolloutsResponse:
		if err := proto.Unmarshal(input, output.(*smpb.ListServiceRolloutsResponse)); err != nil {
			return fmt.Errorf("fail to unmarshal %T: %v", t, err)
		}
	case *servicecontrolpb.ReportResponse:
		if err := proto.Unmarshal(input, output.(*servicecontrolpb.ReportResponse)); err != nil {
			return fmt.Errorf("fail to unmarshal %T: %v", t, err)
		}
		return nil
	default:
		return fmt.Errorf("not support unmarshalling %T", t)
	}
	return nil
}

// UnmarshalServiceConfig converts service config in JSON to proto
func UnmarshalServiceConfig(config io.Reader) (*confpb.Service, error) {
	unmarshaler := &jsonpb.Unmarshaler{
		AllowUnknownFields: true,
		AnyResolver:        Resolver,
	}
	var serviceConfig confpb.Service
	if err := unmarshaler.Unmarshal(config, &serviceConfig); err != nil {
		return nil, fmt.Errorf("fail to unmarshal serviceConfig: %s", err)
	}
	return &serviceConfig, nil
}

func ProtoToJson(msg proto.Message) (string, error) {
	marshaler := &jsonpb.Marshaler{}
	return marshaler.MarshalToString(msg)
}
