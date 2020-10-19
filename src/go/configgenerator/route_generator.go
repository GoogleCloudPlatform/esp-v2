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
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"

	aupb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v9/http/backend_auth"
	commonpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v9/http/common"
	prpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v9/http/path_rewrite"
	scpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v9/http/service_control"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
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

	// Per-selector routes for both local and remote backends.
	brRoutes, err := makeRouteTable(serviceInfo)
	if err != nil {
		return nil, err
	}
	host.Routes = brRoutes

	switch serviceInfo.Options.CorsPreset {
	case "basic":
		org := serviceInfo.Options.CorsAllowOrigin
		if org == "" {
			return nil, fmt.Errorf("cors_allow_origin cannot be empty when cors_preset=basic")
		}
		host.Cors = &routepb.CorsPolicy{
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
			return nil, fmt.Errorf("cors_allow_origin_regex cannot be empty when cors_preset=cors_with_regex")
		}
		if err := util.ValidateRegexProgramSize(orgReg, util.GoogleRE2MaxProgramSize); err != nil {
			return nil, fmt.Errorf("invalid cors origin regex: %v", err)
		}
		host.Cors = &routepb.CorsPolicy{
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
			return nil, fmt.Errorf("cors_preset must be set in order to enable CORS support")
		}
	default:
		return nil, fmt.Errorf(`cors_preset must be either "basic" or "cors_with_regex"`)
	}

	if host.GetCors() != nil {
		host.GetCors().AllowMethods = serviceInfo.Options.CorsAllowMethods
		host.GetCors().AllowHeaders = serviceInfo.Options.CorsAllowHeaders
		host.GetCors().ExposeHeaders = serviceInfo.Options.CorsExposeHeaders
		host.GetCors().AllowCredentials = &wrapperspb.BoolValue{Value: serviceInfo.Options.CorsAllowCredentials}

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
		host.Routes = append(host.Routes, corsRoute)

		jsonStr, _ := util.ProtoToJson(corsRoute)
		glog.Infof("adding cors route configuration: %v", jsonStr)
	}

	virtualHosts = append(virtualHosts, &host)
	return &routepb.RouteConfiguration{
		Name:         routeName,
		VirtualHosts: virtualHosts,
	}, nil
}

func MakePathRewriteConfig(method *configinfo.MethodInfo, httpRule *commonpb.Pattern) *prpb.PerRouteFilterConfig {
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
		// TODO(qiwzhang): Once httppattern.Urltemplate parser is used, change this
		// checking to use len(template.variable) > 0
		if strings.ContainsRune(httpRule.UriTemplate, '{') {
			constPath.UrlTemplate = httpRule.UriTemplate
		}
		return &prpb.PerRouteFilterConfig{
			PathTranslationSpecifier: &prpb.PerRouteFilterConfig_ConstantPath{
				ConstantPath: constPath,
			},
		}
	}
	return nil
}

func makePerRouteFilterConfig(operation string, method *configinfo.MethodInfo, httpRule *commonpb.Pattern) map[string]*anypb.Any {
	perFilterConfig := make(map[string]*anypb.Any)

	// Always add ServiceControl PerRouteConfig
	scPerRoute := &scpb.PerRouteFilterConfig{
		OperationName: operation,
	}
	scpr, _ := ptypes.MarshalAny(scPerRoute)
	perFilterConfig[util.ServiceControl] = scpr

	// add BackendAuth PerRouteConfig if needed
	if method.BackendInfo != nil && method.BackendInfo.JwtAudience != "" {
		auPerRoute := &aupb.PerRouteFilterConfig{
			JwtAudience: method.BackendInfo.JwtAudience,
		}
		aupr, _ := ptypes.MarshalAny(auPerRoute)
		perFilterConfig[util.BackendAuth] = aupr
	}

	// add PathRewrite PerRouteConfig if needed
	if pr := MakePathRewriteConfig(method, httpRule); pr != nil {
		prAny, _ := ptypes.MarshalAny(pr)
		perFilterConfig[util.PathRewrite] = prAny
	}
	return perFilterConfig
}

func makeRouteTable(serviceInfo *configinfo.ServiceInfo) ([]*routepb.Route, error) {
	var backendRoutes []*routepb.Route
	for _, operation := range serviceInfo.Operations {
		method := serviceInfo.Methods[operation]
		var routeMatcher *routepb.RouteMatch

		// Response timeouts are not compatible with streaming methods (documented in Envoy).
		// If this method is non-unary gRPC, explicitly set 0s to disable the timeout.
		// This even applies for routes with gRPC-JSON transcoding where only the upstream is streaming.
		var respTimeout time.Duration
		if method.IsStreaming {
			respTimeout = 0 * time.Second
		} else {
			respTimeout = method.BackendInfo.Deadline
		}

		for _, httpRule := range method.HttpRule {
			var err error
			if routeMatcher, err = makeHttpRouteMatcher(httpRule); err != nil {
				return nil, fmt.Errorf("error making HTTP route matcher for selector (%v): %v", operation, err)
			}

			r := routepb.Route{
				Match: routeMatcher,
				Action: &routepb.Route_Route{
					Route: &routepb.RouteAction{
						ClusterSpecifier: &routepb.RouteAction_Cluster{
							Cluster: method.BackendInfo.ClusterName,
						},
						Timeout: ptypes.DurationProto(respTimeout),
					},
				},
				Decorator: &routepb.Decorator{
					// Note we don't add ApiName to reduce the length of the span name.
					Operation: fmt.Sprintf("%s %s", util.SpanNamePrefix, method.ShortName),
				},
				TypedPerFilterConfig: makePerRouteFilterConfig(operation, method, httpRule),
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
			backendRoutes = append(backendRoutes, &r)

			jsonStr, _ := util.ProtoToJson(&r)
			glog.Infof("adding route: %v", jsonStr)
		}
	}
	return backendRoutes, nil
}

func makeHttpRouteMatcher(httpRule *commonpb.Pattern) (*routepb.RouteMatch, error) {
	if httpRule == nil {
		return nil, fmt.Errorf("httpRule is nil")
	}
	var routeMatcher routepb.RouteMatch

	regex := util.WildcardMatcherForPath(httpRule.UriTemplate)
	if regex == "" {
		// Match with HttpHeader method. Some methods may have same path.
		routeMatcher = routepb.RouteMatch{
			PathSpecifier: &routepb.RouteMatch_Path{
				Path: httpRule.UriTemplate,
			},
		}
	} else {
		if err := util.ValidateRegexProgramSize(regex, util.GoogleRE2MaxProgramSize); err != nil {
			return nil, fmt.Errorf("invalid route path regex: %v, generated by UriTemplate: %s", err, httpRule.UriTemplate)
		}

		routeMatcher = routepb.RouteMatch{
			PathSpecifier: &routepb.RouteMatch_SafeRegex{
				SafeRegex: &matcher.RegexMatcher{
					EngineType: &matcher.RegexMatcher_GoogleRe2{
						GoogleRe2: &matcher.RegexMatcher_GoogleRE2{},
					},
					Regex: regex,
				},
			},
		}
	}

	if httpRule.HttpMethod != "*" {
		routeMatcher.Headers = []*routepb.HeaderMatcher{
			{
				Name: ":method",
				HeaderMatchSpecifier: &routepb.HeaderMatcher_ExactMatch{
					ExactMatch: httpRule.HttpMethod,
				},
			},
		}
	}
	return &routeMatcher, nil
}
