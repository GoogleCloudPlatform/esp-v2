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
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"

	aupb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v9/http/backend_auth"
	prpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v9/http/path_rewrite"
	scpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v9/http/service_control"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	jwtpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	anypb "github.com/golang/protobuf/ptypes/any"
	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"
)

const (
	routeName       = "local_route"
	virtualHostName = "backend"
)

func MakeRouteConfig(serviceInfo *configinfo.ServiceInfo) (*routepb.RouteConfiguration, error) {
	var virtualHosts []*routepb.VirtualHost
	host := routepb.VirtualHost{
		Name:    virtualHostName,
		Domains: []string{"*"},
	}

	// The router will use the first matched route, so the order of routes is important.
	// Right now, the order of routes are:
	// - backend routes
	// - cors routes
	// - fallback `method not allowed` routes
	// - catch all `not found` routes
	//
	//
	// // Per-selector routes for both local and remote backends.
	backendRoutes, methodNotAllowedRoutes, err := makeRouteTable(serviceInfo)
	if err != nil {
		return nil, err
	}
	host.Routes = backendRoutes

	cors, corsRoute, err := makeRouteCors(serviceInfo)
	if err != nil {
		return nil, err
	}

	if cors != nil {
		host.Cors = cors
		host.Routes = append(host.Routes, corsRoute)
		jsonStr, _ := util.ProtoToJson(corsRoute)
		glog.Infof("adding cors route configuration: %v", jsonStr)
	}

	host.Routes = append(host.Routes, methodNotAllowedRoutes...)

	host.Routes = append(host.Routes, makeCatchAllNotFoundRoute())

	virtualHosts = append(virtualHosts, &host)
	return &routepb.RouteConfiguration{
		Name:         routeName,
		VirtualHosts: virtualHosts,
	}, nil
}

