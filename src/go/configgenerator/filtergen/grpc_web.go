package filtergen

import (
	"fmt"

	ci "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	grpcwebpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_web/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/protobuf/types/known/anypb"
)

type GRPCWebGenerator struct{}

func (g *GRPCWebGenerator) FilterName() string {
	return util.GRPCWeb
}

func (g *GRPCWebGenerator) GenFilterConfig(serviceInfo *ci.ServiceInfo) (*hcmpb.HttpFilter, []*ci.MethodInfo, error) {
	a, err := ptypes.MarshalAny(&grpcwebpb.GrpcWeb{})
	if err != nil {
		return nil, nil, err
	}
	return &hcmpb.HttpFilter{
		Name:       util.GRPCWeb,
		ConfigType: &hcmpb.HttpFilter_TypedConfig{TypedConfig: a},
	}, nil, nil
}

func (g *GRPCWebGenerator) GenPerRouteConfig(method *ci.MethodInfo, httpRule *httppattern.Pattern) (*anypb.Any, error) {
	return nil, fmt.Errorf("UNIMPLEMENTED")
}
