// Copyright 2023 Google LLC
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

package filtergen

import (
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	grpcwebpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_web/v3"
	"github.com/golang/glog"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/proto"
)

const (
	// GRPCWebFilterName is the Envoy filter name for debug logging.
	GRPCWebFilterName = "envoy.filters.http.grpc_web"
)

type GRPCWebGenerator struct {
	NoopFilterGenerator
}

// NewGRPCWebFilterGensFromOPConfig creates a GRPCWebGenerator from
// OP service config + descriptor + ESPv2 options. It is a FilterGeneratorOPFactory.
func NewGRPCWebFilterGensFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) ([]FilterGenerator, error) {
	isGRPCSupportRequired, err := IsGRPCSupportRequiredForOPConfig(serviceConfig, opts)
	if err != nil {
		return nil, err
	}
	if !isGRPCSupportRequired {
		glog.Infof("gRPC support is NOT required, skip gRPC web filter completely.")
		return nil, nil
	}

	return []FilterGenerator{
		&GRPCWebGenerator{},
	}, nil
}

func (g *GRPCWebGenerator) FilterName() string {
	return GRPCWebFilterName
}

func (g *GRPCWebGenerator) GenFilterConfig() (proto.Message, error) {
	return &grpcwebpb.GrpcWeb{}, nil
}
