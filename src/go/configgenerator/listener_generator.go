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

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/routegen"
	sc "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerpb "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/golang/glog"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

// MakeListeners provides dynamic listeners for Envoy
func MakeListeners(serviceInfo *sc.ServiceInfo, scParams filtergen.ServiceControlOPFactoryParams) ([]*listenerpb.Listener, error) {
	filterGenFactories := MakeHTTPFilterGenFactories(scParams)
	connectionManager, err := filtergen.NewHTTPConnectionManagerGenFromOPConfig(serviceInfo.ServiceConfig(), serviceInfo.Options)
	if err != nil {
		return nil, fmt.Errorf("fail to create HTTP connection manager from OP config: %v", err)
	}

	filterGens, err := NewFilterGeneratorsFromOPConfig(serviceInfo.ServiceConfig(), serviceInfo.Options, filterGenFactories)
	if err != nil {
		return nil, err
	}

	routeGenFactories := MakeRouteGenFactories()
	routeGens, err := routegen.NewRouteGeneratorsFromOPConfig(serviceInfo.ServiceConfig(), serviceInfo.Options, routeGenFactories)
	if err != nil {
		return nil, err
	}

	listener, err := MakeListener(serviceInfo.Options, filterGens, connectionManager, routeGens)
	if err != nil {
		return nil, err
	}
	return []*listenerpb.Listener{listener}, nil
}

// MakeHttpFilterConfigs generates all enabled HTTP filter configs and returns them (ordered list).
func MakeHttpFilterConfigs(filterGenerators []filtergen.FilterGenerator) ([]*hcmpb.HttpFilter, error) {
	var httpFilters []*hcmpb.HttpFilter

	for _, filterGenerator := range filterGenerators {
		filter, err := filterGenerator.GenFilterConfig()
		if err != nil {
			return nil, fmt.Errorf("fail to create config for the filter %q: %v", filterGenerator.FilterName(), err)
		}
		if filter == nil {
			glog.Infof("No filter config generated for %q, potentially because it only has per-route configs.", filterGenerator.FilterName())
			continue
		}

		httpFilter, err := filtergen.FilterConfigToHTTPFilter(filter, filterGenerator.FilterName())
		if err != nil {
			return nil, err
		}

		jsonStr, err := util.ProtoToJson(httpFilter)
		if err != nil {
			return nil, fmt.Errorf("fail to convert proto to JSON for filter %q: %v", filterGenerator.FilterName(), err)
		}

		glog.Infof("adding filter config of %q : %v", filterGenerator.FilterName(), jsonStr)
		httpFilters = append(httpFilters, httpFilter)
	}
	return httpFilters, nil
}

// MakeListener provides a dynamic listener for Envoy.
// Allows dependency injection of FilterGenerator and RouteGenerator for
// internal use.
func MakeListener(opts options.ConfigGeneratorOptions, httpFilterGenerators []filtergen.FilterGenerator, connectionManagerGen filtergen.FilterGenerator, routeGenerators []routegen.RouteGenerator) (*listenerpb.Listener, error) {
	httpFilterConfigs, err := MakeHttpFilterConfigs(httpFilterGenerators)
	if err != nil {
		return nil, err
	}

	routeConfig, err := MakeRouteConfig(opts, httpFilterGenerators, routeGenerators)
	if err != nil {
		return nil, fmt.Errorf("makeHttpConnectionManagerRouteConfig got err: %s", err)
	}

	// HTTP connection manager filter configuration
	hcmConfig, err := connectionManagerGen.GenFilterConfig()
	if err != nil {
		return nil, err
	}

	typedHCMConfig, ok := hcmConfig.(*hcmpb.HttpConnectionManager)
	if !ok {
		return nil, fmt.Errorf("HCM generator returned proto config of type %T, want HCM config", hcmConfig)
	}

	typedHCMConfig.HttpFilters = httpFilterConfigs
	typedHCMConfig.RouteSpecifier = &hcmpb.HttpConnectionManager_RouteConfig{
		RouteConfig: routeConfig,
	}

	networkFilterConfig, err := filtergen.FilterConfigToNetworkFilter(typedHCMConfig, filtergen.HTTPConnectionManagerFilterName)
	if err != nil {
		return nil, err
	}

	filterChain := &listenerpb.FilterChain{
		Filters: []*listenerpb.Filter{
			networkFilterConfig,
		},
	}

	if opts.SslServerCertPath != "" {
		transportSocket, err := util.CreateDownstreamTransportSocket(
			opts.SslServerCertPath,
			opts.SslServerRootCertPath,
			opts.SslMinimumProtocol,
			opts.SslMaximumProtocol,
			opts.SslServerCipherSuites,
		)
		if err != nil {
			return nil, err
		}
		filterChain.TransportSocket = transportSocket
	}

	listener := &listenerpb.Listener{
		Name: util.IngressListenerName,
		Address: &corepb.Address{
			Address: &corepb.Address_SocketAddress{
				SocketAddress: &corepb.SocketAddress{
					Address: opts.ListenerAddress,
					PortSpecifier: &corepb.SocketAddress_PortValue{
						PortValue: uint32(opts.ListenerPort),
					},
				},
			},
		},
		FilterChains: []*listenerpb.FilterChain{filterChain},
	}

	if opts.ConnectionBufferLimitBytes >= 0 {
		listener.PerConnectionBufferLimitBytes = &wrapperspb.UInt32Value{
			Value: uint32(opts.ConnectionBufferLimitBytes),
		}
	}

	return listener, nil
}
