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

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/tracing"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	sc "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"

	acpb "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerpb "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	facpb "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	structpb "github.com/golang/protobuf/ptypes/struct"
	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"
)

const (
	statPrefix = "ingress_http"
)

// MakeListeners provides dynamic listeners for Envoy
func MakeListeners(serviceInfo *sc.ServiceInfo) ([]*listenerpb.Listener, error) {
	filterGenerators, err := MakeFilterGenerators(serviceInfo)
	if err != nil {
		return nil, err
	}

	listener, err := MakeListener(serviceInfo, filterGenerators)
	if err != nil {
		return nil, err
	}
	return []*listenerpb.Listener{listener}, nil
}

func addPerRouteConfigGenToMethods(methods []*sc.MethodInfo, filterGen *FilterGenerator) error {
	if filterGen.PerRouteConfigGenFunc == nil {
		return fmt.Errorf("the PerRouteConfigGenFunc of filter %s is empty", filterGen.FilterName)
	}
	for _, method := range methods {
		method.AddPerRouteConfigGen(filterGen.FilterName, filterGen.PerRouteConfigGenFunc)
	}
	return nil

}

// MakeListener provides a dynamic listener for Envoy
func MakeListener(serviceInfo *sc.ServiceInfo, filterGenerators []*FilterGenerator) (*listenerpb.Listener, error) {
	httpFilters := []*hcmpb.HttpFilter{}
	for _, filterGenerator := range filterGenerators {
		filter, perRouteConfigRequiredMethods, err := filterGenerator.FilterGenFunc(serviceInfo)
		if err != nil {
			return nil, fmt.Errorf("fail to create config for the filter %s: %v", filterGenerator.FilterName, err)
		}
		if filter != nil {
			jsonStr, _ := util.ProtoToJson(filter)
			glog.Infof("adding filter config of %s : %v", filterGenerator.FilterName, jsonStr)
			httpFilters = append(httpFilters, filter)

			if len(perRouteConfigRequiredMethods) > 0 {
				if err := addPerRouteConfigGenToMethods(perRouteConfigRequiredMethods, filterGenerator); err != nil {
					return nil, err
				}
			}

		}
	}

	route, err := MakeRouteConfig(serviceInfo)
	if err != nil {
		return nil, fmt.Errorf("makeHttpConnectionManagerRouteConfig got err: %s", err)
	}

	httpConMgr, err := makeHttpConMgr(&serviceInfo.Options, route)
	if err != nil {
		return nil, fmt.Errorf("makeHttpConnectionManager got err: %s", err)
	}

	jsonStr, _ := util.ProtoToJson(httpConMgr)
	glog.Infof("adding Http Connection Manager config: %v", jsonStr)
	httpConMgr.HttpFilters = httpFilters

	// HTTP filter configuration
	httpFilterConfig, err := ptypes.MarshalAny(httpConMgr)
	if err != nil {
		return nil, err
	}

	filterChain := &listenerpb.FilterChain{
		Filters: []*listenerpb.Filter{
			{
				Name:       util.HTTPConnectionManager,
				ConfigType: &listenerpb.Filter_TypedConfig{TypedConfig: httpFilterConfig},
			},
		},
	}

	if serviceInfo.Options.SslServerCertPath != "" {
		transportSocket, err := util.CreateDownstreamTransportSocket(
			serviceInfo.Options.SslServerCertPath,
			serviceInfo.Options.SslMinimumProtocol,
			serviceInfo.Options.SslMaximumProtocol,
			serviceInfo.Options.SslServerCipherSuites,
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
					Address: serviceInfo.Options.ListenerAddress,
					PortSpecifier: &corepb.SocketAddress_PortValue{
						PortValue: uint32(serviceInfo.Options.ListenerPort),
					},
				},
			},
		},
		FilterChains: []*listenerpb.FilterChain{filterChain},
	}

	if serviceInfo.Options.ConnectionBufferLimitBytes >= 0 {
		listener.PerConnectionBufferLimitBytes = &wrapperspb.UInt32Value{
			Value: uint32(serviceInfo.Options.ConnectionBufferLimitBytes),
		}
	}

	return listener, nil
}

func makeHttpConMgr(opts *options.ConfigGeneratorOptions, route *routepb.RouteConfiguration) (*hcmpb.HttpConnectionManager, error) {
	httpConMgr := &hcmpb.HttpConnectionManager{
		UpgradeConfigs: []*hcmpb.HttpConnectionManager_UpgradeConfig{
			{
				UpgradeType: "websocket",
			},
		},
		CodecType:  hcmpb.HttpConnectionManager_AUTO,
		StatPrefix: statPrefix,
		RouteSpecifier: &hcmpb.HttpConnectionManager_RouteConfig{
			RouteConfig: route,
		},
		UseRemoteAddress:  &wrapperspb.BoolValue{Value: opts.EnvoyUseRemoteAddress},
		XffNumTrustedHops: uint32(opts.EnvoyXffNumTrustedHops),
		// Converting the error message for requests rejected by Envoy to JSON format:
		//
		//    {
		//       "code": "http-status-code",
		//       "message": "the error message",
		//    }
		//
		LocalReplyConfig: &hcmpb.LocalReplyConfig{
			BodyFormat: &corepb.SubstitutionFormatString{
				Format: &corepb.SubstitutionFormatString_JsonFormat{
					JsonFormat: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"code": {
								Kind: &structpb.Value_StringValue{StringValue: "%RESPONSE_CODE%"},
							},
							"message": {
								Kind: &structpb.Value_StringValue{StringValue: "%LOCAL_REPLY_BODY%"},
							},
						},
					},
				},
			},
		},
	}

	if opts.AccessLog != "" {
		fileAccessLog := &facpb.FileAccessLog{
			Path: opts.AccessLog,
		}

		if opts.AccessLogFormat != "" {
			fileAccessLog.AccessLogFormat = &facpb.FileAccessLog_LogFormat{
				LogFormat: &corepb.SubstitutionFormatString{
					Format: &corepb.SubstitutionFormatString_TextFormat{
						TextFormat: opts.AccessLogFormat,
					},
				},
			}
		}

		serialized, _ := ptypes.MarshalAny(fileAccessLog)

		httpConMgr.AccessLog = []*acpb.AccessLog{
			{
				Name:   util.AccessFileLogger,
				Filter: nil,
				ConfigType: &acpb.AccessLog_TypedConfig{
					TypedConfig: serialized,
				},
			},
		}
	}

	if !opts.DisableTracing {
		var err error
		httpConMgr.Tracing, err = tracing.CreateTracing(opts.CommonOptions)
		if err != nil {
			return nil, err
		}
	}

	if opts.UnderscoresInHeaders {
		httpConMgr.CommonHttpProtocolOptions = &corepb.HttpProtocolOptions{
			HeadersWithUnderscoresAction: corepb.HttpProtocolOptions_ALLOW,
		}
	} else {
		httpConMgr.CommonHttpProtocolOptions = &corepb.HttpProtocolOptions{
			HeadersWithUnderscoresAction: corepb.HttpProtocolOptions_REJECT_REQUEST,
		}
	}

	if opts.EnableGrpcForHttp1 {
		// Retain gRPC trailers if downstream is using http1.
		httpConMgr.HttpProtocolOptions = &corepb.Http1ProtocolOptions{
			EnableTrailers: true,
		}
	}

	return httpConMgr, nil
}
