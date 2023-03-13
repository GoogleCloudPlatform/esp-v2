package filtergen

import (
	"fmt"

	ci "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	routerpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/golang/protobuf/ptypes"
	anypb "github.com/golang/protobuf/ptypes/any"
)

type RouterGenerator struct{}

func (g *RouterGenerator) FilterName() string {
	return util.Router
}

func (g *RouterGenerator) GenFilterConfig(serviceInfo *ci.ServiceInfo) (*hcmpb.HttpFilter, []*ci.MethodInfo, error) {
	router, _ := ptypes.MarshalAny(&routerpb.Router{
		SuppressEnvoyHeaders: serviceInfo.Options.SuppressEnvoyHeaders,
		StartChildSpan:       !serviceInfo.Options.DisableTracing,
	})

	routerFilter := &hcmpb.HttpFilter{
		Name:       util.Router,
		ConfigType: &hcmpb.HttpFilter_TypedConfig{TypedConfig: router},
	}
	return routerFilter, nil, nil
}

func (g *RouterGenerator) GenPerRouteConfig(method *ci.MethodInfo, httpRule *httppattern.Pattern) (*anypb.Any, error) {
	return nil, fmt.Errorf("UNIMPLEMENTED")
}
