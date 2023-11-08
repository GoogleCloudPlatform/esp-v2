package helpers

import (
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	matcherpb "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"google.golang.org/protobuf/types/known/anypb"
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

// NewBackendRouteGeneratorFromOPConfig creates a BackendRouteGenerator from
// ESPv2 options.
func NewBackendRouteGeneratorFromOPConfig(opts options.ConfigGeneratorOptions) *BackendRouteGenerator {
	return &BackendRouteGenerator{
		DisallowColonInWildcardPathSegment: opts.DisallowColonInWildcardPathSegment,
		RetryCfg:                           NewRouteRetryConfigerFromOPConfig(opts),
		HSTSCfg:                            NewRouteHSTSConfigerFromOPConfig(opts),
		OperationNameCfg:                   NewRouteOperationNameConfigerFromOPConfig(opts),
		DeadlineCfg:                        NewRouteDeadlineConfigerFromOPConfig(opts),
	}
}

// MethodCfg is all the config needed to generate routes for a single
// One Platform operation (RPC method).
type MethodCfg struct {
	OperationName      string
	BackendClusterName string
	HostRewrite        string
	Deadline           time.Duration
	IsStreaming        bool
	HTTPPattern        *httppattern.Pattern
}

// GenRoutesForMethod generates the route config for the given URI template.
//
// Forked from `route_generator.go: makeRoute()`
func (r *BackendRouteGenerator) GenRoutesForMethod(methodCfg *MethodCfg, filterGens []filtergen.FilterGenerator) ([]*routepb.Route, error) {
	methodName, err := util.SelectorToMethodName(methodCfg.OperationName)
	if err != nil {
		return nil, fmt.Errorf("fail to parse method short name from selector %q: %v", methodCfg.OperationName, err)
	}

	routeMatchers, err := MakePerMethodRouteMatchers(methodCfg.HTTPPattern, r.DisallowColonInWildcardPathSegment)
	if err != nil {
		return nil, fmt.Errorf("fail to make backend per-method route matchers for operation %q: %v", methodCfg.OperationName, err)
	}

	var routes []*routepb.Route
	for i, routeMatcher := range routeMatchers {
		routeAction := &routepb.RouteAction{
			ClusterSpecifier: &routepb.RouteAction_Cluster{
				Cluster: methodCfg.BackendClusterName,
			},
		}

		if methodCfg.HostRewrite != "" {
			routeAction.HostRewriteSpecifier = &routepb.RouteAction_HostRewriteLiteral{
				HostRewriteLiteral: methodCfg.HostRewrite,
			}
		}

		MaybeAddDeadlines(r.DeadlineCfg, routeAction, methodCfg.Deadline, methodCfg.IsStreaming)
		if err := MaybeAddRetryPolicy(r.RetryCfg, routeAction); err != nil {
			return nil, err
		}

		perFilterConfig, err := makePerRouteFilterConfig(methodCfg.OperationName, methodCfg.HTTPPattern, filterGens)
		if err != nil {
			return nil, fmt.Errorf("fail to make per-route filter config for route matcher %d: %v", i, err)
		}

		route := &routepb.Route{
			Name:  methodCfg.OperationName,
			Match: routeMatcher.RouteMatch,
			Action: &routepb.Route_Route{
				Route: routeAction,
			},
			Decorator: &routepb.Decorator{
				// TODO(taoxuy@): check if the generated span name length less than the limit.
				// Note we don't add ApiName to reduce the length of the span name.
				Operation: fmt.Sprintf("%s %s", util.SpanNamePrefix, methodName),
			},
			TypedPerFilterConfig: perFilterConfig,
		}

		MaybeAddHSTSHeader(r.HSTSCfg, route)
		MaybeAddOperationNameHeader(r.OperationNameCfg, route, methodCfg.OperationName)

		routes = append(routes, route)
	}

	return routes, nil
}

type RouteMatchWrapper struct {
	*routepb.RouteMatch
	UriTemplate string
}

// MakePerMethodRouteMatchers creates all route matchers for a single HTTP rule.
func MakePerMethodRouteMatchers(httpRule *httppattern.Pattern, disallowColonInWildcardPathSegment bool) ([]*RouteMatchWrapper, error) {
	routeMatchers, err := MakeRouteMatchers(httpRule, disallowColonInWildcardPathSegment)
	if err != nil {
		return nil, fmt.Errorf("fail to make backend route matchers: %v", err)
	}

	// Add on header match for `:method`.
	for _, routeMatch := range routeMatchers {
		routeMatcher := routeMatch.RouteMatch

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

// MakeRouteMatchers creates all route matchers for a single HTTP rule.
// Does not add on :method matchers.
func MakeRouteMatchers(httpRule *httppattern.Pattern, disallowColonInWildcardPathSegment bool) ([]*RouteMatchWrapper, error) {
	if httpRule == nil {
		return nil, fmt.Errorf("httpRule is nil")
	}

	// Create matcher with header match for `:path`.
	var routeMatchWrappers []*RouteMatchWrapper
	if httpRule.UriTemplate.IsExactMatch() {
		pathNoTrailingSlash := httpRule.UriTemplate.ExactMatchString(false)
		pathWithTrailingSlash := httpRule.UriTemplate.ExactMatchString(true)

		routeMatchWrappers = append(routeMatchWrappers, &RouteMatchWrapper{
			RouteMatch:  makeHttpExactPathRouteMatcher(pathNoTrailingSlash),
			UriTemplate: pathNoTrailingSlash,
		})

		if pathWithTrailingSlash != pathNoTrailingSlash {
			routeMatchWrappers = append(routeMatchWrappers, &RouteMatchWrapper{
				RouteMatch:  makeHttpExactPathRouteMatcher(pathWithTrailingSlash),
				UriTemplate: pathWithTrailingSlash,
			})
		}
	} else {
		routeMatchWrappers = append(routeMatchWrappers, &RouteMatchWrapper{
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

	return routeMatchWrappers, nil
}

func makeHttpExactPathRouteMatcher(path string) *routepb.RouteMatch {
	return &routepb.RouteMatch{
		PathSpecifier: &routepb.RouteMatch_Path{
			Path: path,
		},
	}
}

// makePerRouteFilterConfig generates the per-route config across all filters
// for a single method and http rule.
func makePerRouteFilterConfig(operation string, httpRule *httppattern.Pattern, filterGens []filtergen.FilterGenerator) (map[string]*anypb.Any, error) {
	perFilterConfig := make(map[string]*anypb.Any)

	for _, filterGen := range filterGens {
		config, err := filterGen.GenPerRouteConfig(operation, httpRule)
		if err != nil {
			return perFilterConfig, fmt.Errorf("fail to generate per-route config for filter %q: %v", filterGen.FilterName(), err)
		}
		if config == nil {
			continue
		}

		perRouteFilterConfig, err := anypb.New(config)
		if err != nil {
			return nil, fmt.Errorf("fail to marshal per-route config to Any for filter %q: %v", filterGen.FilterName(), err)
		}
		perFilterConfig[filterGen.FilterName()] = perRouteFilterConfig
	}

	return perFilterConfig, nil
}
