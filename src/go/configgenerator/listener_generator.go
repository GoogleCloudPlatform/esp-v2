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
	sc "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/tracing"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	acpb "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerpb "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	facpb "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/golang/glog"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

// MakeListeners provides dynamic listeners for Envoy
func MakeListeners(serviceInfo *sc.ServiceInfo, params filtergen.FactoryParams) ([]*listenerpb.Listener, error) {
	filterGenFactories := GetESPv2FilterGenFactories()

	filterGens, err := NewFilterGeneratorsFromOPConfig(serviceInfo.ServiceConfig(), serviceInfo.Options, filterGenFactories, params)
	if err != nil {
		return nil, err
	}

	listener, err := MakeListener(serviceInfo, filterGens, nil)
	if err != nil {
		return nil, err
	}
	return []*listenerpb.Listener{listener}, nil
}

// MakeHttpFilterConfigs generates all enabled HTTP filter configs and returns them (ordered list).
func MakeHttpFilterConfigs(serviceInfo *sc.ServiceInfo, filterGenerators []filtergen.FilterGenerator) ([]*hcmpb.HttpFilter, error) {
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

// MakeListener provides a dynamic listener for Envoy
func MakeListener(serviceInfo *sc.ServiceInfo, filterGenerators []filtergen.FilterGenerator, localReplyConfig *hcmpb.LocalReplyConfig) (*listenerpb.Listener, error) {
	httpFilters, err := MakeHttpFilterConfigs(serviceInfo, filterGenerators)
	if err != nil {
		return nil, err
	}

	route, err := makeRouteConfig(serviceInfo, filterGenerators)
	if err != nil {
		return nil, fmt.Errorf("makeHttpConnectionManagerRouteConfig got err: %s", err)
	}

	httpConMgr, err := makeHTTPConMgr(&serviceInfo.Options, route, localReplyConfig)
	if err != nil {
		return nil, fmt.Errorf("makeHttpConnectionManager got err: %s", err)
	}
	httpConMgr.SchemeHeaderTransformation = makeSchemeHeaderOverride(serviceInfo)

	jsonStr, _ := util.ProtoToJson(httpConMgr)
	glog.Infof("adding Http Connection Manager config: %v", jsonStr)
	httpConMgr.HttpFilters = httpFilters

	// HTTP filter configuration
	httpFilterConfig, err := anypb.New(httpConMgr)
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
			serviceInfo.Options.SslServerRootCertPath,
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

// To fix b/221072669: a hack to work around b/221308324 where
// Cloud Run always set :scheme header to http when using http2 protocol for grpc.
// Override scheme header to https when following conditions meet:
// * Deployed in serverless platform.
// * Backend uses grpc
// * Any remote backends is using TLS
func makeSchemeHeaderOverride(serviceInfo *sc.ServiceInfo) *corepb.SchemeHeaderTransformation {
	if serviceInfo.Options.ComputePlatformOverride != util.ServerlessPlatform || !serviceInfo.GrpcSupportRequired {
		return nil
	}
	useTLS := false
	for _, v := range serviceInfo.RemoteBackendClusters {
		if v.UseTLS {
			useTLS = true
		}
	}
	if useTLS {
		glog.Infof("add config to override scheme header as https.")
		return &corepb.SchemeHeaderTransformation{
			Transformation: &corepb.SchemeHeaderTransformation_SchemeToOverwrite{
				SchemeToOverwrite: "https",
			},
		}
	}
	return nil
}

func makeHTTPConMgr(opts *options.ConfigGeneratorOptions, route *routepb.RouteConfiguration, localReplyConfig *hcmpb.LocalReplyConfig) (*hcmpb.HttpConnectionManager, error) {
	httpConMgr := &hcmpb.HttpConnectionManager{
		UpgradeConfigs: []*hcmpb.HttpConnectionManager_UpgradeConfig{
			{
				UpgradeType: "websocket",
			},
		},
		CodecType:  hcmpb.HttpConnectionManager_AUTO,
		StatPrefix: util.StatPrefix,
		RouteSpecifier: &hcmpb.HttpConnectionManager_RouteConfig{
			RouteConfig: route,
		},
		UseRemoteAddress:  &wrapperspb.BoolValue{Value: opts.EnvoyUseRemoteAddress},
		XffNumTrustedHops: uint32(opts.EnvoyXffNumTrustedHops),

		// Security options for `path` header.
		NormalizePath: &wrapperspb.BoolValue{Value: opts.NormalizePath},
		MergeSlashes:  opts.MergeSlashesInPath,
	}

	if localReplyConfig != nil {
		httpConMgr.LocalReplyConfig = localReplyConfig
	} else {
		// Converting the error message for requests rejected by Envoy to JSON format:
		//
		//    {
		//       "code": "http-status-code",
		//       "message": "the error message",
		//    }
		//
		httpConMgr.LocalReplyConfig = &hcmpb.LocalReplyConfig{
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
		}
	}

	// https://github.com/envoyproxy/envoy/security/advisories/GHSA-4987-27fx-x6cf
	if opts.DisallowEscapedSlashesInPath {
		httpConMgr.PathWithEscapedSlashesAction = hcmpb.HttpConnectionManager_UNESCAPE_AND_REDIRECT
	} else {
		httpConMgr.PathWithEscapedSlashesAction = hcmpb.HttpConnectionManager_KEEP_UNCHANGED
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

		serialized, _ := anypb.New(fileAccessLog)

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
