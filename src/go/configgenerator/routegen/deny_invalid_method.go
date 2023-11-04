package routegen

import (
	"fmt"
	"net/http"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/routegen/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// DenyInvalidMethodGenerator is a RouteGenerator that creates "method not
// allowed" matchers for the RouteGenerators that it wraps.
//
// We allow this configurability because some products (Istio/ASM) do not
// require generation of these deny routes.
type DenyInvalidMethodGenerator struct {
	WrappedGens []RouteGenerator

	DisallowColonInWildcardPathSegment bool

	*NoopRouteGenerator
}

// NewDenyInvalidMethodRouteGenFromOPConfig creates DenyInvalidMethodGenerator
// from OP service config + ESPv2 options.
// It is a RouteGeneratorOPFactory.
func NewDenyInvalidMethodRouteGenFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions, wrappedGenFactories []RouteGeneratorOPFactory) (RouteGenerator, error) {
	wrappedGens, err := NewRouteGeneratorsFromOPConfig(serviceConfig, opts, wrappedGenFactories)
	if err != nil {
		return nil, fmt.Errorf("while creating method not allowed routes, could not create underlying wrapped routes, failed with error: %v", err)
	}

	return &DenyInvalidMethodGenerator{
		WrappedGens:                        wrappedGens,
		DisallowColonInWildcardPathSegment: opts.DisallowColonInWildcardPathSegment,
	}, nil
}

// RouteType implements interface RouteGenerator.
func (g *DenyInvalidMethodGenerator) RouteType() string {
	return "deny_invalid_method_routes"
}

// GenRouteConfig implements interface RouteGenerator.
func (g *DenyInvalidMethodGenerator) GenRouteConfig([]filtergen.FilterGenerator) ([]*routepb.Route, error) {
	var httpPatterns httppattern.MethodSlice
	for _, gen := range g.WrappedGens {
		httpPatterns = append(httpPatterns, gen.AffectedHTTPPatterns()...)
	}

	if err := httppattern.Sort(&httpPatterns); err != nil {
		return nil, err
	}

	var methodNotAllowedRoutes []*routepb.Route
	seenUriTemplatesInRoute := make(map[string]bool)
	for _, httpPattern := range httpPatterns {
		routeMatchers, err := helpers.MakeRouteMatchers(httpPattern.Pattern, g.DisallowColonInWildcardPathSegment)
		if err != nil {
			return nil, fmt.Errorf("fail to make method not allowed route matchers for operation %q with http pattern %q: %v", httpPattern.Operation, httpPattern.Pattern.String(), err)
		}

		for _, routeMatch := range routeMatchers {
			routeMatcher := routeMatch.RouteMatch
			if httpPattern.HttpMethod != httppattern.HttpMethodWildCard {
				uriTemplate := routeMatch.UriTemplate
				if ok, _ := seenUriTemplatesInRoute[uriTemplate]; !ok {
					seenUriTemplatesInRoute[uriTemplate] = true
					methodNotAllowedRoutes = append(methodNotAllowedRoutes, makeMethodNotAllowedRoute(routeMatcher, httpPattern.UriTemplate.Origin))
				}
			}
		}
	}

	return methodNotAllowedRoutes, nil
}

func makeMethodNotAllowedRoute(methodNotAllowedRouteMatcher *routepb.RouteMatch, uriTemplateInSc string) *routepb.Route {
	spanName := util.MaybeTruncateSpanName(fmt.Sprintf("%s UnknownHttpMethodForPath_%s", util.SpanNamePrefix, uriTemplateInSc))

	return &routepb.Route{
		Match: methodNotAllowedRouteMatcher,
		Action: &routepb.Route_DirectResponse{
			DirectResponse: &routepb.DirectResponseAction{
				Status: http.StatusMethodNotAllowed,
				Body: &corepb.DataSource{
					Specifier: &corepb.DataSource_InlineString{
						InlineString: fmt.Sprintf("The current request is matched to the defined url template \"%s\" but its http method is not allowed", uriTemplateInSc),
					},
				},
			},
		},
		Decorator: &routepb.Decorator{
			Operation: spanName,
		},
	}
}
