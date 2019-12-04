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

	"github.com/GoogleCloudPlatform/api-proxy/src/go/configinfo"

	commonpb "github.com/GoogleCloudPlatform/api-proxy/src/go/proto/api/envoy/http/common"
	v2pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	routepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
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

	if serviceInfo.Options.EnableBackendRouting {
		brRoute, err := makeDynamicRoutingConfig(serviceInfo)
		if err != nil {
			return nil, err
		}
		host.Routes = brRoute
	}

	host.Routes = append(host.Routes, &routepb.Route{
		Match: &routepb.RouteMatch{
			PathSpecifier: &routepb.RouteMatch_Prefix{
				Prefix: "/",
			},
		},
		Action: &routepb.Route_Route{
			Route: &routepb.RouteAction{
				ClusterSpecifier: &routepb.RouteAction_Cluster{
					Cluster: serviceInfo.ApiName},
			},
		},
	})

	switch serviceInfo.Options.CorsPreset {
	case "basic":
		org := serviceInfo.Options.CorsAllowOrigin
		if org == "" {
			return nil, fmt.Errorf("cors_allow_origin cannot be empty when cors_preset=basic")
		}
		host.Cors = &routepb.CorsPolicy{
			AllowOrigin: []string{org},
		}
	case "cors_with_regex":
		orgReg := serviceInfo.Options.CorsAllowOriginRegex
		if orgReg == "" {
			return nil, fmt.Errorf("cors_allow_origin_regex cannot be empty when cors_preset=cors_with_regex")
		}
		host.Cors = &routepb.CorsPolicy{
			AllowOriginRegex: []string{orgReg},
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
		if method.BackendInfo == nil || method.BackendInfo.TranslationType == confpb.BackendRule_PATH_TRANSLATION_UNSPECIFIED {
			continue
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
					},
				},
			}
			backendRoutes = append(backendRoutes, &r)
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

	// Replacing query parameters inside "{}" by regex "[^\/]+", which means
	// any character except `/`, also adds `$` to match to the end of the string.
	if re.MatchString(httpRule.UriTemplate) {
		routeMatcher = routepb.RouteMatch{
			PathSpecifier: &routepb.RouteMatch_Regex{
				Regex: re.ReplaceAllString(httpRule.UriTemplate, `[^\/]+`) + `$`,
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