func makeRouteCors(serviceInfo *configinfo.ServiceInfo) (*routepb.CorsPolicy, *routepb.Route, error) {
	var cors *routepb.CorsPolicy
	switch serviceInfo.Options.CorsPreset {
	case "basic":
		org := serviceInfo.Options.CorsAllowOrigin
		if org == "" {
			return nil, nil, fmt.Errorf("cors_allow_origin cannot be empty when cors_preset=basic")
		}
		cors = &routepb.CorsPolicy{
			AllowOriginStringMatch: []*matcher.StringMatcher{
				{
					MatchPattern: &matcher.StringMatcher_Exact{
						Exact: org,
					},
				},
			},
		}
	case "cors_with_regex":
		orgReg := serviceInfo.Options.CorsAllowOriginRegex
		if orgReg == "" {
			return nil, nil, fmt.Errorf("cors_allow_origin_regex cannot be empty when cors_preset=cors_with_regex")
		}
		if err := util.ValidateRegexProgramSize(orgReg, util.GoogleRE2MaxProgramSize); err != nil {
			return nil, nil, fmt.Errorf("invalid cors origin regex: %v", err)
		}
		cors = &routepb.CorsPolicy{
			AllowOriginStringMatch: []*matcher.StringMatcher{
				{
					MatchPattern: &matcher.StringMatcher_SafeRegex{
						SafeRegex: &matcher.RegexMatcher{
							EngineType: &matcher.RegexMatcher_GoogleRe2{
								GoogleRe2: &matcher.RegexMatcher_GoogleRE2{},
							},
							Regex: orgReg,
						},
					},
				},
			},
		}
	case "":
		if serviceInfo.Options.CorsAllowMethods != "" || serviceInfo.Options.CorsAllowHeaders != "" ||
			serviceInfo.Options.CorsExposeHeaders != "" || serviceInfo.Options.CorsAllowCredentials {
			return nil, nil, fmt.Errorf("cors_preset must be set in order to enable CORS support")
		}
	default:
		return nil, nil, fmt.Errorf(`cors_preset must be either "basic" or "cors_with_regex"`)
	}

	if cors == nil {
		return nil, nil, nil
	}
	cors.AllowMethods = serviceInfo.Options.CorsAllowMethods
	cors.AllowHeaders = serviceInfo.Options.CorsAllowHeaders
	cors.ExposeHeaders = serviceInfo.Options.CorsExposeHeaders
	cors.AllowCredentials = &wrapperspb.BoolValue{Value: serviceInfo.Options.CorsAllowCredentials}

	// In order apply Envoy cors policy, need to have a route rule
	// to route OPTIONS request to this host
	corsRoute := &routepb.Route{
		Match: &routepb.RouteMatch{
			PathSpecifier: &routepb.RouteMatch_Prefix{
				Prefix: "/",
			},
			Headers: []*routepb.HeaderMatcher{{
				Name: ":method",
				HeaderMatchSpecifier: &routepb.HeaderMatcher_ExactMatch{
					ExactMatch: "OPTIONS",
				},
			}},
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
	return cors, corsRoute, nil
}

func MakePathRewriteConfig(method *configinfo.MethodInfo, httpRule *httppattern.Pattern) *prpb.PerRouteFilterConfig {
	if method.BackendInfo == nil {
		return nil
	}

	if method.BackendInfo.TranslationType == confpb.BackendRule_APPEND_PATH_TO_ADDRESS {
		if method.BackendInfo.Path != "" {
			return &prpb.PerRouteFilterConfig{
				PathTranslationSpecifier: &prpb.PerRouteFilterConfig_PathPrefix{
					PathPrefix: method.BackendInfo.Path,
				},
			}
		}
	}
	if method.BackendInfo.TranslationType == confpb.BackendRule_CONSTANT_ADDRESS {
		constPath := &prpb.ConstantPath{
			Path: method.BackendInfo.Path,
		}

		if uriTemplate := httpRule.UriTemplate; uriTemplate != nil && len(uriTemplate.Variables) > 0 {
			constPath.UrlTemplate = uriTemplate.ExactMatchString(false)
		}
		return &prpb.PerRouteFilterConfig{
			PathTranslationSpecifier: &prpb.PerRouteFilterConfig_ConstantPath{
				ConstantPath: constPath,
			},
		}
	}
	return nil
}

func makePerRouteFilterConfig(operation string, method *configinfo.MethodInfo, httpRule *httppattern.Pattern) (map[string]*anypb.Any, error) {
	perFilterConfig := make(map[string]*anypb.Any)

	// Always add ServiceControl PerRouteConfig
	scPerRoute := &scpb.PerRouteFilterConfig{
		OperationName: operation,
	}
	scpr, err := ptypes.MarshalAny(scPerRoute)
	if err != nil {
		return perFilterConfig, fmt.Errorf("error marshaling service_control per-route config to Any: %v", err)
	}

	perFilterConfig[util.ServiceControl] = scpr

	// add BackendAuth PerRouteConfig if needed
	if method.BackendInfo != nil && method.BackendInfo.JwtAudience != "" {
		auPerRoute := &aupb.PerRouteFilterConfig{
			JwtAudience: method.BackendInfo.JwtAudience,
		}
		aupr, err := ptypes.MarshalAny(auPerRoute)
		if err != nil {
			return perFilterConfig, fmt.Errorf("error marshaling backend_auth per-route config to Any: %v", err)
		}
		perFilterConfig[util.BackendAuth] = aupr
	}

	// add PathRewrite PerRouteConfig if needed
	if pr := MakePathRewriteConfig(method, httpRule); pr != nil {
		prAny, err := ptypes.MarshalAny(pr)
		if err != nil {
			return perFilterConfig, fmt.Errorf("error marshaling path_rewrite per-route config to Any: %v", err)
		}
		perFilterConfig[util.PathRewrite] = prAny
	}

	// add JwtAuthn PerRouteConfig
	if method.RequireAuth {
		jwtPerRoute := &jwtpb.PerRouteConfig{
			RequirementSpecifier: &jwtpb.PerRouteConfig_RequirementName{
				RequirementName: operation,
			},
		}
		jwt, err := ptypes.MarshalAny(jwtPerRoute)
		if err != nil {
			return perFilterConfig, fmt.Errorf("error marshaling jwt_authn per-route config to Any: %v", err)
		}
		perFilterConfig[util.JwtAuthn] = jwt
	}

	return perFilterConfig, nil
}

func makeRouteTable(serviceInfo *configinfo.ServiceInfo) ([]*routepb.Route, []*routepb.Route, error) {
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

		// Response timeouts are not compatible with streaming methods (documented in Envoy).
		// If this method is non-unary gRPC, explicitly set 0s to disable the timeout.
		// This even applies for routes with gRPC-JSON transcoding where only the upstream is streaming.
		var respTimeout time.Duration
		if method.IsStreaming {
			respTimeout = 0 * time.Second
		} else {
			respTimeout = method.BackendInfo.Deadline
		}

		// The `methodNotAllowedRouteMatchers` are the route matches covers all the defined uri templates
		// but no specific methods. As all the defined requests are matched by `routeMatchers`, the rest
		// matched by `methodNotAllowedRouteMatchers` fall in the category of `405 Method Not Allowed`.
		var routeMatchers, methodNotAllowedRouteMatchers []*routepb.RouteMatch

		var err error
		if routeMatchers, methodNotAllowedRouteMatchers, err = makeHttpRouteMatchers(httpRule, seenUriTemplatesInRoute); err != nil {
			return nil, nil, fmt.Errorf("error making HTTP route matcher for selector (%v): %v", operation, err)
		}

		for _, methodNotAllowedRouteMatcher := range methodNotAllowedRouteMatchers {
			methodNotAllowedRoutes = append(methodNotAllowedRoutes, makeMethodNotAllowedRoute(methodNotAllowedRouteMatcher, httpRule.UriTemplate.Origin))
		}

		for _, routeMatcher := range routeMatchers {
			r := makeRoute(routeMatcher, method, respTimeout)

			r.TypedPerFilterConfig, err = makePerRouteFilterConfig(operation, method, httpRule)
			if err != nil {
				return nil, nil, fmt.Errorf("fail to make per-route filter config, %v", err)
			}

			if method.BackendInfo.Hostname != "" {
				// For routing to remote backends.
				r.GetRoute().HostRewriteSpecifier = &routepb.RouteAction_HostRewriteLiteral{
					HostRewriteLiteral: method.BackendInfo.Hostname,
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
			backendRoutes = append(backendRoutes, r)

			jsonStr, _ := util.ProtoToJson(r)
			glog.Infof("adding route: %v", jsonStr)
		}
	}

	return backendRoutes, methodNotAllowedRoutes, nil
}

func makeRoute(routeMatcher *routepb.RouteMatch, method *configinfo.MethodInfo, respTimeout time.Duration) *routepb.Route {
	return &routepb.Route{
		Match: routeMatcher,
		Action: &routepb.Route_Route{
			Route: &routepb.RouteAction{
				ClusterSpecifier: &routepb.RouteAction_Cluster{
					Cluster: method.BackendInfo.ClusterName,
				},
				Timeout: ptypes.DurationProto(respTimeout),
				RetryPolicy: &routepb.RetryPolicy{
					RetryOn: method.BackendInfo.RetryOns,
					NumRetries: &wrapperspb.UInt32Value{
						Value: uint32(method.BackendInfo.RetryNum),
					},
				},
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
	spanName := fmt.Sprintf("%s UnknownHttpMethodForPath_%s", util.SpanNamePrefix, uriTemplateInSc)

	if len(spanName) > util.SpanNameMaxByteNum {
		newSpanName := fmt.Sprintf("%s UnknownHttpMethod", util.SpanNamePrefix)
		glog.Warningf("oversized spanName: %s, replace it with the span name: %s", spanName, newSpanName)
		spanName = newSpanName
	}

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

func makeHttpRouteMatchers(httpRule *httppattern.Pattern, seenUriTemplatesInRoute map[string]bool) ([]*routepb.RouteMatch, []*routepb.RouteMatch, error) {
	if httpRule == nil {
		return nil, nil, fmt.Errorf("httpRule is nil")
	}
	var routeMatchers []*routepb.RouteMatch
	var uriTemplates []string

	if httpRule.UriTemplate.IsExactMatch() {
		pathNoTrailingSlash := httpRule.UriTemplate.ExactMatchString(false)
		pathWithTrailingSlash := httpRule.UriTemplate.ExactMatchString(true)

		uriTemplates = append(uriTemplates, pathNoTrailingSlash)
		routeMatchers = append(routeMatchers, makeHttpExactPathRouteMatcher(pathNoTrailingSlash))
		if pathWithTrailingSlash != pathNoTrailingSlash {
			uriTemplates = append(uriTemplates, pathWithTrailingSlash)
			routeMatchers = append(routeMatchers, makeHttpExactPathRouteMatcher(pathWithTrailingSlash))
		}
	} else {
		uriTemplates = append(uriTemplates, httpRule.UriTemplate.Regex())
		routeMatchers = []*routepb.RouteMatch{
			{
				PathSpecifier: &routepb.RouteMatch_SafeRegex{
					SafeRegex: &matcher.RegexMatcher{
						EngineType: &matcher.RegexMatcher_GoogleRe2{
							GoogleRe2: &matcher.RegexMatcher_GoogleRE2{},
						},
						Regex: httpRule.UriTemplate.Regex(),
					},
				},
			},
		}

	}

	var methodNotAllowedRouteMatchers []*routepb.RouteMatch
	if httpRule.HttpMethod != httppattern.HttpMethodWildCard {
		for idx, routeMatcher := range routeMatchers {
			uriTemplate := uriTemplates[idx]
			if ok, _ := seenUriTemplatesInRoute[uriTemplate]; !ok {
				seenUriTemplatesInRoute[uriTemplate] = true
				methodUndefinedRouterMatcherMsg := proto.Clone(routeMatcher)
				methodNotAllowedRouteMatchers = append(methodNotAllowedRouteMatchers, methodUndefinedRouterMatcherMsg.(*routepb.RouteMatch))
			}

			routeMatcher.Headers = []*routepb.HeaderMatcher{
				{
					Name: ":method",
					HeaderMatchSpecifier: &routepb.HeaderMatcher_ExactMatch{
						ExactMatch: httpRule.HttpMethod,
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
