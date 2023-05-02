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
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/health_check/v3"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	envoytypepb "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/proto"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	// HealthCheckFilterName is the Envoy filter name for debug logging.
	HealthCheckFilterName = "envoy.filters.http.health_check"
)

type HealthCheckGenerator struct {
	HealthzPath                  string
	ShouldHealthCheckGrpcBackend bool
	LocalBackendClusterName      string
}

// NewHealthCheckFilterGensFromOPConfig creates a HealthCheckGenerator from
// OP service config + descriptor + ESPv2 options. It is a FilterGeneratorOPFactory.
func NewHealthCheckFilterGensFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions, params FactoryParams) ([]FilterGenerator, error) {
	if opts.Healthz == "" {
		return nil, nil
	}

	healthzPath := opts.Healthz
	if !strings.HasPrefix(healthzPath, "/") {
		healthzPath = fmt.Sprintf("/%s", healthzPath)
	}

	return []FilterGenerator{
		&HealthCheckGenerator{
			HealthzPath:                  healthzPath,
			ShouldHealthCheckGrpcBackend: opts.HealthCheckGrpcBackend,
			LocalBackendClusterName:      clustergen.MakeLocalBackendClusterName(serviceConfig),
		},
	}, nil
}

func (g *HealthCheckGenerator) FilterName() string {
	return HealthCheckFilterName
}

func (g *HealthCheckGenerator) GenFilterConfig() (proto.Message, error) {
	hcFilterConfig := &hcpb.HealthCheck{
		PassThroughMode: &wrapperspb.BoolValue{Value: false},

		Headers: []*routepb.HeaderMatcher{
			{
				Name: ":path",
				HeaderMatchSpecifier: &routepb.HeaderMatcher_StringMatch{
					StringMatch: &matcher.StringMatcher{
						MatchPattern: &matcher.StringMatcher_Exact{
							Exact: g.HealthzPath,
						},
					},
				},
			},
		},
	}

	if g.ShouldHealthCheckGrpcBackend {
		hcFilterConfig.ClusterMinHealthyPercentages = map[string]*envoytypepb.Percent{
			g.LocalBackendClusterName: {Value: 100.0},
		}
	}

	return hcFilterConfig, nil
}

func (g *HealthCheckGenerator) GenPerRouteConfig(selector string, httpRule *httppattern.Pattern) (proto.Message, error) {
	return nil, nil
}
