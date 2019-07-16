// Copyright 2019 Google Cloud Platform Proxy Authors
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

	"cloudesf.googlesource.com/gcpproxy/src/go/flags"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/gogo/protobuf/types"

	sc "cloudesf.googlesource.com/gcpproxy/src/go/configinfo"
	commonpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/common"
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

const (
	routeName       = "local_route"
	virtualHostName = "backend"
)

func MakeRouteConfig(serviceInfo *sc.ServiceInfo) (*v2.RouteConfiguration, error) {
	var virtualHosts []route.VirtualHost
	host := route.VirtualHost{
		Name:    virtualHostName,
		Domains: []string{"*"},
	}

	if *flags.EnableBackendRouting {
		brRoute, err := makeDynamicRoutingConfig(serviceInfo)
		if err != nil {
			return nil, err
		}
		host.Routes = brRoute
	}

	host.Routes = append(host.Routes, route.Route{
		Match: route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Prefix{
				Prefix: "/",
			},
		},
		Action: &route.Route_Route{
			Route: &route.RouteAction{
				ClusterSpecifier: &route.RouteAction_Cluster{
					Cluster: serviceInfo.ApiName},
			},
		},
	})

	switch *flags.CorsPreset {
	case "basic":
		org := *flags.CorsAllowOrigin
		if org == "" {
			return nil, fmt.Errorf("cors_allow_origin cannot be empty when cors_preset=basic")
		}
		host.Cors = &route.CorsPolicy{
			AllowOrigin: []string{org},
		}
	case "cors_with_regex":
		orgReg := *flags.CorsAllowOriginRegex
		if orgReg == "" {
			return nil, fmt.Errorf("cors_allow_origin_regex cannot be empty when cors_preset=cors_with_regex")
		}
		host.Cors = &route.CorsPolicy{
			AllowOriginRegex: []string{orgReg},
		}
	case "":
		if *flags.CorsAllowMethods != "" || *flags.CorsAllowHeaders != "" || *flags.CorsExposeHeaders != "" || *flags.CorsAllowCredentials {
			return nil, fmt.Errorf("cors_preset must be set in order to enable CORS support")
		}
	default:
		return nil, fmt.Errorf(`cors_preset must be either "basic" or "cors_with_regex"`)
	}

	if host.GetCors() != nil {
		host.GetCors().AllowMethods = *flags.CorsAllowMethods
		host.GetCors().AllowHeaders = *flags.CorsAllowHeaders
		host.GetCors().ExposeHeaders = *flags.CorsExposeHeaders
		host.GetCors().AllowCredentials = &types.BoolValue{Value: *flags.CorsAllowCredentials}
	}

	virtualHosts = append(virtualHosts, host)
	return &v2.RouteConfiguration{
		Name:         routeName,
		VirtualHosts: virtualHosts,
	}, nil
}

func makeDynamicRoutingConfig(serviceInfo *sc.ServiceInfo) ([]route.Route, error) {
	var backendRoutes []route.Route
	for _, operation := range serviceInfo.Operations {
		method := serviceInfo.Methods[operation]
		var routeMatcher *route.RouteMatch
		if method.BackendRule.TranslationType == conf.BackendRule_PATH_TRANSLATION_UNSPECIFIED {
			continue
		}
		for _, httpRule := range method.HttpRule {
			if routeMatcher = makeHttpRouteMatcher(httpRule); routeMatcher == nil {
				return nil, fmt.Errorf("error making HTTP route matcher for selector: %v", operation)
			}

			r := route.Route{
				Match: *routeMatcher,
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{
							Cluster: method.BackendRule.ClusterName,
						},
						HostRewriteSpecifier: &route.RouteAction_HostRewrite{
							HostRewrite: method.BackendRule.Hostname,
						},
					},
				},
			}
			backendRoutes = append(backendRoutes, r)
		}
	}
	return backendRoutes, nil
}

func makeHttpRouteMatcher(httpRule *commonpb.Pattern) *route.RouteMatch {
	if httpRule == nil {
		return nil
	}
	var routeMatcher route.RouteMatch
	re := regexp.MustCompile(`{[^{}]+}`)

	// Replacing query parameters inside "{}" by regex "[^\/]+", which means
	// any character except `/`, also adds `$` to match to the end of the string.
	if re.MatchString(httpRule.UriTemplate) {
		routeMatcher = route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Regex{
				Regex: re.ReplaceAllString(httpRule.UriTemplate, `[^\/]+`) + `$`,
			},
		}
	} else {
		// Match with HttpHeader method. Some methods may have same path.
		routeMatcher = route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Path{
				Path: httpRule.UriTemplate,
			},
		}
	}
	routeMatcher.Headers = []*route.HeaderMatcher{
		{
			Name: ":method",
			HeaderMatchSpecifier: &route.HeaderMatcher_ExactMatch{
				ExactMatch: httpRule.HttpMethod,
			},
		},
	}
	return &routeMatcher
}
