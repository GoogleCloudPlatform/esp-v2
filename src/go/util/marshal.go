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

	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
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
	default:
		return nil, fmt.Errorf("unexpected protobuf.Any with url: %s", url)
	}
})

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
