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

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/routegen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/glog"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	routeName       = "local_route"
	virtualHostName = "backend"
)

// MakeRouteGenFactories creates the route generator factories (in order).
func MakeRouteGenFactories() []routegen.RouteGeneratorOPFactory {
	return []routegen.RouteGeneratorOPFactory{
		routegen.NewProxyBackendRouteGenFromOPConfig,
		routegen.NewProxyCORSRouteGenFromOPConfig,
		routegen.NewDirectResponseHealthCheckRouteGenFromOPConfig,
		routegen.NewDirectResponseCORSRouteGenFromOPConfig,
		func(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) (routegen.RouteGenerator, error) {
			return routegen.NewDenyInvalidMethodRouteGenFromOPConfig(serviceConfig, opts, []routegen.RouteGeneratorOPFactory{
				routegen.NewProxyBackendRouteGenFromOPConfig,
				routegen.NewDirectResponseHealthCheckRouteGenFromOPConfig,
			})
		},
		routegen.NewDenyAllRouteGenFromOPConfig,
	}
}

// MakeRouteConfig creates the virtual host and route table with the default
// route generators for ESPv2.
func MakeRouteConfig(serviceInfo *configinfo.ServiceInfo, filterGenerators []filtergen.FilterGenerator) (*routepb.RouteConfiguration, error) {
	routeGenFactories := MakeRouteGenFactories()

	routeGens, err := routegen.NewRouteGeneratorsFromOPConfig(serviceInfo.ServiceConfig(), serviceInfo.Options, routeGenFactories)
	if err != nil {
		return nil, err
	}

	return MakeRouteConfigWithGens(serviceInfo.Options, filterGenerators, routeGens)
}

// MakeRouteConfigWithGens is a version of MakeRouteConfig that allows injection
// of different route generators that are not the default.
//
// Useful when extending the config generator internally.
func MakeRouteConfigWithGens(opts options.ConfigGeneratorOptions, filterGenerators []filtergen.FilterGenerator, routeGenerators []routegen.RouteGenerator) (*routepb.RouteConfiguration, error) {
	host := &routepb.VirtualHost{
		Name:    virtualHostName,
		Domains: []string{"*"},
	}

	perHostConfig, err := makePerVHostFilterConfig(host.Name, filterGenerators)
	if err != nil {
		return nil, fmt.Errorf("fail to make per-vHost filter config for virtual host %q: %v", host.Name, err)
	}
	host.TypedPerFilterConfig = perHostConfig

	backendRoutes, err := makeRouteTable(filterGenerators, routeGenerators)
	if err != nil {
		return nil, err
	}
	host.Routes = backendRoutes

	requestHeaders, err := makeRequestHeadersToAdd(opts)
	if err != nil {
		return nil, err
	}
	responseHeaders, err := makeResponseHeadersToAdd(opts)
	if err != nil {
		return nil, err
	}

	return &routepb.RouteConfiguration{
		Name: routeName,
		VirtualHosts: []*routepb.VirtualHost{
			host,
		},
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

func makeRequestHeadersToAdd(opts options.ConfigGeneratorOptions) ([]*corepb.HeaderValueOption, error) {
	l, err := makeHeaders(opts.AddRequestHeaders, false)
	if err != nil {
		return l, err
	}

	m, err := makeHeaders(opts.AppendRequestHeaders, true)
	if err != nil {
		return l, err
	}

	l = append(l, m...)
	return l, nil
}

func makeResponseHeadersToAdd(opts options.ConfigGeneratorOptions) ([]*corepb.HeaderValueOption, error) {
	l, err := makeHeaders(opts.AddResponseHeaders, false)
	if err != nil {
		return l, err
	}

	m, err := makeHeaders(opts.AppendResponseHeaders, true)
	if err != nil {
		return l, err
	}

	l = append(l, m...)
	return l, nil
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

// makeRouteTable generates all routes for the service.
func makeRouteTable(filterGens []filtergen.FilterGenerator, routeGens []routegen.RouteGenerator) ([]*routepb.Route, error) {
	var allRoutes []*routepb.Route

	for _, routeGen := range routeGens {
		routes, err := routeGen.GenRouteConfig(filterGens)
		if err != nil {
			return nil, fmt.Errorf("fail to create config for the route type %q: %v", routeGen.RouteType(), err)
		}

		wrapper := &routepb.VirtualHost{
			Routes: routes,
		}
		jsonStr, err := util.ProtoToJson(wrapper)
		if err != nil {
			return nil, fmt.Errorf("fail to convert proto to JSON for route type %q: %v", routeGen.RouteType(), err)
		}

		glog.Infof("adding routes of type %q to route table : %v", routeGen.RouteType(), jsonStr)
		allRoutes = append(allRoutes, routes...)
	}
	return allRoutes, nil
}
