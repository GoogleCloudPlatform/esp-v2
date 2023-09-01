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

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	servicecontrolpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	// Import all protos that should be linked into the binary here.
	_ "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/backend_auth"
	_ "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/grpc_metadata_scrubber"
	_ "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/header_sanitizer"
	_ "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/path_rewrite"
	_ "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/service_control"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/metrics/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/trace/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/grpc/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/compression/brotli/compressor/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/compression/gzip/compressor/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/compressor/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/cors/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_json_transcoder/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_stats/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_web/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	_ "google.golang.org/genproto/googleapis/api/serviceconfig"
	_ "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
	_ "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
	_ "google.golang.org/genproto/googleapis/api/visibility"
	_ "google.golang.org/protobuf/types/known/wrapperspb"
)

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

// UnmarshalServiceConfig converts service config in JSON to proto.
// Allows unknown fields.
func UnmarshalServiceConfig(config []byte) (*confpb.Service, error) {
	var serviceConfig confpb.Service
	unmarshaller := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaller.Unmarshal(config, &serviceConfig); err != nil {
		return nil, fmt.Errorf("fail to unmarshal serviceConfig: %s", err)
	}
	return &serviceConfig, nil
}

func ProtoToJson(msg proto.Message) (string, error) {
	b, err := protojson.Marshal(msg)
	return string(b), err
}
