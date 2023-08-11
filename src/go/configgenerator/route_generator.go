// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configgenerator

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	corspb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/cors/v3"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/golang/glog"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	routeName       = "local_route"
	virtualHostName = "backend"
)

// perResourceCallback is a callback to generate per-resource config for a
// specific FilterGenerator.
type perResourceCallback func(gen filtergen.FilterGenerator) (proto.Message, error)

func makeRouteConfig(serviceInfo *configinfo.ServiceInfo, filterGenerators []filtergen.FilterGenerator) (*routepb.RouteConfiguration, error) {
	var virtualHosts []*routepb.VirtualHost
	host := routepb.VirtualHost{
		Name:    virtualHostName,
		Domains: []string{"*"},
	}

	perHostConfig, err := makePerVHostFilterConfig(host.Name, filterGenerators)
	if err != nil {
		return nil, fmt.Errorf("fail to make per-vHost filter config for virtual host %q: %v", host.Name, err)
	}
	host.TypedPerFilterConfig = perHostConfig

	// The router will use the first matched route, so the order of routes is important.
	// Right now, the order of routes are:
	// - backend routes
	// - cors routes
	// - fallback `method not allowed` routes
	// - catch all `not found` routes
	//
	//
	// // Per-selector routes for both local and remote backends.
	backendRoutes, methodNotAllowedRoutes, err := MakeRouteTable(serviceInfo, filterGenerators)
	if err != nil {
		return nil, err
	}
	host.Routes = backendRoutes

	_, corsRoutes, err := makeRouteCors(serviceInfo)
	if err != nil {
		return nil, err
	}

	if len(corsRoutes) > 0 {
		host.Routes = append(host.Routes, corsRoutes...)
		for i, corsRoute := range corsRoutes {
			jsonStr, _ := util.ProtoToJson(corsRoute)
			glog.Infof("adding cors route configuration [%v]: %v", i, jsonStr)
		}
	}

	host.Routes = append(host.Routes, methodNotAllowedRoutes...)

	host.Routes = append(host.Routes, makeCatchAllNotFoundRoute())

	virtualHosts = append(virtualHosts, &host)

	requestHeaders, err := makeRequestHeadersToAdd(serviceInfo)
	if err != nil {
		return nil, err
	}
	responseHeaders, err := makeResponseHeadersToAdd(serviceInfo)
	if err != nil {
		return nil, err
	}
	return &routepb.RouteConfiguration{
		Name:                 routeName,
		VirtualHosts:         virtualHosts,
		RequestHeadersToAdd:  requestHeaders,
		ResponseHeadersToAdd: responseHeaders,
	}, nil
}

func makeHeaders(headers string, a bool) ([]*corepb.HeaderValueOption, error) {
	var l []*corepb.HeaderValueOption
	for _, h := range strings.Split(headers, ";") {
		if h == "" {
			continue
		}
		key_value := strings.Split(h, "=")
		if len(key_value) != 2 {
			return l, fmt.Errorf("invalid header: %v. should be in key=value format.", h)
		}
		if key_value[0] == "" {
			return l, fmt.Errorf("header key should not be empty for: %v.", h)
		}
		l = append(l, &corepb.HeaderValueOption{
			Header: &corepb.HeaderValue{
				Key:   key_value[0],
				Value: key_value[1],
			},
			Append: &wrapperspb.BoolValue{
				Value: a,
			},
		})
	}
	return l, nil
}

func makeRequestHeadersToAdd(serviceInfo *configinfo.ServiceInfo) ([]*corepb.HeaderValueOption, error) {
	l, err := makeHeaders(serviceInfo.Options.AddRequestHeaders, false)
	if err != nil {
		return l, err
	}

	m, err := makeHeaders(serviceInfo.Options.AppendRequestHeaders, true)
	if err != nil {
		return l, err
	}

	l = append(l, m...)
	return l, nil
}

func makeResponseHeadersToAdd(serviceInfo *configinfo.ServiceInfo) ([]*corepb.HeaderValueOption, error) {
	l, err := makeHeaders(serviceInfo.Options.AddResponseHeaders, false)
	if err != nil {
		return l, err
	}

	m, err := makeHeaders(serviceInfo.Options.AppendResponseHeaders, true)
	if err != nil {
		return l, err
	}

	l = append(l, m...)
	return l, nil
}

