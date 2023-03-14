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
	"sort"

	ci "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	transcoderpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_json_transcoder/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	ahpb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
	"google.golang.org/protobuf/proto"
	descpb "google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
)

type GRPCTranscoderGenerator struct{}

func (g *GRPCTranscoderGenerator) FilterName() string {
	return util.GRPCJSONTranscoder
}

func (g *GRPCTranscoderGenerator) GenFilterConfig(serviceInfo *ci.ServiceInfo) (*hcmpb.HttpFilter, []*ci.MethodInfo, error) {
	if serviceInfo.Options.LocalHTTPBackendAddress != "" {
		glog.Warningf("Test-only http backend address is set to %q; skip transcoder filter completely.", serviceInfo.Options.LocalHTTPBackendAddress)
		return nil, nil, nil
	}
	for _, sourceFile := range serviceInfo.ServiceConfig().GetSourceInfo().GetSourceFiles() {
		configFile := &smpb.ConfigFile{}
		ptypes.UnmarshalAny(sourceFile, configFile)

		if configFile.GetFileType() == smpb.ConfigFile_FILE_DESCRIPTOR_SET_PROTO {
			var ignoredQueryParameterList []string
			for IgnoredQueryParameter := range serviceInfo.AllTranscodingIgnoredQueryParams {
				ignoredQueryParameterList = append(ignoredQueryParameterList, IgnoredQueryParameter)

			}
			sort.Sort(sort.StringSlice(ignoredQueryParameterList))

			configContent, err := updateProtoDescriptor(serviceInfo.ServiceConfig(), serviceInfo.ApiNames,
				configFile.GetFileContents())
			if err != nil {
				return nil, nil, err
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
				ConfigType: &hcmpb.HttpFilter_TypedConfig{TypedConfig: transcodeConfigStruct},
			}
			return transcodeFilter, nil, nil
		}
	}

	// b/148605552: Previous versions of the `gcloud_build_image` script did not download the proto descriptor.
	// We cannot ensure that users have the latest version of the script, so notify them via non-fatal logs.
	// Log as error instead of warning because error logs will show up even if `--enable_debug` is false.
	glog.Error("Unable to setup gRPC-JSON transcoding because no proto descriptor was found in the service config. " +
		"Please use version 2020-01-29 (or later) of the `gcloud_build_image` script. " +
		"https://github.com/GoogleCloudPlatform/esp-v2/blob/master/docker/serverless/gcloud_build_image")
	return nil, nil, nil
}

func (g *GRPCTranscoderGenerator) GenPerRouteConfig(method *ci.MethodInfo, httpRule *httppattern.Pattern) (*anypb.Any, error) {
	return nil, fmt.Errorf("UNIMPLEMENTED")
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
	defaultBinding := &ahpb.HttpRule{Pattern: &ahpb.HttpRule_Post{Post: defaultPath}, Body: "*"}

	// Check existence of the default binding in httpRule's additional_bindings to avoid duplication.
	for _, additionalBinding := range httpRule.AdditionalBindings {
		if proto.Equal(additionalBinding, defaultBinding) {
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
