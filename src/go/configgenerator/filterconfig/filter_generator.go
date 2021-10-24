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

package filterconfig

import (
	"fmt"
	"sort"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	ci "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	commonpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v10/http/common"

	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	transcoderpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_json_transcoder/v3"
	hcpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/health_check/v3"
	routerpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

// A wrapper of filter generation logic.
type FilterGenerator struct {
	// The filter name.
	FilterName string
	// The function to generate filter config and the methods requiring per route configs.
	FilterGenFunc
	// The function to generate per route config.
	// It should be set if the filter needs to set per route config.
	ci.PerRouteConfigGenFunc
}

// The function type to generate filter config.
// Return
//  - the filter config
//  - the methods needed to add per route config
//  - the error
type FilterGenFunc func(sc *ci.ServiceInfo) (*hcmpb.HttpFilter, []*ci.MethodInfo, error)

// MakeFilterGenerators provide of a slice of FilterGenerator in sequence.
func MakeFilterGenerators(serviceInfo *ci.ServiceInfo) ([]*FilterGenerator, error) {
	filterGenerators := []*FilterGenerator{}

	if serviceInfo.Options.CorsPreset == "basic" || serviceInfo.Options.CorsPreset == "cors_with_regex" {
		filterGenerators = append(filterGenerators, &FilterGenerator{
			FilterName: util.CORS,
			FilterGenFunc: func(sc *ci.ServiceInfo) (*hcmpb.HttpFilter, []*ci.MethodInfo, error) {
				corsFilter := &hcmpb.HttpFilter{
					Name: util.CORS,
				}
				return corsFilter, nil, nil
			},
		})
	}

	// Add Health Check filter if needed. It must behind Path Matcher filter, since Service Control
	// filter needs to get the corresponding rule for health check calls, in order to skip Report
	if serviceInfo.Options.Healthz != "" {
		filterGenerators = append(filterGenerators, &FilterGenerator{
			FilterName: util.HealthCheck,
			FilterGenFunc: func(sc *ci.ServiceInfo) (*hcmpb.HttpFilter, []*ci.MethodInfo, error) {
				hcFilter, err := makeHealthCheckFilter(serviceInfo)
				if err != nil {
					return nil, nil, err
				}
				return hcFilter, nil, nil
			},
		})
	}

	// Add JWT Authn filter if needed.
	if !serviceInfo.Options.SkipJwtAuthnFilter {
		// TODO(b/176432170): Handle errors here, prevent startup.
		filterGenerators = append(filterGenerators, &FilterGenerator{
			FilterName:            util.JwtAuthn,
			FilterGenFunc:         jaFilterGenFunc,
			PerRouteConfigGenFunc: jaPerRouteFilterConfigGen,
		})
	}

	// Add Service Control filter if needed.
	if !serviceInfo.Options.SkipServiceControlFilter {
		filterGenerators = append(filterGenerators, &FilterGenerator{
			FilterName:            util.ServiceControl,
			FilterGenFunc:         scFilterGenFunc,
			PerRouteConfigGenFunc: scPerRouteFilterConfigGen,
		})
	}

	// Add gRPC Transcoder filter and gRPCWeb filter configs for gRPC backend.
	if serviceInfo.GrpcSupportRequired {
		// grpc-web filter should be before grpc transcoder filter.
		// It converts content-type application/grpc-web to application/grpc and
		// grpc transcoder will bypass requests with application/grpc content type.
		// Otherwise grpc transcoder will try to transcode a grpc-web request which
		// will fail.
		filterGenerators = append(filterGenerators, &FilterGenerator{
			FilterName: util.GRPCWeb,
			FilterGenFunc: func(sc *ci.ServiceInfo) (*hcmpb.HttpFilter, []*ci.MethodInfo, error) {
				return &hcmpb.HttpFilter{
					Name: util.GRPCWeb,
				}, nil, nil
			},
		})

		filterGenerators = append(filterGenerators, &FilterGenerator{
			FilterName: util.GRPCJSONTranscoder,
			FilterGenFunc: func(sc *ci.ServiceInfo) (*hcmpb.HttpFilter, []*ci.MethodInfo, error) {
				return makeTranscoderFilter(serviceInfo), nil, nil
			},
		})
	}

	filterGenerators = append(filterGenerators, &FilterGenerator{
		FilterName:            util.BackendAuth,
		FilterGenFunc:         baFilterGenFunc,
		PerRouteConfigGenFunc: baPerRouteFilterConfigGen,
	})

	filterGenerators = append(filterGenerators, &FilterGenerator{
		FilterName:            util.PathRewrite,
		FilterGenFunc:         prFilterGenFunc,
		PerRouteConfigGenFunc: prPerRouteFilterConfigGen,
	})

	if serviceInfo.Options.EnableGrpcForHttp1 {
		// Add GrpcMetadataScrubber filter to retain gRPC trailers

		filterGenerators = append(filterGenerators, &FilterGenerator{
			FilterName: util.GrpcMetadataScrubber,
			FilterGenFunc: func(sc *ci.ServiceInfo) (*hcmpb.HttpFilter, []*ci.MethodInfo, error) {
				return &hcmpb.HttpFilter{
					Name: util.GrpcMetadataScrubber,
				}, nil, nil
			},
		})
	}

	// Add Envoy Router filter so requests are routed upstream.
	// Router filter should be the last.
	filterGenerators = append(filterGenerators, &FilterGenerator{
		FilterName: util.Router,
		FilterGenFunc: func(sc *ci.ServiceInfo) (*hcmpb.HttpFilter, []*ci.MethodInfo, error) {
			return makeRouterFilter(serviceInfo.Options), nil, nil
		},
	})
	return filterGenerators, nil
}

func makeTranscoderFilter(serviceInfo *ci.ServiceInfo) *hcmpb.HttpFilter {
	for _, sourceFile := range serviceInfo.ServiceConfig().GetSourceInfo().GetSourceFiles() {
		configFile := &smpb.ConfigFile{}
		ptypes.UnmarshalAny(sourceFile, configFile)

		if configFile.GetFileType() == smpb.ConfigFile_FILE_DESCRIPTOR_SET_PROTO {
			ignoredQueryParameterList := []string{}
			for IgnoredQueryParameter := range serviceInfo.AllTranscodingIgnoredQueryParams {
				ignoredQueryParameterList = append(ignoredQueryParameterList, IgnoredQueryParameter)

			}
			sort.Sort(sort.StringSlice(ignoredQueryParameterList))

			configContent := configFile.GetFileContents()
			transcodeConfig := &transcoderpb.GrpcJsonTranscoder{
				DescriptorSet: &transcoderpb.GrpcJsonTranscoder_ProtoDescriptorBin{
					ProtoDescriptorBin: configContent,
				},
				AutoMapping:                  true,
				ConvertGrpcStatus:            true,
				IgnoredQueryParameters:       ignoredQueryParameterList,
				IgnoreUnknownQueryParameters: serviceInfo.Options.TranscodingIgnoreUnknownQueryParameters,
				QueryParamUnescapePlus:       !serviceInfo.Options.TranscodingQueryParametersDisableUnescapePlus,
				PrintOptions: &transcoderpb.GrpcJsonTranscoder_PrintOptions{
					AlwaysPrintPrimitiveFields: serviceInfo.Options.TranscodingAlwaysPrintPrimitiveFields,
					AlwaysPrintEnumsAsInts:     serviceInfo.Options.TranscodingAlwaysPrintEnumsAsInts,
					PreserveProtoFieldNames:    serviceInfo.Options.TranscodingPreserveProtoFieldNames,
				},
			}
			if serviceInfo.Options.TranscodingFilePath != "" {
				transcodeConfig.DescriptorSet = &transcoderpb.GrpcJsonTranscoder_ProtoDescriptor{
					ProtoDescriptor: serviceInfo.Options.TranscodingFilePath,
				}
			}

			transcodeConfig.Services = append(transcodeConfig.Services, serviceInfo.ApiNames...)

			transcodeConfigStruct, _ := ptypes.MarshalAny(transcodeConfig)
			transcodeFilter := &hcmpb.HttpFilter{
				Name:       util.GRPCJSONTranscoder,
				ConfigType: &hcmpb.HttpFilter_TypedConfig{transcodeConfigStruct},
			}
			return transcodeFilter
		}
	}

	// b/148605552: Previous versions of the `gcloud_build_image` script did not download the proto descriptor.
	// We cannot ensure that users have the latest version of the script, so notify them via non-fatal logs.
	// Log as error instead of warning because error logs will show up even if `--enable_debug` is false.
	glog.Error("Unable to setup gRPC-JSON transcoding because no proto descriptor was found in the service config. " +
		"Please use version 2020-01-29 (or later) of the `gcloud_build_image` script. " +
		"https://github.com/GoogleCloudPlatform/esp-v2/blob/master/docker/serverless/gcloud_build_image")
	return nil
}

func makeHealthCheckFilter(serviceInfo *ci.ServiceInfo) (*hcmpb.HttpFilter, error) {
	hcFilterConfig := &hcpb.HealthCheck{
		PassThroughMode: &wrapperspb.BoolValue{Value: false},

		Headers: []*routepb.HeaderMatcher{
			{
				Name: ":path",
				HeaderMatchSpecifier: &routepb.HeaderMatcher_StringMatch{
					StringMatch: &matcher.StringMatcher{
						MatchPattern: &matcher.StringMatcher_Exact{
							Exact: serviceInfo.Options.Healthz,
						},
					},
				},
			},
		},
	}
	hcFilterConfigStruc, err := ptypes.MarshalAny(hcFilterConfig)
	if err != nil {
		return nil, err
	}
	return &hcmpb.HttpFilter{
		Name:       util.HealthCheck,
		ConfigType: &hcmpb.HttpFilter_TypedConfig{hcFilterConfigStruc},
	}, nil
}

func makeRouterFilter(opts options.ConfigGeneratorOptions) *hcmpb.HttpFilter {
	router, _ := ptypes.MarshalAny(&routerpb.Router{
		SuppressEnvoyHeaders: opts.SuppressEnvoyHeaders,
		StartChildSpan:       !opts.DisableTracing,
	})

	routerFilter := &hcmpb.HttpFilter{
		Name:       util.Router,
		ConfigType: &hcmpb.HttpFilter_TypedConfig{TypedConfig: router},
	}
	return routerFilter
}

func parseDepErrorBehavior(stringVal string) (commonpb.DependencyErrorBehavior, error) {
	depErrorBehaviorInt, ok := commonpb.DependencyErrorBehavior_value[stringVal]
	if !ok {
		keys := make([]string, 0, len(commonpb.DependencyErrorBehavior_value))
		for k := range commonpb.DependencyErrorBehavior_value {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return commonpb.DependencyErrorBehavior_UNSPECIFIED, fmt.Errorf("unknown value for DependencyErrorBehavior (%v), accepted values are: %+q", stringVal, keys)
	}
	return commonpb.DependencyErrorBehavior(depErrorBehaviorInt), nil
}
