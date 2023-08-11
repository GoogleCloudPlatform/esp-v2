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
	"strings"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
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
	ProtoDescriptorBin []byte
	ServiceNames       []string

	// IgnoredQueryParams is the list of query params the transcoder should ignore.
	// Query params not in this list will result in request rejection.
	IgnoredQueryParams map[string]bool

	// DisabledSelectors contains selectors that the transcoder should be disabled
	// for via per-route filter config.
	DisabledSelectors map[string]bool

	// Below are all small behavior changes the API Producer can fine-tune via options.

	IgnoreUnknownQueryParameters       bool
	QueryParametersDisableUnescapePlus bool
	MatchUnregisteredCustomVerb        bool
	CaseInsensitiveEnumParsing         bool
	StrictRequestValidation            bool
	RejectCollision                    bool
	PrintOptions                       *transcoderpb.GrpcJsonTranscoder_PrintOptions

	NoopFilterGenerator
}

// NewGRPCTranscoderFilterGensFromOPConfig creates a GRPCTranscoderGenerator from
// OP service config + descriptor + ESPv2 options. It is a FilterGeneratorOPFactory.
func NewGRPCTranscoderFilterGensFromOPConfig(serviceConfig *confpb.Service, opts options.ConfigGeneratorOptions) ([]FilterGenerator, error) {
	grpcGen, err := NewGRPCTranscoderFilterGenFromOPConfig(serviceConfig, opts, true)
	if err != nil {
		return nil, err
	}
	if grpcGen == nil {
		return nil, nil
	}

	return []FilterGenerator{
		grpcGen,
	}, nil
}

// NewGRPCTranscoderFilterGenFromOPConfig creates a single GRPCTranscoderGenerator.
//
// It also has the option to skip the filter. If enabled, all checks will occur
// and filter may not be created. Otherwise, filter is always created.
func NewGRPCTranscoderFilterGenFromOPConfig(serviceConfig *confpb.Service, opts options.ConfigGeneratorOptions, maybeSkipFilter bool) (*GRPCTranscoderGenerator, error) {
	if maybeSkipFilter && opts.LocalHTTPBackendAddress != "" {
		glog.Warningf("Local http backend address is set to %q; skip transcoder filter completely.", opts.LocalHTTPBackendAddress)
		return nil, nil
	}

	isGRPCSupportRequired, err := IsGRPCSupportRequiredForOPConfig(serviceConfig, opts)
	if err != nil {
		return nil, err
	}
	if maybeSkipFilter && !isGRPCSupportRequired {
		glog.Infof("gRPC support is NOT required, skip transcoder filter completely.")
		return nil, nil
	}

	var descriptorBin []byte
	foundDescriptor := false

	for _, sourceFile := range serviceConfig.GetSourceInfo().GetSourceFiles() {
		configFile := &smpb.ConfigFile{}
		err := sourceFile.UnmarshalTo(configFile)
		if err != nil {
			continue
		}

		if configFile.GetFileType() == smpb.ConfigFile_FILE_DESCRIPTOR_SET_PROTO {
			foundDescriptor = true
			descriptorBin = configFile.GetFileContents()
		}
	}

	// Cannot check `descriptorBin == nil` because many tests use empty descriptor
	// to verify transcoding.
	if !foundDescriptor {
		// b/148605552: Previous versions of the `gcloud_build_image` script did not download the proto descriptor.
		// We cannot ensure that users have the latest version of the script, so notify them via non-fatal logs.
		// Log as error instead of warning because error logs will show up even if `--enable_debug` is false.
		glog.Error("Unable to setup gRPC-JSON transcoding because no proto descriptor was found in the service config. " +
			"Please use version 2020-01-29 (or later) of the `gcloud_build_image` script. " +
			"https://google3/third_party/espv2/source/v12/blob/master/docker/serverless/gcloud_build_image/gcloud_build_image")
		return nil, nil
	}

	descriptorBin, err = UpdateProtoDescriptorFromOPConfig(serviceConfig, opts, descriptorBin)
	if err != nil {
		return nil, err
	}

	ignoredQueryParams, err := GetIgnoredQueryParamsFromOPConfig(serviceConfig, opts)
	if err != nil {
		return nil, err
	}

	disabledSelectors, err := GetHTTPBackendSelectorsFromOPConfig(serviceConfig, opts)
	if err != nil {
		return nil, err
	}

	serviceNames := GetAPINamesListFromOPConfig(serviceConfig, opts)

	return &GRPCTranscoderGenerator{
		ProtoDescriptorBin:                 descriptorBin,
		ServiceNames:                       serviceNames,
		IgnoredQueryParams:                 ignoredQueryParams,
		DisabledSelectors:                  disabledSelectors,
		IgnoreUnknownQueryParameters:       opts.TranscodingIgnoreUnknownQueryParameters,
		QueryParametersDisableUnescapePlus: opts.TranscodingQueryParametersDisableUnescapePlus,
		MatchUnregisteredCustomVerb:        opts.TranscodingMatchUnregisteredCustomVerb,
		CaseInsensitiveEnumParsing:         opts.TranscodingCaseInsensitiveEnumParsing,
		StrictRequestValidation:            opts.TranscodingStrictRequestValidation,
		RejectCollision:                    opts.TranscodingRejectCollision,
		PrintOptions: &transcoderpb.GrpcJsonTranscoder_PrintOptions{
			AlwaysPrintPrimitiveFields: opts.TranscodingAlwaysPrintPrimitiveFields,
			AlwaysPrintEnumsAsInts:     opts.TranscodingAlwaysPrintEnumsAsInts,
			PreserveProtoFieldNames:    opts.TranscodingPreserveProtoFieldNames,
			StreamNewlineDelimited:     opts.TranscodingStreamNewLineDelimited,
		},
	}, nil
}

