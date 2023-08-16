// Copyright 2023 Google LLC
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

package filtergen

import (
	"fmt"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/tracing"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	acpb "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	facpb "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/golang/glog"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	// HTTPConnectionManagerFilterName is the Envoy filter name for debug logging.
	HTTPConnectionManagerFilterName = "envoy.filters.network.http_connection_manager"
)

type HTTPConnectionManagerGenerator struct {
	IsSchemeHeaderOverrideRequired bool

	// ESPv2 options
	EnvoyUseRemoteAddress        bool
	EnvoyXffNumTrustedHops       int
	NormalizePath                bool
	MergeSlashesInPath           bool
	DisallowEscapedSlashesInPath bool
	AccessLogPath                string
	AccessLogFormat              string
	UnderscoresInHeaders         bool
	EnableGrpcForHttp1           bool
	TracingOptions               *options.TracingOptions

	NoopFilterGenerator
}

// NewHTTPConnectionManagerGenFromOPConfig creates a HTTPConnectionManagerGenerator from
// OP service config + descriptor + ESPv2 options.
//
// This is a special case generator as it a network filter, not HTTP filter.
func NewHTTPConnectionManagerGenFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) (*HTTPConnectionManagerGenerator, error) {
	isSchemeHeaderOverrideRequired, err := IsSchemeHeaderOverrideRequiredForOPConfig(serviceConfig, opts)
	if err != nil {
		return nil, err
	}

	return &HTTPConnectionManagerGenerator{
		IsSchemeHeaderOverrideRequired: isSchemeHeaderOverrideRequired,
		EnvoyUseRemoteAddress:          opts.EnvoyUseRemoteAddress,
		EnvoyXffNumTrustedHops:         opts.EnvoyXffNumTrustedHops,
		NormalizePath:                  opts.NormalizePath,
		MergeSlashesInPath:             opts.MergeSlashesInPath,
		DisallowEscapedSlashesInPath:   opts.DisallowEscapedSlashesInPath,
		AccessLogPath:                  opts.AccessLog,
		AccessLogFormat:                opts.AccessLogFormat,
		UnderscoresInHeaders:           opts.UnderscoresInHeaders,
		EnableGrpcForHttp1:             opts.EnableGrpcForHttp1,
		TracingOptions:                 opts.TracingOptions,
	}, nil
}

func (g *HTTPConnectionManagerGenerator) FilterName() string {
	return HTTPConnectionManagerFilterName
}

