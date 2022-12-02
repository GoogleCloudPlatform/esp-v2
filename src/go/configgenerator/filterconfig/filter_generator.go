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
	commonpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v11/http/common"
	gmspb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v11/http/grpc_metadata_scrubber"

	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	corspb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/cors/v3"
	transcoderpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_json_transcoder/v3"
	grpcwebpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_web/v3"
	hcpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/health_check/v3"
	routerpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	envoytypepb "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	descpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"

	ahpb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
	"google.golang.org/protobuf/proto"
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
//   - the filter config
//   - the methods needed to add per route config
//   - the error
type FilterGenFunc func(sc *ci.ServiceInfo) (*hcmpb.HttpFilter, []*ci.MethodInfo, error)

// MakeFilterGenerators provide of a slice of FilterGenerator in sequence.
func MakeFilterGenerators(serviceInfo *ci.ServiceInfo) ([]*FilterGenerator, error) {
	filterGenerators := []*FilterGenerator{}

	if serviceInfo.Options.CorsPreset == "basic" || serviceInfo.Options.CorsPreset == "cors_with_regex" {
		filterGenerators = append(filterGenerators, &FilterGenerator{
			FilterName: util.CORS,
			FilterGenFunc: func(sc *ci.ServiceInfo) (*hcmpb.HttpFilter, []*ci.MethodInfo, error) {
				a, err := ptypes.MarshalAny(&corspb.Cors{})
				if err != nil {
					return nil, nil, err
				}
				corsFilter := &hcmpb.HttpFilter{
					Name:       util.CORS,
					ConfigType: &hcmpb.HttpFilter_TypedConfig{TypedConfig: a},
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
	if serviceInfo.Options.EnableResponseCompression {
		filterGenerators = append(filterGenerators, &FilterGenerator{
			FilterName:    util.EnvoyGzipCompressor,
			FilterGenFunc: gzipCompressorGenFunc,
		})
		filterGenerators = append(filterGenerators, &FilterGenerator{
			FilterName:    util.EnvoyBrotliCompressor,
			FilterGenFunc: brotliCompressorGenFunc,
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
				a, err := ptypes.MarshalAny(&grpcwebpb.GrpcWeb{})
				if err != nil {
					return nil, nil, err
				}
				return &hcmpb.HttpFilter{
					Name:       util.GRPCWeb,
					ConfigType: &hcmpb.HttpFilter_TypedConfig{TypedConfig: a},
				}, nil, nil
			},
		})

		filterGenerators = append(filterGenerators, &FilterGenerator{
			FilterName: util.GRPCJSONTranscoder,
			FilterGenFunc: func(sc *ci.ServiceInfo) (*hcmpb.HttpFilter, []*ci.MethodInfo, error) {
				filter, err := makeTranscoderFilter(serviceInfo)
				if err != nil {
					return nil, nil, err
				}
				return filter, nil, nil
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
				a, err := ptypes.MarshalAny(&gmspb.FilterConfig{})
				if err != nil {
					return nil, nil, err
				}
				return &hcmpb.HttpFilter{
					Name:       util.GrpcMetadataScrubber,
					ConfigType: &hcmpb.HttpFilter_TypedConfig{TypedConfig: a},
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

func updateProtoDescriptor(service *confpb.Service, apiNames []string, descriptorBytes []byte) ([]byte, error) {
	// To support specifying custom http rules in service config.
	// Envoy grpc_json_transcoder only uses the http.rules in the proto descriptor
	// generated from "google.api.http" annotation in the proto file.
	// For some shared grpc services, each service may want to define its own
	// http.rules mapping. This function will copy the http.rules from the service config
	// into proto descriptor.
	//
	// api-compiler has following behaviours:
	// * If a "google.api.http" annotation is specified in a method in the proto,
	//   and the service config yaml doesn't specify one, api-compiler will copy it out
	//   to the normalized service config.
	// * If a http.rule is specified in the service config, it will overwrite
	//   the one from the proto annotation.
	//
	// So it should be ok to blindly copy the http.rules from the service config to
	// proto descriptor.
	ruleMap := make(map[string]*ahpb.HttpRule)
	for _, rule := range service.GetHttp().GetRules() {
		ruleMap[rule.GetSelector()] = rule
	}
	apiMap := make(map[string]bool)
	for _, apiName := range apiNames {
		apiMap[apiName] = true
	}

	fds := &descpb.FileDescriptorSet{}
	if err := proto.Unmarshal(descriptorBytes, fds); err != nil {
		glog.Error("failed to unmarshal protodescriptor, error: ", err)
		return nil, fmt.Errorf("failed to unmarshal proto descriptor, error: %v", err)
	}

	for _, file := range fds.GetFile() {
		for _, service := range file.GetService() {
			apiName := fmt.Sprintf("%s.%s", file.GetPackage(), service.GetName())

			// Only modify the API in the serviceInfo.ApiNames.
			// These are the ones to enable grpc transcoding.
			if _, ok := apiMap[apiName]; !ok {
				continue
			}

			for _, method := range service.GetMethod() {
				sel := fmt.Sprintf("%s.%s", apiName, method.GetName())
				if rule, ok := ruleMap[sel]; ok {
					json, _ := util.ProtoToJson(rule)
					glog.Info("Set http.rule: ", json)
					if method.GetOptions() == nil {
						method.Options = &descpb.MethodOptions{}
					}
					proto.SetExtension(method.GetOptions(), ahpb.E_Http, rule)
				}

				// If an http rule is specified for a rpc endpoint then the rpc's default http binding will be
				// disabled according to the logic in the envoy's json transcoder filter. To still enable
				// the default http binding, which is the designed behavior, the default http binding needs to be
				// added to the http rule's additional bindings.
				if httpRule := proto.GetExtension(method.GetOptions(), ahpb.E_Http).(*ahpb.HttpRule); httpRule != nil {
					defaultPath := fmt.Sprintf("/%s/%s", apiName, method.GetName())
					preserveDefaultHttpBinding(httpRule, defaultPath)
				}
			}
		}
	}

	newData, err := proto.Marshal(fds)
	if err != nil {
		glog.Error("failed to marshal proto descriptor, error: ", err)
		return nil, fmt.Errorf("failed to marshal proto descriptor, error: %v", err)
	}
	return newData, nil
}

func preserveDefaultHttpBinding(httpRule *ahpb.HttpRule, defaultPath string) {
	defaultBinding := &ahpb.HttpRule{Pattern: &ahpb.HttpRule_Post{defaultPath}, Body: "*"}

	// Check existence of the default binding in httpRule's additional_bindings to avoid duplication.
	for _, addtionalBinding := range httpRule.AdditionalBindings {
		if proto.Equal(addtionalBinding, defaultBinding) {
			return
		}
	}
	// check if httpRule is the same as default binding, ignore the difference in fields selector and
	// additional_bindings.
	defaultBinding.Selector = httpRule.GetSelector()
	defaultBinding.AdditionalBindings = httpRule.GetAdditionalBindings()
	if proto.Equal(httpRule, defaultBinding) {
		return
	}
	defaultBinding.Selector = ""
	defaultBinding.AdditionalBindings = nil

	httpRule.AdditionalBindings = append(httpRule.AdditionalBindings, defaultBinding)
}

func makeTranscoderFilter(serviceInfo *ci.ServiceInfo) (*hcmpb.HttpFilter, error) {
	for _, sourceFile := range serviceInfo.ServiceConfig().GetSourceInfo().GetSourceFiles() {
		configFile := &smpb.ConfigFile{}
		ptypes.UnmarshalAny(sourceFile, configFile)

		if configFile.GetFileType() == smpb.ConfigFile_FILE_DESCRIPTOR_SET_PROTO {
			ignoredQueryParameterList := []string{}
			for IgnoredQueryParameter := range serviceInfo.AllTranscodingIgnoredQueryParams {
				ignoredQueryParameterList = append(ignoredQueryParameterList, IgnoredQueryParameter)

			}
			sort.Sort(sort.StringSlice(ignoredQueryParameterList))

			configContent, err := updateProtoDescriptor(serviceInfo.ServiceConfig(), serviceInfo.ApiNames,
				configFile.GetFileContents())
			if err != nil {
				return nil, err
			}

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
					StreamNewlineDelimited:     serviceInfo.Options.TranscodingStreamNewLineDelimited,
				},
				MatchUnregisteredCustomVerb: serviceInfo.Options.TranscodingMatchUnregisteredCustomVerb,
				CaseInsensitiveEnumParsing:  serviceInfo.Options.TranscodingCaseInsensitiveEnumParsing,
			}
			if serviceInfo.Options.TranscodingStrictRequestValidation {
				transcodeConfig.RequestValidationOptions = &transcoderpb.GrpcJsonTranscoder_RequestValidationOptions{
					RejectUnknownMethod:              true,
					RejectUnknownQueryParameters:     true,
					RejectBindingBodyFieldCollisions: serviceInfo.Options.TranscodingRejectCollision,
				}
			}

			transcodeConfig.Services = append(transcodeConfig.Services, serviceInfo.ApiNames...)

			transcodeConfigStruct, _ := ptypes.MarshalAny(transcodeConfig)
			transcodeFilter := &hcmpb.HttpFilter{
				Name:       util.GRPCJSONTranscoder,
				ConfigType: &hcmpb.HttpFilter_TypedConfig{transcodeConfigStruct},
			}
			return transcodeFilter, nil
		}
	}

	// b/148605552: Previous versions of the `gcloud_build_image` script did not download the proto descriptor.
	// We cannot ensure that users have the latest version of the script, so notify them via non-fatal logs.
	// Log as error instead of warning because error logs will show up even if `--enable_debug` is false.
	glog.Error("Unable to setup gRPC-JSON transcoding because no proto descriptor was found in the service config. " +
		"Please use version 2020-01-29 (or later) of the `gcloud_build_image` script. " +
		"https://github.com/GoogleCloudPlatform/esp-v2/blob/master/docker/serverless/gcloud_build_image")
	return nil, nil
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

	if serviceInfo.Options.HealthCheckGrpcBackend {
		hcFilterConfig.ClusterMinHealthyPercentages = map[string]*envoytypepb.Percent{
			serviceInfo.LocalBackendCluster.ClusterName: &envoytypepb.Percent{Value: 100.0},
		}
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
