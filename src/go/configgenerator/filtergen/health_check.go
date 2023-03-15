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
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/health_check/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	envoytypepb "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/golang/protobuf/ptypes"
	anypb "github.com/golang/protobuf/ptypes/any"
	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"
)

type HealthCheckGenerator struct {
	// skipFilter indicates if this filter is disabled based on options and config.
	skipFilter bool
}

// NewHealthCheckGenerator creates the HealthCheckGenerator with cached config.
func NewHealthCheckGenerator(serviceInfo *ci.ServiceInfo) *HealthCheckGenerator {
	return &HealthCheckGenerator{
		skipFilter: serviceInfo.Options.Healthz == "",
	}
}

func (g *HealthCheckGenerator) FilterName() string {
	return util.HealthCheck
}

func (g *HealthCheckGenerator) IsEnabled() bool {
	return !g.skipFilter
}

func (g *HealthCheckGenerator) GenFilterConfig(serviceInfo *ci.ServiceInfo) (*hcmpb.HttpFilter, error) {
	hcFilterConfig := &hcpb.HealthCheck{
		PassThroughMode: &wrapperspb.BoolValue{Value: false},

		Headers: []*routepb.HeaderMatcher{
			{
				Name: ":path",
				HeaderMatchSpecifier: &routepb.HeaderMatcher_StringMatch{
					StringMatch: &matcher.StringMatcher{
						MatchPattern: &matcher.StringMatcher_Exact{
							Exact: serviceInfo.Options.Healthz,
						},
					},
				},
			},
		},
	}

	if serviceInfo.Options.HealthCheckGrpcBackend {
		hcFilterConfig.ClusterMinHealthyPercentages = map[string]*envoytypepb.Percent{
			serviceInfo.LocalBackendCluster.ClusterName: &envoytypepb.Percent{Value: 100.0},
		}
	}

	hcFilterConfigStruc, err := ptypes.MarshalAny(hcFilterConfig)
	if err != nil {
		return nil, err
	}
	return &hcmpb.HttpFilter{
		Name: util.HealthCheck,
		ConfigType: &hcmpb.HttpFilter_TypedConfig{
			TypedConfig: hcFilterConfigStruc,
		},
	}, nil
}

func (g *HealthCheckGenerator) GenPerRouteConfig(method *ci.MethodInfo, httpRule *httppattern.Pattern) (*anypb.Any, error) {
	return nil, nil
}