func (g *GRPCTranscoderGenerator) FilterName() string {
	return GRPCTranscoderFilterName
}

func (g *GRPCTranscoderGenerator) GenFilterConfig() (proto.Message, error) {
	var ignoredQueryParameterList []string
	for IgnoredQueryParameter := range g.IgnoredQueryParams {
		ignoredQueryParameterList = append(ignoredQueryParameterList, IgnoredQueryParameter)

	}
	sort.Sort(sort.StringSlice(ignoredQueryParameterList))

	transcodeConfig := &transcoderpb.GrpcJsonTranscoder{
		DescriptorSet: &transcoderpb.GrpcJsonTranscoder_ProtoDescriptorBin{
			ProtoDescriptorBin: g.ProtoDescriptorBin,
		},
		Services:                     g.ServiceNames,
		AutoMapping:                  true,
		ConvertGrpcStatus:            true,
		IgnoredQueryParameters:       ignoredQueryParameterList,
		IgnoreUnknownQueryParameters: g.IgnoreUnknownQueryParameters,
		QueryParamUnescapePlus:       !g.QueryParametersDisableUnescapePlus,
		PrintOptions:                 g.PrintOptions,
		MatchUnregisteredCustomVerb:  g.MatchUnregisteredCustomVerb,
		CaseInsensitiveEnumParsing:   g.CaseInsensitiveEnumParsing,
	}
	if g.StrictRequestValidation {
		transcodeConfig.RequestValidationOptions = &transcoderpb.GrpcJsonTranscoder_RequestValidationOptions{
			RejectUnknownMethod:              true,
			RejectUnknownQueryParameters:     true,
			RejectBindingBodyFieldCollisions: g.RejectCollision,
		}
	}
	return transcodeConfig, nil
}

func (g *GRPCTranscoderGenerator) GenPerRouteConfig(selector string, httpRule *httppattern.Pattern) (protov2.Message, error) {
	disabled := g.DisabledSelectors[selector]
	if !disabled {
		// Transcoding occurs for this selector because of listener-level filter config.
		return nil, nil
	}

	glog.Infof("Disable transcoder for the per-route config for method %q because it has HTTP backends.", selector)
	return &transcoderpb.GrpcJsonTranscoder{
		DescriptorSet: &transcoderpb.GrpcJsonTranscoder_ProtoDescriptor{
			ProtoDescriptor: "",
		},
	}, nil
}

