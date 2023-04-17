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
	"github.com/golang/glog"
	ahpb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
	"google.golang.org/protobuf/proto"
	protov2 "google.golang.org/protobuf/proto"
	descpb "google.golang.org/protobuf/types/descriptorpb"
)

const (
	// GRPCTranscoderFilterName is the Envoy filter name for debug logging.
	GRPCTranscoderFilterName = "envoy.filters.http.grpc_json_transcoder"
)

type GRPCTranscoderGenerator struct {
	// configFile points to the proto descriptor file. Nil if no descriptor.
	configFile *smpb.ConfigFile

	// skipFilter indicates if this filter is disabled based on options and config.
	skipFilter bool
}

// NewGRPCTranscoderGenerator creates the GRPCTranscoderGenerator with cached config.
func NewGRPCTranscoderGenerator(serviceInfo *ci.ServiceInfo) *GRPCTranscoderGenerator {
	gen := &GRPCTranscoderGenerator{}

	if serviceInfo.Options.LocalHTTPBackendAddress != "" {
		glog.Warningf("Local http backend address is set to %q; skip transcoder filter completely.", serviceInfo.Options.LocalHTTPBackendAddress)
		gen.skipFilter = true
	}

	if !serviceInfo.GrpcSupportRequired {
		gen.skipFilter = true
	}

	for _, sourceFile := range serviceInfo.ServiceConfig().GetSourceInfo().GetSourceFiles() {
		// Error ignored to match pre-existing behavior.
		configFile := &smpb.ConfigFile{}
		_ = sourceFile.UnmarshalTo(configFile)
		if configFile.GetFileType() == smpb.ConfigFile_FILE_DESCRIPTOR_SET_PROTO {
			gen.configFile = configFile
		}
	}

	return gen
}

func (g *GRPCTranscoderGenerator) FilterName() string {
	return GRPCTranscoderFilterName
}

func (g *GRPCTranscoderGenerator) IsEnabled() bool {
	if g.skipFilter {
		return false
	}

	if g.configFile != nil {
		return true
	}

	// b/148605552: Previous versions of the `gcloud_build_image` script did not download the proto descriptor.
	// We cannot ensure that users have the latest version of the script, so notify them via non-fatal logs.
	// Log as error instead of warning because error logs will show up even if `--enable_debug` is false.
	glog.Error("Unable to setup gRPC-JSON transcoding because no proto descriptor was found in the service config. " +
			"Please use version 2020-01-29 (or later) of the `gcloud_build_image` script. " +
			"https://github.com/GoogleCloudPlatform/esp-v2/blob/master/docker/serverless/gcloud_build_image")
	return false
}

func (g *GRPCTranscoderGenerator) GenFilterConfig(serviceInfo *ci.ServiceInfo) (proto.Message, error) {
	if g.configFile == nil {
		return nil, fmt.Errorf("internal error, config file should be set as transcoder filer is enabled")
	}

	var ignoredQueryParameterList []string
	for IgnoredQueryParameter := range serviceInfo.AllTranscodingIgnoredQueryParams {
		ignoredQueryParameterList = append(ignoredQueryParameterList, IgnoredQueryParameter)

	}
	sort.Sort(sort.StringSlice(ignoredQueryParameterList))

	configContent, err := updateProtoDescriptor(serviceInfo.ServiceConfig(), serviceInfo.ApiNames,
		g.configFile.GetFileContents())
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
	return transcodeConfig, nil
}

func (g *GRPCTranscoderGenerator) GenPerRouteConfig(method *ci.MethodInfo, httpRule *httppattern.Pattern) (proto.Message, error) {
	if method.HttpBackendInfo != nil {
		glog.Infof("Disable transcoder for the per-route config for method %q because it has HTTP backends.", method.Operation())
		return &transcoderpb.GrpcJsonTranscoder{
			DescriptorSet: &transcoderpb.GrpcJsonTranscoder_ProtoDescriptor{
				ProtoDescriptor: "",
			},
		}, nil
	}
	return nil, nil
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
	if err := protov2.Unmarshal(descriptorBytes, fds); err != nil {
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
					protov2.SetExtension(method.GetOptions(), ahpb.E_Http, rule)
				}

				// If an http rule is specified for a rpc endpoint then the rpc's default http binding will be
				// disabled according to the logic in the envoy's json transcoder filter. To still enable
				// the default http binding, which is the designed behavior, the default http binding needs to be
				// added to the http rule's additional bindings.
				if httpRule := protov2.GetExtension(method.GetOptions(), ahpb.E_Http).(*ahpb.HttpRule); httpRule != nil {
					defaultPath := fmt.Sprintf("/%s/%s", apiName, method.GetName())
					preserveDefaultHttpBinding(httpRule, defaultPath)
				}
			}
		}
	}

	newData, err := protov2.Marshal(fds)
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
		if protov2.Equal(additionalBinding, defaultBinding) {
			return
		}
	}
	// check if httpRule is the same as default binding, ignore the difference in fields selector and
	// additional_bindings.
	defaultBinding.Selector = httpRule.GetSelector()
	defaultBinding.AdditionalBindings = httpRule.GetAdditionalBindings()
	if protov2.Equal(httpRule, defaultBinding) {
		return
	}
	defaultBinding.Selector = ""
	defaultBinding.AdditionalBindings = nil

	httpRule.AdditionalBindings = append(httpRule.AdditionalBindings, defaultBinding)
}
