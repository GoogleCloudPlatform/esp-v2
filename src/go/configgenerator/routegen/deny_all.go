package routegen

import (
	"fmt"
	"net/http"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// DenyAllGenerator is a RouteGenerator that denies all requests.
type DenyAllGenerator struct {
	*NoopRouteGenerator
}

// NewDenyAllRouteGenFromOPConfig creates DenyAllGenerator
// from OP service config + ESPv2 options.
// It is a RouteGeneratorOPFactory.
func NewDenyAllRouteGenFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) (RouteGenerator, error) {
	return &DenyAllGenerator{}, nil
}

// RouteType implements interface RouteGenerator.
func (g *DenyAllGenerator) RouteType() string {
	return "deny_all"
}

// GenRouteConfig implements interface RouteGenerator.
func (g *DenyAllGenerator) GenRouteConfig([]filtergen.FilterGenerator) ([]*routepb.Route, error) {
	return []*routepb.Route{
		{
			Match: &routepb.RouteMatch{
				PathSpecifier: &routepb.RouteMatch_Prefix{
					Prefix: "/",
				},
			},
			Action: &routepb.Route_DirectResponse{
				DirectResponse: &routepb.DirectResponseAction{
					Status: http.StatusNotFound,
					Body: &corepb.DataSource{
						Specifier: &corepb.DataSource_InlineString{
							InlineString: `The current request is not defined by this API.`,
						},
					},
				},
			},
			Decorator: &routepb.Decorator{
				Operation: fmt.Sprintf("%s UnknownOperationName", util.SpanNamePrefix),
			},
		},
	}, nil
}
