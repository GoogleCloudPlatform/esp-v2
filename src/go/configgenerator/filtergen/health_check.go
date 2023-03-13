package filtergen

import (
	"fmt"

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

type HealthCheckGenerator struct{}

func (g *HealthCheckGenerator) FilterName() string {
	return util.HealthCheck
}

func (g *HealthCheckGenerator) GenFilterConfig(serviceInfo *ci.ServiceInfo) (*hcmpb.HttpFilter, []*ci.MethodInfo, error) {
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
		return nil, nil, err
	}
	return &hcmpb.HttpFilter{
		Name: util.HealthCheck,
		ConfigType: &hcmpb.HttpFilter_TypedConfig{
			TypedConfig: hcFilterConfigStruc,
		},
	}, nil, nil
}

func (g *HealthCheckGenerator) GenPerRouteConfig(method *ci.MethodInfo, httpRule *httppattern.Pattern) (*anypb.Any, error) {
	return nil, fmt.Errorf("UNIMPLEMENTED")
}
