package routegen

import (
	"fmt"
	"net/http"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	matcherpb "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/golang/glog"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// CORSGenerator is a RouteGenerator to configure CORS routes.
type CORSGenerator struct {
	Preset string
	// AllowOrigin should only be set if preset=basic
	AllowOrigin string
	// AllowOriginRegex should only be set if preset=cors_with_regex
	AllowOriginRegex string

	// LocalBackendClusterName is the name of the local backend cluster to apply
	// CORS policies to.
	LocalBackendClusterName string
}

// NewCORSRouteGensFromOPConfig creates CORSGenerator
// from OP service config + descriptor + ESPv2 options.
// It is a RouteGeneratorOPFactory.
func NewCORSRouteGensFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) ([]RouteGenerator, error) {
	if opts.CorsPreset == "" {
		glog.Infof("Not adding CORS route gen because the feature is disabled by option, option is currently %q", opts.CorsPreset)
		return nil, nil
	}

	return []RouteGenerator{
		&CORSGenerator{
			Preset:                  opts.CorsPreset,
			AllowOrigin:             opts.CorsAllowOrigin,
			AllowOriginRegex:        opts.CorsAllowOriginRegex,
			LocalBackendClusterName: clustergen.MakeLocalBackendClusterName(serviceConfig),
		},
	}, nil
}

func (g *CORSGenerator) GenRouteConfig() ([]*routepb.Route, error) {
	originMatcher := &routepb.HeaderMatcher{
		Name: "origin",
	}

	switch g.Preset {
	case "basic":
		if err := fillBasicOriginMatcher(originMatcher, g.AllowOrigin); err != nil {
			return nil, fmt.Errorf("fail to fill basic origin matcher: %v", err)
		}

	case "cors_with_regex":
		if err := fillRegexOriginMatcher(originMatcher, g.AllowOriginRegex); err != nil {
			return nil, fmt.Errorf("fail to fill regex origin matcher: %v", err)
		}

	default:
		return nil, fmt.Errorf(`cors_preset must be either "basic" or "cors_with_regex"`)
	}

	return []*routepb.Route{
		genPreflightCorsRoute(g.LocalBackendClusterName, originMatcher),
		genPreflightCorsMissingHeadersRoute(),
	}, nil
}

func fillBasicOriginMatcher(originMatcher *routepb.HeaderMatcher, allowOrigin string) error {
	if allowOrigin == "" {
		return fmt.Errorf("cors_allow_origin cannot be empty when cors_preset=basic")
	}

	if allowOrigin == "*" {
		originMatcher.HeaderMatchSpecifier = &routepb.HeaderMatcher_PresentMatch{
			PresentMatch: true,
		}
		return nil
	}

	originMatcher.HeaderMatchSpecifier = &routepb.HeaderMatcher_StringMatch{
		StringMatch: &matcherpb.StringMatcher{
			MatchPattern: &matcherpb.StringMatcher_Exact{
				Exact: allowOrigin,
			},
		},
	}
	return nil
}

func fillRegexOriginMatcher(originMatcher *routepb.HeaderMatcher, allowOriginRegex string) error {
	if allowOriginRegex == "" {
		return fmt.Errorf("cors_allow_origin_regex cannot be empty when cors_preset=cors_with_regex")
	}

	if err := util.ValidateRegexProgramSize(allowOriginRegex, util.GoogleRE2MaxProgramSize); err != nil {
		return fmt.Errorf("invalid cors origin regex: %v", err)
	}

	originMatcher.HeaderMatchSpecifier = &routepb.HeaderMatcher_StringMatch{
		StringMatch: &matcherpb.StringMatcher{
			MatchPattern: &matcherpb.StringMatcher_SafeRegex{
				SafeRegex: &matcherpb.RegexMatcher{
					Regex: allowOriginRegex,
				},
			},
		},
	}

	return nil
}

// In order to apply Envoy cors policy, need to have a catch-all route to match
// preflight CORS requests.
func genPreflightCorsRoute(localBackendClusterName string, originMatcher *routepb.HeaderMatcher) *routepb.Route {
	return &routepb.Route{
		Match: &routepb.RouteMatch{
			PathSpecifier: &routepb.RouteMatch_Prefix{
				Prefix: "/",
			},
			Headers: []*routepb.HeaderMatcher{
				{
					Name: ":method",
					HeaderMatchSpecifier: &routepb.HeaderMatcher_StringMatch{
						StringMatch: &matcherpb.StringMatcher{
							MatchPattern: &matcherpb.StringMatcher_Exact{
								Exact: "OPTIONS",
							},
						},
					},
				},
				originMatcher,
				{
					Name: "access-control-request-method",
					HeaderMatchSpecifier: &routepb.HeaderMatcher_PresentMatch{
						PresentMatch: true,
					},
				},
			},
		},
		// Envoy requires to have a Route action in order to create a route
		// for cors Route to work.
		Action: &routepb.Route_Route{
			Route: &routepb.RouteAction{
				ClusterSpecifier: &routepb.RouteAction_Cluster{
					Cluster: localBackendClusterName,
				},
			},
		},
		Decorator: &routepb.Decorator{
			Operation: util.SpanNamePrefix,
		},
	}
}

// Try to catch malformed preflight CORS requests. They are still OPTIONS,
// but are missing the required CORS headers.
func genPreflightCorsMissingHeadersRoute() *routepb.Route {
	return &routepb.Route{
		Match: &routepb.RouteMatch{
			PathSpecifier: &routepb.RouteMatch_Prefix{
				Prefix: "/",
			},
			Headers: []*routepb.HeaderMatcher{
				{
					Name: ":method",
					HeaderMatchSpecifier: &routepb.HeaderMatcher_StringMatch{
						StringMatch: &matcherpb.StringMatcher{
							MatchPattern: &matcherpb.StringMatcher_Exact{
								Exact: "OPTIONS",
							},
						},
					},
				},
			},
		},
		Action: &routepb.Route_DirectResponse{
			DirectResponse: &routepb.DirectResponseAction{
				Status: http.StatusBadRequest,
				Body: &corepb.DataSource{
					Specifier: &corepb.DataSource_InlineString{
						InlineString: fmt.Sprintf("The CORS preflight request is missing one (or more) of the following required headers [Origin, Access-Control-Request-Method] or has an unmatched Origin header."),
					},
				},
			},
		},
		Decorator: &routepb.Decorator{
			Operation: util.SpanNamePrefix,
		},
	}
}
