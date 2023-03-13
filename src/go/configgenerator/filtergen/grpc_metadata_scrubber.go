package filtergen

import (
	"fmt"

	ci "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	gmspb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v11/http/grpc_metadata_scrubber"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/golang/protobuf/ptypes"
	anypb "github.com/golang/protobuf/ptypes/any"
)

type GRPCMetadataScrubberGenerator struct{}

func (g *GRPCMetadataScrubberGenerator) FilterName() string {
	return util.GrpcMetadataScrubber
}

func (g *GRPCMetadataScrubberGenerator) GenFilterConfig(serviceInfo *ci.ServiceInfo) (*hcmpb.HttpFilter, []*ci.MethodInfo, error) {
	a, err := ptypes.MarshalAny(&gmspb.FilterConfig{})
	if err != nil {
		return nil, nil, err
	}
	return &hcmpb.HttpFilter{
		Name:       util.GrpcMetadataScrubber,
		ConfigType: &hcmpb.HttpFilter_TypedConfig{TypedConfig: a},
	}, nil, nil
}

func (g *GRPCMetadataScrubberGenerator) GenPerRouteConfig(method *ci.MethodInfo, httpRule *httppattern.Pattern) (*anypb.Any, error) {
	return nil, fmt.Errorf("UNIMPLEMENTED")
}