// UpdateProtoDescriptorFromOPConfig mutates the proto descriptor based on
// OP service configuration.
//
// To support specifying custom http rules in service config.
// Envoy grpc_json_transcoder only uses the http.rules in the proto descriptor
// generated from "google.api.http" annotation in the proto file.
// For some shared grpc services, each service may want to define its own
// http.rules mapping. This function will copy the http.rules from the service config
// into proto descriptor.
//
// api-compiler has following behaviours:
//   - If a "google.api.http" annotation is specified in a method in the proto,
//     and the service config yaml doesn't specify one, api-compiler will copy it out
//     to the normalized service config.
//   - If a http.rule is specified in the service config, it will overwrite
//     the one from the proto annotation.
//
// So it should be ok to blindly copy the http.rules from the service config to
// proto descriptor.
func UpdateProtoDescriptorFromOPConfig(serviceConfig *confpb.Service, opts options.ConfigGeneratorOptions, descriptorBytes []byte) ([]byte, error) {
	ruleMap := make(map[string]*ahpb.HttpRule)
	for _, rule := range serviceConfig.GetHttp().GetRules() {
		ruleMap[rule.GetSelector()] = rule
	}
	apiMap := GetAPINamesSetFromOPConfig(serviceConfig, opts)

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
					PreserveDefaultHttpBinding(httpRule, defaultPath)
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

func PreserveDefaultHttpBinding(httpRule *ahpb.HttpRule, defaultPath string) {
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

// GetIgnoredQueryParamsFromOPConfig returns a list of query params that should be
// ignored during transcoding. These params may be for JWT authn, API key, etc.
//
// Replaces ServiceInfo::processTranscodingIgnoredQueryParams,
// ServiceInfo::processApiKeyLocations, and ServiceInfo::extractApiKeyLocations.
func GetIgnoredQueryParamsFromOPConfig(serviceConfig *confpb.Service, opts options.ConfigGeneratorOptions) (map[string]bool, error) {
	ignoredQueryParams := make(map[string]bool)

	// Process ignored query params from jwt locations
	authn := serviceConfig.GetAuthentication()
	for _, provider := range authn.GetProviders() {
		// no custom JwtLocation so use default ones and set the one in query
		// parameter for transcoder to ignore.
		if len(provider.JwtLocations) == 0 {
			ignoredQueryParams[util.DefaultJwtQueryParamAccessToken] = true
			continue
		}

		for _, jwtLocation := range provider.JwtLocations {
			switch jwtLocation.In.(type) {
			case *confpb.JwtLocation_Query:
				if jwtLocation.ValuePrefix != "" {
					return nil, fmt.Errorf("error processing authentication provider (%v): JwtLocation type [Query] should be set without valuePrefix, but it was set to [%v]", provider.Id, jwtLocation.ValuePrefix)
				}
				// set the custom JwtLocation in query parameter for transcoder to ignore.
				ignoredQueryParams[jwtLocation.GetQuery()] = true
			default:
				continue
			}
		}
	}

	// Process ignored query params from flag --transcoding_ignore_query_params
	if opts.TranscodingIgnoreQueryParameters != "" {
		ignoredQueryParametersFlag := strings.Split(opts.TranscodingIgnoreQueryParameters, ",")
		for _, ignoredQueryParameter := range ignoredQueryParametersFlag {
			ignoredQueryParams[ignoredQueryParameter] = true
		}
	}

	// Process ignored query params from API Key system parameters.
	apiKeySystemParametersBySelector := GetAPIKeySystemParametersBySelectorFromOPConfig(serviceConfig, opts)
	for _, api := range serviceConfig.GetApis() {
		for _, method := range api.GetMethods() {
			selector := MethodToSelector(api, method)

			systemParameters, ok := apiKeySystemParametersBySelector[selector]
			if !ok {
				// If any of method is not set with custom ApiKeyLocations, use the default
				// one and set the custom ApiKeyLocations in query parameter for transcoder
				// to ignore.
				ignoredQueryParams[util.DefaultApiKeyQueryParamKey] = true
				ignoredQueryParams[util.DefaultApiKeyQueryParamApiKey] = true
				continue
			}

			for _, systemParameter := range systemParameters {
				if systemParameter.GetUrlQueryParameter() != "" {
					ignoredQueryParams[systemParameter.GetUrlQueryParameter()] = true
				}
			}
		}
	}

	return ignoredQueryParams, nil
}

// GetHTTPBackendSelectorsFromOPConfig returns a list of selectors that the transcoder
// should be completely disabled for. Useful for local (non-OpenAPI) HTTP backend usage.
// Not really used for ESPv2.
func GetHTTPBackendSelectorsFromOPConfig(serviceConfig *confpb.Service, opts options.ConfigGeneratorOptions) (map[string]bool, error) {
	disabledSelectors := make(map[string]bool)
	if opts.EnableBackendAddressOverride {
		glog.Infof("Skipping create grpc transcoding disabled selectors because backend address override is enabled.")
		return disabledSelectors, nil
	}

	for _, rule := range serviceConfig.GetBackend().GetRules() {
		if util.ShouldSkipOPDiscoveryAPI(rule.GetSelector(), opts.AllowDiscoveryAPIs) {
			glog.Warningf("Skip backend rule %q because discovery API is not supported.", rule.GetSelector())
			continue
		}

		if _, ok := rule.GetOverridesByRequestProtocol()[util.HTTPBackendProtocolKey]; ok {
			disabledSelectors[rule.GetSelector()] = true
		}
	}

	return disabledSelectors, nil
}
