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

	bapb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v11/http/backend_auth"
	gmspb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v11/http/grpc_metadata_scrubber"
	prpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v11/http/path_rewrite"
	scpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v11/http/service_control"

	listenerpb "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	statspb "github.com/envoyproxy/go-control-plane/envoy/config/metrics/v3"
	tracepb "github.com/envoyproxy/go-control-plane/envoy/config/trace/v3"
	accessfilepb "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	accessgrpcpb "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/grpc/v3"
	brpb "github.com/envoyproxy/go-control-plane/envoy/extensions/compression/brotli/compressor/v3"
	gzippb "github.com/envoyproxy/go-control-plane/envoy/extensions/compression/gzip/compressor/v3"
	comppb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/compressor/v3"
	corspb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/cors/v3"
	transcoderpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_json_transcoder/v3"
	gspb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_stats/v3"
	grpcwebpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_web/v3"
	jwtpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	routerpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tlspb "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	httppb "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"

	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	servicecontrolpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
	visibilitypb "google.golang.org/genproto/googleapis/api/visibility"
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
	case "type.googleapis.com/google.api.VisibilityRule":
		return new(visibilitypb.VisibilityRule), nil
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
	case "type.googleapis.com/envoy.extensions.filters.http.compressor.v3.Compressor":
		return new(comppb.Compressor), nil
	case "type.googleapis.com/envoy.extensions.filters.http.cors.v3.Cors":
		return new(corspb.Cors), nil
	case "type.googleapis.com/envoy.extensions.filters.http.grpc_stats.v3.FilterConfig":
		return new(gspb.FilterConfig), nil
	case "type.googleapis.com/envoy.extensions.filters.http.grpc_json_transcoder.v3.GrpcJsonTranscoder":
		return new(transcoderpb.GrpcJsonTranscoder), nil
	case "type.googleapis.com/envoy.extensions.filters.http.grpc_web.v3.GrpcWeb":
		return new(grpcwebpb.GrpcWeb), nil
	case "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication":
		return new(jwtpb.JwtAuthentication), nil
	case "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.PerRouteConfig":
		return new(jwtpb.PerRouteConfig), nil
	case "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager":
		return new(hcmpb.HttpConnectionManager), nil
	case "type.googleapis.com/espv2.api.envoy.v11.http.path_rewrite.FilterConfig":
		return new(prpb.FilterConfig), nil
	case "type.googleapis.com/espv2.api.envoy.v11.http.path_rewrite.PerRouteFilterConfig":
		return new(prpb.PerRouteFilterConfig), nil
	case "type.googleapis.com/espv2.api.envoy.v11.http.service_control.PerRouteFilterConfig":
		return new(scpb.PerRouteFilterConfig), nil
	case "type.googleapis.com/espv2.api.envoy.v11.http.service_control.FilterConfig":
		return new(scpb.FilterConfig), nil
	case "type.googleapis.com/espv2.api.envoy.v11.http.backend_auth.PerRouteFilterConfig":
		return new(bapb.PerRouteFilterConfig), nil
	case "type.googleapis.com/espv2.api.envoy.v11.http.backend_auth.FilterConfig":
		return new(bapb.FilterConfig), nil
	case "type.googleapis.com/espv2.api.envoy.v11.http.grpc_metadata_scrubber.FilterConfig":
		return new(gmspb.FilterConfig), nil
	case "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router":
		return new(routerpb.Router), nil
	case "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext":
		return new(tlspb.UpstreamTlsContext), nil
	case "type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog":
		return new(accessfilepb.FileAccessLog), nil
	case "type.googleapis.com/envoy.extensions.access_loggers.grpc.v3.HttpGrpcAccessLogConfig":
		return new(accessgrpcpb.HttpGrpcAccessLogConfig), nil
	case "type.googleapis.com/envoy.extensions.access_loggers.grpc.v3.TcpGrpcAccessLogConfig":
		return new(accessgrpcpb.TcpGrpcAccessLogConfig), nil
	case "type.googleapis.com/envoy.extensions.access_loggers.grpc.v3.CommonGrpcAccessLogConfig":
		return new(accessgrpcpb.CommonGrpcAccessLogConfig), nil
	case "type.googleapis.com/envoy.extensions.compression.brotli.compressor.v3.Brotli":
		return new(brpb.Brotli), nil
	case "type.googleapis.com/envoy.extensions.compression.gzip.compressor.v3.Gzip":
		return new(gzippb.Gzip), nil
	case "type.googleapis.com/envoy.extensions.upstreams.http.v3.HttpProtocolOptions":
		return new(httppb.HttpProtocolOptions), nil
	case "type.googleapis.com/envoy.config.listener.v3.Listener":
		return new(listenerpb.Listener), nil
	case "type.googleapis.com/envoy.config.metrics.v3.StatsConfig":
		return new(statspb.StatsConfig), nil
	case "type.googleapis.com/envoy.config.metrics.v3.StatsSink":
		return new(statspb.StatsSink), nil
	case "type.googleapis.com/envoy.config.metrics.v3.StatsdSink":
		return new(statspb.StatsdSink), nil
	case "type.googleapis.com/envoy.config.trace.v3.OpenCensusConfig":
		return new(tracepb.OpenCensusConfig), nil
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
