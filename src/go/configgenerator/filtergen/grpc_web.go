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
	ci "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	grpcwebpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_web/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/protobuf/types/known/anypb"
)

type GRPCWebGenerator struct {
	// skipFilter indicates if this filter is disabled based on options and config.
	skipFilter bool
}

// NewGRPCWebGenerator creates the GRPCWebGenerator with cached config.
func NewGRPCWebGenerator(serviceInfo *ci.ServiceInfo) *GRPCWebGenerator {
	return &GRPCWebGenerator{
		skipFilter: !serviceInfo.GrpcSupportRequired,
	}
}

func (g *GRPCWebGenerator) FilterName() string {
	return util.GRPCWeb
}

func (g *GRPCWebGenerator) IsEnabled() bool {
	return !g.skipFilter
}

func (g *GRPCWebGenerator) GenFilterConfig(serviceInfo *ci.ServiceInfo) (*hcmpb.HttpFilter, error) {
	a, err := ptypes.MarshalAny(&grpcwebpb.GrpcWeb{})
	if err != nil {
		return nil, err
	}
	return &hcmpb.HttpFilter{
		Name:       util.GRPCWeb,
		ConfigType: &hcmpb.HttpFilter_TypedConfig{TypedConfig: a},
	}, nil
}

func (g *GRPCWebGenerator) GenPerRouteConfig(method *ci.MethodInfo, httpRule *httppattern.Pattern) (*anypb.Any, error) {
	return nil, nil
}