func (g *HTTPConnectionManagerGenerator) GenFilterConfig() (proto.Message, error) {
	httpConMgr := &hcmpb.HttpConnectionManager{
		UpgradeConfigs: []*hcmpb.HttpConnectionManager_UpgradeConfig{
			{
				UpgradeType: "websocket",
			},
		},
		CodecType:         hcmpb.HttpConnectionManager_AUTO,
		StatPrefix:        util.StatPrefix,
		UseRemoteAddress:  &wrapperspb.BoolValue{Value: g.EnvoyUseRemoteAddress},
		XffNumTrustedHops: uint32(g.EnvoyXffNumTrustedHops),

		// Security options for `path` header.
		NormalizePath: &wrapperspb.BoolValue{Value: g.NormalizePath},
		MergeSlashes:  g.MergeSlashesInPath,
	}

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

	// https://github.com/envoyproxy/envoy/security/advisories/GHSA-4987-27fx-x6cf
	if g.DisallowEscapedSlashesInPath {
		httpConMgr.PathWithEscapedSlashesAction = hcmpb.HttpConnectionManager_UNESCAPE_AND_REDIRECT
	} else {
		httpConMgr.PathWithEscapedSlashesAction = hcmpb.HttpConnectionManager_KEEP_UNCHANGED
	}

	if g.AccessLogPath != "" {
		fileAccessLog := &facpb.FileAccessLog{
			Path: g.AccessLogPath,
		}

		if g.AccessLogFormat != "" {
			fileAccessLog.AccessLogFormat = &facpb.FileAccessLog_LogFormat{
				LogFormat: &corepb.SubstitutionFormatString{
					Format: &corepb.SubstitutionFormatString_TextFormat{
						TextFormat: g.AccessLogFormat,
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

	if !g.TracingOptions.DisableTracing {
		var err error
		httpConMgr.Tracing, err = tracing.CreateTracing(*g.TracingOptions)
		if err != nil {
			return nil, err
		}
	}

	if g.UnderscoresInHeaders {
		httpConMgr.CommonHttpProtocolOptions = &corepb.HttpProtocolOptions{
			HeadersWithUnderscoresAction: corepb.HttpProtocolOptions_ALLOW,
		}
	} else {
		httpConMgr.CommonHttpProtocolOptions = &corepb.HttpProtocolOptions{
			HeadersWithUnderscoresAction: corepb.HttpProtocolOptions_REJECT_REQUEST,
		}
	}

	if g.EnableGrpcForHttp1 {
		// Retain gRPC trailers if downstream is using http1.
		httpConMgr.HttpProtocolOptions = &corepb.Http1ProtocolOptions{
			EnableTrailers: true,
		}
	}

	if g.IsSchemeHeaderOverrideRequired {
		httpConMgr.SchemeHeaderTransformation = &corepb.SchemeHeaderTransformation{
			Transformation: &corepb.SchemeHeaderTransformation_SchemeToOverwrite{
				SchemeToOverwrite: "https",
			},
		}
	}

	jsonStr, _ := util.ProtoToJson(httpConMgr)
	glog.Infof("HTTP Connection Manager config before adding routes or HTTP filters: %v", jsonStr)

	return httpConMgr, nil
}

// IsSchemeHeaderOverrideRequiredForOPConfig fixes b/221072669:
// a hack to work around b/221308324 where
// Cloud Run always set :scheme header to http when using http2 protocol for grpc.
// Override scheme header to https when following conditions meet:
// * Deployed in serverless platform.
// * Backend uses grpc
// * Any remote backends is using TLS
func IsSchemeHeaderOverrideRequiredForOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) (bool, error) {
	if opts.ComputePlatformOverride != util.ServerlessPlatform {
		glog.Infof("Skip HTTP Conn Manager scheme override because platform is NOT Cloud Run.")
		return false, nil
	}
	isGRPCSupportRequired, err := IsGRPCSupportRequiredForOPConfig(serviceConfig, opts)
	if err != nil {
		return false, err
	}
	if !isGRPCSupportRequired {
		glog.Infof("Skip HTTP Conn Manager scheme override because there is no gRPC backend.")
		return false, nil
	}
	if opts.EnableBackendAddressOverride {
		glog.Warningf("Skip HTTP Conn Manager scheme override because backend address override is enabled.")
		return false, nil
	}

	for _, rule := range serviceConfig.GetBackend().GetRules() {
		if util.ShouldSkipOPDiscoveryAPI(rule.GetSelector(), opts.AllowDiscoveryAPIs) {
			glog.Warningf("Skip backend rule %q because discovery API is not supported.", rule.GetSelector())
			continue
		}

		if rule.GetAddress() == "" {
			glog.Infof("Skip backend rule %q because it does not have dynamic routing address.", rule.GetSelector())
			return false, nil
		}

		scheme, _, _, _, err := util.ParseURI(rule.GetAddress())
		if err != nil {
			return false, fmt.Errorf("error parsing remote backend rule's address for operation %q, %v", rule.GetSelector(), err)
		}

		// Create a cluster for the remote backend.
		_, useTLS, err := util.ParseBackendProtocol(scheme, rule.GetProtocol())
		if err != nil {
			return false, fmt.Errorf("error parsing remote backend rule's protocol for operation %q, %v", rule.GetSelector(), err)
		}

		if useTLS {
			glog.Infof("add config to override scheme header as https.")
			return true, nil
		}
	}

	return false, nil
}