func makeRouteCors(serviceInfo *configinfo.ServiceInfo) (*corspb.CorsPolicy, []*routepb.Route, error) {
	var cors *corspb.CorsPolicy
	originMatcher := &routepb.HeaderMatcher{
		Name: "origin",
	}
	switch serviceInfo.Options.CorsPreset {
	case "basic":
		org := serviceInfo.Options.CorsAllowOrigin
		if org == "" {
			return nil, nil, fmt.Errorf("cors_allow_origin cannot be empty when cors_preset=basic")
		}
		stringMatcher := &matcher.StringMatcher{
			MatchPattern: &matcher.StringMatcher_Exact{
				Exact: org,
			},
		}

		cors = &corspb.CorsPolicy{
			AllowOriginStringMatch: []*matcher.StringMatcher{stringMatcher},
		}
		if org == "*" {
			originMatcher.HeaderMatchSpecifier = &routepb.HeaderMatcher_PresentMatch{
				PresentMatch: true,
			}
		} else {
			originMatcher.HeaderMatchSpecifier = &routepb.HeaderMatcher_StringMatch{
				StringMatch: stringMatcher,
			}
		}
	case "cors_with_regex":
		orgReg := serviceInfo.Options.CorsAllowOriginRegex
		if orgReg == "" {
			return nil, nil, fmt.Errorf("cors_allow_origin_regex cannot be empty when cors_preset=cors_with_regex")
		}
		if err := util.ValidateRegexProgramSize(orgReg, util.GoogleRE2MaxProgramSize); err != nil {
			return nil, nil, fmt.Errorf("invalid cors origin regex: %v", err)
		}
		stringMatcher := &matcher.StringMatcher{
			MatchPattern: &matcher.StringMatcher_SafeRegex{
				SafeRegex: &matcher.RegexMatcher{
					Regex: orgReg,
				},
			},
		}
		cors = &corspb.CorsPolicy{
			AllowOriginStringMatch: []*matcher.StringMatcher{stringMatcher},
		}
		originMatcher.HeaderMatchSpecifier = &routepb.HeaderMatcher_StringMatch{
			StringMatch: stringMatcher,
		}
	case "":
		glog.Infof("CORS support is disabled")
	default:
		return nil, nil, fmt.Errorf(`cors_preset must be either "basic" or "cors_with_regex"`)
	}

	if cors == nil {
		return nil, nil, nil
	}
	cors.MaxAge = strconv.Itoa(int(serviceInfo.Options.CorsMaxAge.Seconds()))
	cors.AllowMethods = serviceInfo.Options.CorsAllowMethods
	cors.AllowHeaders = serviceInfo.Options.CorsAllowHeaders
	cors.ExposeHeaders = serviceInfo.Options.CorsExposeHeaders
	cors.AllowCredentials = &wrapperspb.BoolValue{Value: serviceInfo.Options.CorsAllowCredentials}

	// In order apply Envoy cors policy, need to have a catch-all route to match
	// preflight CORS requests.
	preflightCorsRoute := &routepb.Route{
		Match: &routepb.RouteMatch{
			PathSpecifier: &routepb.RouteMatch_Prefix{
				Prefix: "/",
			},
			Headers: []*routepb.HeaderMatcher{
				{
					Name: ":method",
					HeaderMatchSpecifier: &routepb.HeaderMatcher_StringMatch{
						StringMatch: &matcher.StringMatcher{
							MatchPattern: &matcher.StringMatcher_Exact{
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
		// for cors filter to work.
		Action: &routepb.Route_Route{
			Route: &routepb.RouteAction{
				ClusterSpecifier: &routepb.RouteAction_Cluster{
					Cluster: serviceInfo.LocalBackendClusterName(),
				},
			},
		},
		Decorator: &routepb.Decorator{
			Operation: util.SpanNamePrefix,
		},
	}

	// Try to catch malformed preflight CORS requests. They are still OPTIONS,
	// but are missing the required CORS headers.
	preflightCorsMissingHeadersRoute := &routepb.Route{
		Match: &routepb.RouteMatch{
			PathSpecifier: &routepb.RouteMatch_Prefix{
				Prefix: "/",
			},
			Headers: []*routepb.HeaderMatcher{
				{
					Name: ":method",
					HeaderMatchSpecifier: &routepb.HeaderMatcher_StringMatch{
						StringMatch: &matcher.StringMatcher{
							MatchPattern: &matcher.StringMatcher_Exact{
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

	corsRoutes := []*routepb.Route{
		// Order matters: most specific to least specific match.
		preflightCorsRoute,
		preflightCorsMissingHeadersRoute,
	}
	return cors, corsRoutes, nil
}

// makePerRouteFilterConfig generates the per-route config across all filters for a single method and http rule.
func makePerRouteFilterConfig(method *configinfo.MethodInfo, httpRule *httppattern.Pattern, filterGenerators []filtergen.FilterGenerator) (map[string]*anypb.Any, error) {
	perFilterConfig := make(map[string]*anypb.Any)

	for _, filterGen := range filterGenerators {
		config, err := filterGen.GenPerRouteConfig(method.Operation(), httpRule)
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

// makePerVHostFilterConfig generates the per virtual host config across all filters.
func makePerVHostFilterConfig(vHost string, filterGenerators []filtergen.FilterGenerator) (map[string]*anypb.Any, error) {
	perFilterConfig := make(map[string]*anypb.Any)

	for _, filterGen := range filterGenerators {
		config, err := filterGen.GenPerHostConfig(vHost)
		if err != nil {
			return perFilterConfig, fmt.Errorf("fail to generate per-vHost config for filter %q: %v", filterGen.FilterName(), err)
		}
		if config == nil {
			continue
		}

		perVHostFilterConfig, err := anypb.New(config)
		if err != nil {
			return nil, fmt.Errorf("fail to marshal per-vHost config to Any for filter %q: %v", filterGen.FilterName(), err)
		}
		perFilterConfig[filterGen.FilterName()] = perVHostFilterConfig
	}

	return perFilterConfig, nil
}

// MakeRouteTable generates the backendRoute and denylistRoute from serviceInfo
func MakeRouteTable(serviceInfo *configinfo.ServiceInfo, filterGenerators []filtergen.FilterGenerator) ([]*routepb.Route, []*routepb.Route, error) {
	var backendRoutes []*routepb.Route
	var methodNotAllowedRoutes []*routepb.Route
	httpPatternMethods, err := getSortMethodsByHttpPattern(serviceInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("fail to sort route match, %v", err)
	}

	seenUriTemplatesInRoute := map[string]bool{}
	for _, httpPatternMethod := range *httpPatternMethods {
		operation := httpPatternMethod.Operation
		method := serviceInfo.Methods[operation]
		httpRule := &httppattern.Pattern{
			UriTemplate: httpPatternMethod.UriTemplate,
			HttpMethod:  httpPatternMethod.HttpMethod,
		}

		// The `methodNotAllowedRouteMatchers` are the route matches covers all the defined uri templates
		// but no specific methods. As all the defined requests are matched by `routeMatchers`, the rest
		// matched by `methodNotAllowedRouteMatchers` fall in the category of `405 Method Not Allowed`.
		var routeMatchers, methodNotAllowedRouteMatchers []*routepb.RouteMatch

		var err error
		if routeMatchers, methodNotAllowedRouteMatchers, err = makeHttpRouteMatchers(httpRule, seenUriTemplatesInRoute, serviceInfo.Options.DisallowColonInWildcardPathSegment); err != nil {
			return nil, nil, fmt.Errorf("error making HTTP route matcher for operation (%v): %v", operation, err)
		}

		for _, methodNotAllowedRouteMatcher := range methodNotAllowedRouteMatchers {
			methodNotAllowedRoutes = append(methodNotAllowedRoutes, makeMethodNotAllowedRoute(methodNotAllowedRouteMatcher, httpRule.UriTemplate.Origin))
		}

		for _, routeMatcher := range routeMatchers {
			bi := method.BackendInfo
			useLocalHTTPBackend := !method.IsGRPCPath(routeMatcher.GetPath()) && !configinfo.IsDiscoveryAPI(method.Operation()) && method.HttpBackendInfo != nil
			if useLocalHTTPBackend {
				glog.Infof("Use HTTP backend for method %q.", method.Operation())
				bi = method.HttpBackendInfo
			}
			r := makeRoute(routeMatcher, method, useLocalHTTPBackend)

			r.TypedPerFilterConfig, err = makePerRouteFilterConfig(method, httpRule, filterGenerators)
			if err != nil {
				return nil, nil, fmt.Errorf("fail to make per-route filter config for operation %q: %v", operation, err)
			}

			if bi.Hostname != "" {
				// For routing to remote backends.
				r.GetRoute().HostRewriteSpecifier = &routepb.RouteAction_HostRewriteLiteral{
					HostRewriteLiteral: bi.Hostname,
				}
			}

			if serviceInfo.Options.EnableHSTS {
				r.ResponseHeadersToAdd = []*corepb.HeaderValueOption{
					{
						Header: &corepb.HeaderValue{
							Key:   util.HSTSHeaderKey,
							Value: util.HSTSHeaderValue,
						},
					},
				}
			}

			if serviceInfo.Options.EnableOperationNameHeader {
				r.RequestHeadersToAdd = []*corepb.HeaderValueOption{
					{
						Header: &corepb.HeaderValue{
							Key:   serviceInfo.Options.GeneratedHeaderPrefix + util.OperationHeaderSuffix,
							Value: operation,
						},
						Append: &wrapperspb.BoolValue{
							Value: false,
						},
					},
				}
			}

			backendRoutes = append(backendRoutes, r)

			jsonStr, err := util.ProtoToJson(r)
			if err != nil {
				return nil, nil, err
			}
			glog.Infof("adding route: %v", jsonStr)
		}
	}

	return backendRoutes, methodNotAllowedRoutes, nil
}

func makeRoute(routeMatcher *routepb.RouteMatch, method *configinfo.MethodInfo, useLocalHTTPBackend bool) *routepb.Route {
	bi := method.BackendInfo
	if useLocalHTTPBackend {
		bi = method.HttpBackendInfo
	}

	retryPolicy := &routepb.RetryPolicy{
		RetryOn: bi.RetryOns,
		NumRetries: &wrapperspb.UInt32Value{
			Value: uint32(bi.RetryNum),
		},
		RetriableStatusCodes: bi.RetriableStatusCodes,
	}

	if bi.PerTryTimeout.Nanoseconds() > 0 {
		retryPolicy.PerTryTimeout = durationpb.New(bi.PerTryTimeout)
	}

	return &routepb.Route{
		Name:  method.Operation(),
		Match: routeMatcher,
		Action: &routepb.Route_Route{
			Route: &routepb.RouteAction{
				ClusterSpecifier: &routepb.RouteAction_Cluster{
					Cluster: bi.ClusterName,
				},
				Timeout:     durationpb.New(bi.Deadline),
				IdleTimeout: durationpb.New(bi.IdleTimeout),
				RetryPolicy: retryPolicy,
			},
		},
		Decorator: &routepb.Decorator{
			// TODO(taoxuy@): check if the generated span name length less than the limit.
			// Note we don't add ApiName to reduce the length of the span name.
			Operation: fmt.Sprintf("%s %s", util.SpanNamePrefix, method.ShortName),
		},
	}
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
func makeCatchAllNotFoundRoute() *routepb.Route {
	return &routepb.Route{
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
	}
}

func makeHttpExactPathRouteMatcher(path string) *routepb.RouteMatch {
	return &routepb.RouteMatch{
		PathSpecifier: &routepb.RouteMatch_Path{
			Path: path,
		},
	}
}

func makeHttpRouteMatchers(httpRule *httppattern.Pattern, seenUriTemplatesInRoute map[string]bool, disallowColonInWildcardPathSegment bool) ([]*routepb.RouteMatch, []*routepb.RouteMatch, error) {
	if httpRule == nil {
		return nil, nil, fmt.Errorf("httpRule is nil")
	}

	type routeMatchWrapper struct {
		*routepb.RouteMatch
		UriTemplate string
	}

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
					SafeRegex: &matcher.RegexMatcher{
						Regex: httpRule.UriTemplate.Regex(disallowColonInWildcardPathSegment),
					},
				},
			},
			UriTemplate: httpRule.UriTemplate.Regex(disallowColonInWildcardPathSegment),
		})

	}

	var routeMatchers, methodNotAllowedRouteMatchers []*routepb.RouteMatch
	for _, routeMatch := range routeMatchWrappers {
		routeMatcher := routeMatch.RouteMatch
		routeMatchers = append(routeMatchers, routeMatcher)

		if httpRule.HttpMethod != httppattern.HttpMethodWildCard {
			uriTemplate := routeMatch.UriTemplate
			if ok, _ := seenUriTemplatesInRoute[uriTemplate]; !ok {
				seenUriTemplatesInRoute[uriTemplate] = true
				methodUndefinedRouterMatcherMsg := proto.Clone(routeMatcher)
				methodNotAllowedRouteMatchers = append(methodNotAllowedRouteMatchers, methodUndefinedRouterMatcherMsg.(*routepb.RouteMatch))
			}

			routeMatcher.Headers = []*routepb.HeaderMatcher{
				{
					Name: ":method",
					HeaderMatchSpecifier: &routepb.HeaderMatcher_StringMatch{
						StringMatch: &matcher.StringMatcher{
							MatchPattern: &matcher.StringMatcher_Exact{
								Exact: httpRule.HttpMethod,
							},
						},
					},
				},
			}
		}
	}

	return routeMatchers, methodNotAllowedRouteMatchers, nil
}

func getSortMethodsByHttpPattern(serviceInfo *configinfo.ServiceInfo) (*httppattern.MethodSlice, error) {
	httpPatternMethods := &httppattern.MethodSlice{}
	for _, operation := range serviceInfo.Operations {
		method := serviceInfo.Methods[operation]
		for _, httpRule := range method.HttpRule {
			httpPatternMethods.AppendMethod(&httppattern.Method{
				Pattern:   httpRule,
				Operation: operation,
			})
		}
	}

	if err := httppattern.Sort(httpPatternMethods); err != nil {
		return nil, err
	}

	return httpPatternMethods, nil
}
