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
	"regexp"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	commonpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/common"
	v2pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	routepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"
	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"
)

const (
	routeName       = "local_route"
	virtualHostName = "backend"
)

func MakeRouteConfig(serviceInfo *configinfo.ServiceInfo) (*v2pb.RouteConfiguration, error) {
	var virtualHosts []*routepb.VirtualHost
	host := routepb.VirtualHost{
		Name:    virtualHostName,
		Domains: []string{"*"},
	}

	// Per-selector routes for dynamic routing.
	brRoutes, err := makeDynamicRoutingConfig(serviceInfo)
	if err != nil {
		return nil, err
	}
	host.Routes = brRoutes

	if len(host.Routes) == 0 {
		// Catch-all route if dynamic routing is not enabled.
		catchAllRt := &routepb.Route{
			Match: &routepb.RouteMatch{
				PathSpecifier: &routepb.RouteMatch_Prefix{
					Prefix: "/",
				},
			},
			Action: &routepb.Route_Route{
				Route: &routepb.RouteAction{
					ClusterSpecifier: &routepb.RouteAction_Cluster{
						Cluster: serviceInfo.BackendClusterName(),
					},
					// Use the default deadline for the catch-all route.
					// If a customer needs to override this, dynamic routing must be used.
					// This is the intended design of the feature (b/147813008).
					Timeout: ptypes.DurationProto(util.DefaultResponseDeadline),
				},
			},
		}
		if serviceInfo.Options.EnableHSTS {
			catchAllRt.ResponseHeadersToAdd = []*corepb.HeaderValueOption{
				{
					Header: &corepb.HeaderValue{
						Key:   util.HSTSHeaderKey,
						Value: util.HSTSHeaderValue,
					},
				},
			}
		}

		host.Routes = append(host.Routes, catchAllRt)

		jsonStr, _ := util.ProtoToJson(catchAllRt)
		glog.Infof("adding catch-all routing configuration: %v", jsonStr)
	}

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
		host.Cors = &routepb.CorsPolicy{
			AllowOriginStringMatch: []*matcher.StringMatcher{
				{
					MatchPattern: &matcher.StringMatcher_SafeRegex{
						SafeRegex: &matcher.RegexMatcher{
							EngineType: &matcher.RegexMatcher_GoogleRe2{
								GoogleRe2: &matcher.RegexMatcher_GoogleRE2{
									MaxProgramSize: &wrapperspb.UInt32Value{
										Value: util.GoogleRE2MaxProgramSize,
									},
								},
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
						Cluster: serviceInfo.BackendClusterName(),
					},
				},
			},
		}
		host.Routes = append(host.Routes, corsRoute)

		jsonStr, _ := util.ProtoToJson(corsRoute)
		glog.Infof("adding cors route configuration: %v", jsonStr)
	}

	virtualHosts = append(virtualHosts, &host)
	return &v2pb.RouteConfiguration{
		Name:         routeName,
		VirtualHosts: virtualHosts,
	}, nil
}

func makeDynamicRoutingConfig(serviceInfo *configinfo.ServiceInfo) ([]*routepb.Route, error) {
	var backendRoutes []*routepb.Route
	for _, operation := range serviceInfo.Operations {
		method := serviceInfo.Methods[operation]
		var routeMatcher *routepb.RouteMatch
		if method.BackendInfo == nil {
			continue
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

		for _, httpRule := range method.HttpRule {
			if routeMatcher = makeHttpRouteMatcher(httpRule); routeMatcher == nil {
				return nil, fmt.Errorf("error making HTTP route matcher for selector: %v", operation)
			}

			r := routepb.Route{
				Match: routeMatcher,
				Action: &routepb.Route_Route{
					Route: &routepb.RouteAction{
						ClusterSpecifier: &routepb.RouteAction_Cluster{
							Cluster: method.BackendInfo.ClusterName,
						},
						HostRewriteSpecifier: &routepb.RouteAction_HostRewrite{
							HostRewrite: method.BackendInfo.Hostname,
						},
						Timeout: ptypes.DurationProto(respTimeout),
					},
				},
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
			glog.Infof("adding Dynamic Routing configuration: %v", jsonStr)
		}
	}
	return backendRoutes, nil
}

func makeHttpRouteMatcher(httpRule *commonpb.Pattern) *routepb.RouteMatch {
	if httpRule == nil {
		return nil
	}
	var routeMatcher routepb.RouteMatch
	re := regexp.MustCompile(`{[^{}]+}`)

	// Replace path templates inside "{}" by regex "[^\/]+", which means
	// any character except `/`, also adds `$` to match to the end of the string.
	if re.MatchString(httpRule.UriTemplate) {
		routeMatcher = routepb.RouteMatch{
			PathSpecifier: &routepb.RouteMatch_SafeRegex{
				SafeRegex: &matcher.RegexMatcher{
					EngineType: &matcher.RegexMatcher_GoogleRe2{
						GoogleRe2: &matcher.RegexMatcher_GoogleRE2{
							MaxProgramSize: &wrapperspb.UInt32Value{
								Value: util.GoogleRE2MaxProgramSize,
							},
						},
					},
					Regex: re.ReplaceAllString(httpRule.UriTemplate, `[^\/]+`) + `$`,
				},
			},
		}
	} else {
		// Match with HttpHeader method. Some methods may have same path.
		routeMatcher = routepb.RouteMatch{
			PathSpecifier: &routepb.RouteMatch_Path{
				Path: httpRule.UriTemplate,
			},
		}
	}
	routeMatcher.Headers = []*routepb.HeaderMatcher{
		{
			Name: ":method",
			HeaderMatchSpecifier: &routepb.HeaderMatcher_ExactMatch{
				ExactMatch: httpRule.HttpMethod,
			},
		},
	}
	return &routeMatcher
}
