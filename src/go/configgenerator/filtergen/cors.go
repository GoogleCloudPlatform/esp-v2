package filtergen

import (
	"fmt"

	ci "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	corspb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/cors/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/golang/protobuf/ptypes"
	anypb "github.com/golang/protobuf/ptypes/any"
)

type CORSGenerator struct{}

func (g *CORSGenerator) FilterName() string {
	return util.CORS
}

func (g *CORSGenerator) GenFilterConfig(serviceInfo *ci.ServiceInfo) (*hcmpb.HttpFilter, []*ci.MethodInfo, error) {
	a, err := ptypes.MarshalAny(&corspb.Cors{})
	if err != nil {
		return nil, nil, err
	}
	corsFilter := &hcmpb.HttpFilter{
		Name:       util.CORS,
		ConfigType: &hcmpb.HttpFilter_TypedConfig{TypedConfig: a},
	}
	return corsFilter, nil, nil
}

func (g *CORSGenerator) GenPerRouteConfig(method *ci.MethodInfo, httpRule *httppattern.Pattern) (*anypb.Any, error) {
	return nil, fmt.Errorf("UNIMPLEMENTED")
}
