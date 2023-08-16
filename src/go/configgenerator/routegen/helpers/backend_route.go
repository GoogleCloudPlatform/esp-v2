package helpers

import (
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	matcherpb "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
)

// BackendRouteGenerator generates routes that forward request to the backend.
// (i.e. NO direct response routes are generated)
//
// This should NOT be used directly.
// Use it via an abstraction like RemoteBackendRoute or LocalBackendRoute.
type BackendRouteGenerator struct {
	DisallowColonInWildcardPathSegment bool
	RetryCfg                           *RouteRetryConfiger
	HSTSCfg                            *RouteHSTSConfiger
	OperationNameCfg                   *RouteOperationNameConfiger
	DeadlineCfg                        *RouteDeadlineConfiger
}

// MethodCfg is all the config needed to generate routes for a single
// One Platform operation (RPC method).
type MethodCfg struct {
	OperationName      string
	MethodShortName    string
	BackendClusterName string
	Deadline           time.Duration
	HTTPRule           *httppattern.Pattern
}

// GenRoutesForMethod generates the route config for the given URI template.
//
// Forked from `route_generator.go: makeRoute()`
func (r *BackendRouteGenerator) GenRoutesForMethod(methodCfg *MethodCfg) ([]*routepb.Route, error) {
	routeMatchers, err := makeBackedRouteMatchers(methodCfg.HTTPRule, r.DisallowColonInWildcardPathSegment)
	if err != nil {
		return nil, fmt.Errorf("fail to make backend route matchers for operation %q: %v", methodCfg.OperationName, err)
	}

	var routes []*routepb.Route
	for _, routeMatcher := range routeMatchers {
		routeAction := &routepb.RouteAction{
			ClusterSpecifier: &routepb.RouteAction_Cluster{
				Cluster: methodCfg.BackendClusterName,
			},
		}

		MaybeAddDeadlines(r.DeadlineCfg, routeAction, methodCfg.Deadline)
		if err := MaybeAddRetryPolicy(r.RetryCfg, routeAction); err != nil {
			return nil, err
		}

		route := &routepb.Route{
			Name:  methodCfg.OperationName,
			Match: routeMatcher,
			Action: &routepb.Route_Route{
				Route: routeAction,
			},
			Decorator: &routepb.Decorator{
				// TODO(taoxuy@): check if the generated span name length less than the limit.
				// Note we don't add ApiName to reduce the length of the span name.
				Operation: fmt.Sprintf("%s %s", util.SpanNamePrefix, methodCfg.MethodShortName),
			},
		}

		MaybeAddHSTSHeader(r.HSTSCfg, route)
		MaybeAddOperationNameHeader(r.OperationNameCfg, route, methodCfg.OperationName)

		routes = append(routes, route)
	}

	return routes, nil
}

// makeBackedRouteMatchers creates all route matchers for a single HTTP rule.
// This only consists of routes that will be forwarded to the backend.
//
// Forked from `route_generator.go: makeHttpRouteMatchers()`
func makeBackedRouteMatchers(httpRule *httppattern.Pattern, disallowColonInWildcardPathSegment bool) ([]*routepb.RouteMatch, error) {
	if httpRule == nil {
		return nil, fmt.Errorf("httpRule is nil")
	}

	type routeMatchWrapper struct {
		*routepb.RouteMatch
		UriTemplate string
	}

	// Create matcher with header match for `:path`.
	var routeMatchWrappers []*routeMatchWrapper
	if httpRule.UriTemplate.IsExactMatch() {
		pathNoTrailingSlash := httpRule.UriTemplate.ExactMatchString(false)
		pathWithTrailingSlash := httpRule.UriTemplate.ExactMatchString(true)

		routeMatchWrappers = append(routeMatchWrappers, &routeMatchWrapper{
			RouteMatch:  makeHttpExactPathRouteMatcher(pathNoTrailingSlash),
			UriTemplate: pathNoTrailingSlash,
		})

		if pathWithTrailingSlash != pathNoTrailingSlash {
			routeMatchWrappers = append(routeMatchWrappers, &routeMatchWrapper{
				RouteMatch:  makeHttpExactPathRouteMatcher(pathWithTrailingSlash),
				UriTemplate: pathWithTrailingSlash,
			})
		}
	} else {
		routeMatchWrappers = append(routeMatchWrappers, &routeMatchWrapper{
			RouteMatch: &routepb.RouteMatch{
				PathSpecifier: &routepb.RouteMatch_SafeRegex{
					SafeRegex: &matcherpb.RegexMatcher{
						Regex: httpRule.UriTemplate.Regex(disallowColonInWildcardPathSegment),
					},
				},
			},
			UriTemplate: httpRule.UriTemplate.Regex(disallowColonInWildcardPathSegment),
		})

	}

	// Add on header match for `:method`.
	var routeMatchers []*routepb.RouteMatch
	for _, routeMatch := range routeMatchWrappers {
		routeMatcher := routeMatch.RouteMatch
		routeMatchers = append(routeMatchers, routeMatcher)

		if httpRule.HttpMethod != httppattern.HttpMethodWildCard {
			routeMatcher.Headers = []*routepb.HeaderMatcher{
				{
					Name: ":method",
					HeaderMatchSpecifier: &routepb.HeaderMatcher_StringMatch{
						StringMatch: &matcherpb.StringMatcher{
							MatchPattern: &matcherpb.StringMatcher_Exact{
								Exact: httpRule.HttpMethod,
							},
						},
					},
				},
			}
		}
	}

	return routeMatchers, nil
}

func makeHttpExactPathRouteMatcher(path string) *routepb.RouteMatch {
	return &routepb.RouteMatch{
		PathSpecifier: &routepb.RouteMatch_Path{
			Path: path,
		},
	}
}
