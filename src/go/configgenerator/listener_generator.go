// Copyright 2019 Google Cloud Platform Proxy Authors
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
	"sort"
	"strings"

	"cloudesf.googlesource.com/gcpproxy/src/go/flags"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/envoyproxy/go-control-plane/pkg/util"
	"github.com/gogo/protobuf/types"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/duration"
	"google.golang.org/genproto/googleapis/api/annotations"

	sc "cloudesf.googlesource.com/gcpproxy/src/go/configinfo"
	bapb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/backend_auth"
	brpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/backend_routing"
	commonpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/common"
	pmpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/path_matcher"
	scpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/service_control"
	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	ac "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/jwt_authn/v2alpha"
	tc "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/transcoder/v2"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	sm "github.com/google/go-genproto/googleapis/api/servicemanagement/v1"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

const (
	statPrefix = "ingress_http"
)

func MakeListener(serviceInfo *sc.ServiceInfo, backendProtocol ut.BackendProtocol) (*v2.Listener, error) {
	httpFilters := []*hcm.HttpFilter{}

	if *flags.CorsPreset == "basic" || *flags.CorsPreset == "cors_with_regex" {
		corsFilter := &hcm.HttpFilter{
			Name: util.CORS,
		}
		httpFilters = append(httpFilters, corsFilter)
		glog.Infof("adding CORS Filter config: %v", corsFilter)
	}

	// Add Path Matcher filter. The following filters rely on the dynamic
	// metadata populated by Path Matcher filter.
	// * Service Control filter
	// * Backend Authentication filter
	// * Backend Routing filter (WIP)
	pathMathcherFilter := makePathMatcherFilter(serviceInfo, backendProtocol)
	if pathMathcherFilter != nil {
		httpFilters = append(httpFilters, pathMathcherFilter)
		glog.Infof("adding Path Matcher Filter config: %v", pathMathcherFilter)
	}

	// Add JWT Authn filter if needed.
	if !*flags.SkipJwtAuthnFilter {
		jwtAuthnFilter := makeJwtAuthnFilter(serviceInfo, backendProtocol)
		if jwtAuthnFilter != nil {
			httpFilters = append(httpFilters, jwtAuthnFilter)
			glog.Infof("adding JWT Authn Filter config: %v", jwtAuthnFilter)
		}
	}

	// Add Service Control filter if needed.
	if !*flags.SkipServiceControlFilter {
		serviceControlFilter := makeServiceControlFilter(serviceInfo, backendProtocol)
		if serviceControlFilter != nil {
			httpFilters = append(httpFilters, serviceControlFilter)
			glog.Infof("adding Service Control Filter config: %v", serviceControlFilter)
		}
	}

	// Add gRPC Transcoder filter and gRPCWeb filter configs for gRPC backend.
	if backendProtocol == ut.GRPC {
		transcoderFilter := makeTranscoderFilter(serviceInfo)
		if transcoderFilter != nil {
			httpFilters = append(httpFilters, transcoderFilter)
			glog.Infof("adding Transcoder Filter config: %v", transcoderFilter)
		}

		grpcWebFilter := &hcm.HttpFilter{
			Name:       util.GRPCWeb,
			ConfigType: &hcm.HttpFilter_Config{&types.Struct{}},
		}
		httpFilters = append(httpFilters, grpcWebFilter)
	}

	// Add Backend Auth filter and Backend Routing if needed.
	if *flags.EnableBackendRouting {
		backendAuthFilter := makeBackendAuthFilter(serviceInfo)
		httpFilters = append(httpFilters, backendAuthFilter)
		glog.Infof("adding Backend Auth Filter config: %v", backendAuthFilter)
		backendRoutingFilter := makeBackendRoutingFilter(serviceInfo)
		httpFilters = append(httpFilters, backendRoutingFilter)
		glog.Infof("adding Backend Routing Filter config: %v", backendRoutingFilter)
	}

	// Add Envoy Router filter so requests are routed upstream.
	// Router filter should be the last.
	routerFilter := &hcm.HttpFilter{
		Name:       util.Router,
		ConfigType: &hcm.HttpFilter_Config{&types.Struct{}},
	}
	httpFilters = append(httpFilters, routerFilter)

	route, err := MakeRouteConfig(serviceInfo)
	if err != nil {
		return nil, fmt.Errorf("makeHttpConnectionManagerRouteConfig got err: %s", err)
	}

	httpConMgr := &hcm.HttpConnectionManager{
		CodecType:  hcm.AUTO,
		StatPrefix: statPrefix,
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: route,
		},
	}

	glog.Infof("adding Http Connection Manager config: %v", httpConMgr)
	httpConMgr.HttpFilters = httpFilters

	// HTTP filter configuration
	httpFilterConfig, err := util.MessageToStruct(httpConMgr)
	if err != nil {
		return nil, err
	}

	return &v2.Listener{
		Address: core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address: *flags.ListenerAddress,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: uint32(*flags.ListenerPort),
					},
				},
			},
		},
		FilterChains: []listener.FilterChain{
			{
				Filters: []listener.Filter{
					{
						Name:       util.HTTPConnectionManager,
						ConfigType: &listener.Filter_Config{httpFilterConfig},
					},
				},
			},
		},
	}, nil
}

func makePathMatcherFilter(serviceInfo *sc.ServiceInfo, backendProtocol ut.BackendProtocol) *hcm.HttpFilter {
	rules := []*pmpb.PathMatcherRule{}
	if backendProtocol == ut.GRPC {
		for _, method := range serviceInfo.ServiceConfig().GetApis()[0].GetMethods() {
			rules = append(rules, &pmpb.PathMatcherRule{
				Operation: fmt.Sprintf("%s.%s", serviceInfo.ApiName, method.GetName()),
				Pattern: &commonpb.Pattern{
					UriTemplate: fmt.Sprintf("/%s/%s", serviceInfo.ApiName, method.GetName()),
					HttpMethod:  ut.POST,
				},
			})
		}
	}

	constantAddressRules := make(map[string]bool)
	for _, rule := range serviceInfo.ServiceConfig().GetBackend().GetRules() {
		if rule.GetPathTranslation() == conf.BackendRule_CONSTANT_ADDRESS {
			constantAddressRules[rule.GetSelector()] = true
		}
	}

	for _, httpRule := range serviceInfo.ServiceConfig().GetHttp().GetRules() {
		var newPattern *commonpb.Pattern
		switch httpPattern := httpRule.GetPattern().(type) {
		case *annotations.HttpRule_Get:
			newPattern = &commonpb.Pattern{
				UriTemplate: httpPattern.Get,
				HttpMethod:  ut.GET,
			}
		case *annotations.HttpRule_Put:
			newPattern = &commonpb.Pattern{
				UriTemplate: httpPattern.Put,
				HttpMethod:  ut.PUT,
			}
		case *annotations.HttpRule_Post:
			newPattern = &commonpb.Pattern{
				UriTemplate: httpPattern.Post,
				HttpMethod:  ut.POST,
			}
		case *annotations.HttpRule_Delete:
			newPattern = &commonpb.Pattern{
				UriTemplate: httpPattern.Delete,
				HttpMethod:  ut.DELETE,
			}
		case *annotations.HttpRule_Patch:
			newPattern = &commonpb.Pattern{
				UriTemplate: httpPattern.Patch,
				HttpMethod:  ut.PATCH,
			}
		// TODO(kyuc): might need to handle HttpRule_Custom as well
		case *annotations.HttpRule_Custom:
			if httpPattern.Custom.Kind == ut.OPTIONS {
				newPattern = &commonpb.Pattern{
					UriTemplate: httpPattern.Custom.Path,
					HttpMethod:  ut.OPTIONS,
				}
			}
		}

		newRule := &pmpb.PathMatcherRule{
			Operation: httpRule.GetSelector(),
			Pattern:   newPattern,
		}

		isConstantAddress := constantAddressRules[httpRule.GetSelector()]
		if isConstantAddress && hasPathParameter(newPattern.UriTemplate) {
			newRule.ExtractPathParameters = true
		}

		rules = append(rules, newRule)
	}

	serviceInfo.OperationSet = make(map[string]bool)
	for _, rule := range rules {
		serviceInfo.OperationSet[rule.Operation] = true
	}

	// TODO(kyuc): should we support CORS for gRPC?
	// In order to support CORS. HTTP method OPTIONS needs to be added to all
	// urls except the ones already with options.
	if serviceInfo.GetEndpointAllowCorsFlag() {
		httpPathArray := make([]*sc.HttpRule, 0)
		for _, v := range serviceInfo.HttpPathMap {
			httpPathArray = append(httpPathArray, v)
		}
		sort.Slice(httpPathArray, func(i, j int) bool {
			if httpPathArray[i].Path == httpPathArray[j].Path {
				return httpPathArray[i].Method < httpPathArray[i].Method
			}
			return httpPathArray[i].Path < httpPathArray[j].Path
		})
		// All options have their operation as the following format: CORS.suffix.
		// Appends suffix to make sure it is not used by any http rules.
		corsOperationBase := "CORS"
		corsID := 0
		for _, v := range httpPathArray {
			path := v.Path
			if _, exist := serviceInfo.HttpPathWithOptionsSet[path]; !exist {
				corsOperation := ""
				for {
					corsOperation = fmt.Sprintf("%s.%d", corsOperationBase, corsID)
					corsID++
					if !serviceInfo.OperationSet[corsOperation] {
						break
					}
				}

				optionsPattern := &commonpb.Pattern{
					UriTemplate: path,
					HttpMethod:  ut.OPTIONS,
				}

				newRule := &pmpb.PathMatcherRule{
					Operation: corsOperation,
					Pattern:   optionsPattern,
				}
				serviceInfo.GeneratedOptionsOperations = append(serviceInfo.GeneratedOptionsOperations, corsOperation)
				rules = append(rules, newRule)
			}
		}
	}

	if len(rules) == 0 {
		return nil
	}

	// Create snake name to JSON name mapping.
	var segmentNames []*pmpb.SegmentName
	for _, t := range serviceInfo.ServiceConfig().GetTypes() {
		for _, f := range t.GetFields() {
			if strings.ContainsRune(f.GetName(), '_') {
				segmentNames = append(segmentNames, &pmpb.SegmentName{
					SnakeName: f.GetName(),
					JsonName:  f.GetJsonName(),
				})
			}
		}
	}

	pathMathcherConfig := &pmpb.FilterConfig{Rules: rules}
	if len(segmentNames) > 0 {
		pathMathcherConfig.SegmentNames = segmentNames
	}

	pathMathcherConfigStruct, _ := util.MessageToStruct(pathMathcherConfig)
	pathMatcherFilter := &hcm.HttpFilter{
		Name:       ut.PathMatcher,
		ConfigType: &hcm.HttpFilter_Config{pathMathcherConfigStruct},
	}
	return pathMatcherFilter
}

func hasPathParameter(httpPattern string) bool {
	return strings.ContainsRune(httpPattern, '{')
}

func makeJwtAuthnFilter(serviceInfo *sc.ServiceInfo, backendProtocol ut.BackendProtocol) *hcm.HttpFilter {
	auth := serviceInfo.ServiceConfig().GetAuthentication()
	if len(auth.GetProviders()) == 0 {
		return nil
	}
	providers := make(map[string]*ac.JwtProvider)
	for _, provider := range auth.GetProviders() {
		jwk, err := fetchJwk(provider.GetJwksUri())
		if err != nil {
			glog.Warningf("fetch jwk from issuer %s got error: %s", provider.GetIssuer(), err)
			continue
		}
		jp := &ac.JwtProvider{
			Issuer: provider.GetIssuer(),
			JwksSourceSpecifier: &ac.JwtProvider_LocalJwks{
				LocalJwks: &core.DataSource{
					Specifier: &core.DataSource_InlineString{
						InlineString: string(jwk),
					},
				},
			},
		}
		if len(provider.GetAudiences()) != 0 {
			for _, a := range strings.Split(provider.GetAudiences(), ",") {
				jp.Audiences = append(jp.Audiences, strings.TrimSpace(a))
			}
		}
		providers[provider.GetId()] = jp
	}

	if len(providers) == 0 {
		return nil
	}

	rules := []*ac.RequirementRule{}
	for _, rule := range auth.GetRules() {
		if len(rule.GetRequirements()) == 0 {
			continue
		}
		// By default, if there are multi requirements, treat it as RequireAny.
		requires := &ac.JwtRequirement{
			RequiresType: &ac.JwtRequirement_RequiresAny{
				RequiresAny: &ac.JwtRequirementOrList{},
			},
		}
		for _, r := range rule.GetRequirements() {
			var require *ac.JwtRequirement
			if r.GetAudiences() == "" {
				require = &ac.JwtRequirement{
					RequiresType: &ac.JwtRequirement_ProviderName{
						ProviderName: r.GetProviderId(),
					},
				}
			} else {
				var audiences []string
				for _, a := range strings.Split(r.GetAudiences(), ",") {
					audiences = append(audiences, strings.TrimSpace(a))
				}
				require = &ac.JwtRequirement{
					RequiresType: &ac.JwtRequirement_ProviderAndAudiences{
						ProviderAndAudiences: &ac.ProviderWithAudiences{
							ProviderName: r.GetProviderId(),
							Audiences:    audiences,
						},
					},
				}
			}
			if len(rule.GetRequirements()) == 1 {
				requires = require
			} else {
				requires.GetRequiresAny().Requirements = append(requires.GetRequiresAny().GetRequirements(), require)
			}
		}

		if httpRule, ok := serviceInfo.HttpPathMap[rule.GetSelector()]; ok {
			ruleConfig := &ac.RequirementRule{
				Match:    makeHttpRouteMatcher(httpRule),
				Requires: requires,
			}
			rules = append(rules, ruleConfig)
		}

		s := strings.Split(rule.GetSelector(), ".")
		// For gRPC protocol, needs to add extra match rule for grpc client.
		if backendProtocol == ut.GRPC {
			rules = append(rules, &ac.RequirementRule{
				Match: &route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Path{
						Path: fmt.Sprintf("/%s/%s", serviceInfo.ApiName, s[len(s)-1]),
					},
				},
				Requires: requires,
			})
		}
	}

	jwtAuthentication := &ac.JwtAuthentication{
		Providers: providers,
		Rules:     rules,
	}

	jas, _ := util.MessageToStruct(jwtAuthentication)
	jwtAuthnFilter := &hcm.HttpFilter{
		Name:       ut.JwtAuthn,
		ConfigType: &hcm.HttpFilter_Config{jas},
	}
	return jwtAuthnFilter
}

func makeServiceControlFilter(serviceInfo *sc.ServiceInfo, backendProtocol ut.BackendProtocol) *hcm.HttpFilter {
	if serviceInfo == nil || serviceInfo.ServiceConfig().GetControl().GetEnvironment() == "" {
		return nil
	}
	lowercaseProtocol := strings.ToLower(*flags.BackendProtocol)
	serviceName := serviceInfo.ServiceConfig().GetName()
	service := &scpb.Service{
		ServiceName:       serviceName,
		ServiceConfigId:   serviceInfo.ConfigID,
		ProducerProjectId: serviceInfo.ServiceConfig().GetProducerProjectId(),
		TokenCluster:      ut.TokenCluster,
		ServiceControlUri: &scpb.HttpUri{
			Uri:     serviceInfo.ServiceControlURI,
			Cluster: serviceControlClusterName,
			Timeout: &duration.Duration{Seconds: 5},
		},
		ServiceConfig:   copyServiceConfigForReportMetrics(serviceInfo.ServiceConfig()),
		BackendProtocol: lowercaseProtocol,
	}

	requirementMap := make(map[string]*scpb.Requirement)
	for operation := range serviceInfo.OperationSet {
		requirementMap[operation] = &scpb.Requirement{
			ServiceName:   serviceName,
			OperationName: operation,
		}
	}

	// For these OPTIONS methods, auth should be disabled and AllowWithoutApiKey
	// should be true for each CORS
	for _, corsOperation := range serviceInfo.GeneratedOptionsOperations {
		requirementMap[corsOperation] =
			&scpb.Requirement{
				ServiceName:   serviceName,
				OperationName: corsOperation,
				ApiKey: &scpb.APIKeyRequirement{
					AllowWithoutApiKey: true,
				},
			}
	}

	for _, usageRule := range serviceInfo.ServiceConfig().GetUsage().GetRules() {
		requirement, ok := requirementMap[usageRule.GetSelector()]
		if !ok {
			continue
		}
		requirement.ApiKey = &scpb.APIKeyRequirement{
			AllowWithoutApiKey: usageRule.GetAllowUnregisteredCalls(),
		}
	}

	filterConfig := &scpb.FilterConfig{
		Services: []*scpb.Service{service},
	}

	if serviceInfo.GcpAttributes != nil {
		filterConfig.GcpAttributes = serviceInfo.GcpAttributes
	}

	// Map order is not deterministic, so sort by key here to make the filter
	// config rules order deterministic. Simply iterating map will introduce
	// flakiness to the tests.
	var operations []string
	for operation := range requirementMap {
		operations = append(operations, operation)
	}
	sort.Strings(operations)

	for _, operation := range operations {
		filterConfig.Requirements = append(filterConfig.Requirements, requirementMap[operation])
	}

	scs, err := ut.MessageToStruct(filterConfig)
	if err != nil {
		glog.Warningf("failed to convert message to struct: %v", err)
	}
	filter := &hcm.HttpFilter{
		Name:       ut.ServiceControl,
		ConfigType: &hcm.HttpFilter_Config{scs},
	}
	return filter
}

func copyServiceConfigForReportMetrics(src *conf.Service) *any.Any {
	// Logs and metrics fields are needed by the Envoy HTTP filter
	// to generate proper Metrics for Report calls.
	copy := &conf.Service{
		Logs:               src.GetLogs(),
		Metrics:            src.GetMetrics(),
		MonitoredResources: src.GetMonitoredResources(),
		Monitoring:         src.GetMonitoring(),
		Logging:            src.GetLogging(),
	}
	a, err := ptypes.MarshalAny(copy)
	if err != nil {
		glog.Warningf("failed to copy certain service config, error: %v", err)
		return nil
	}
	return a
}

func makeTranscoderFilter(serviceInfo *sc.ServiceInfo) *hcm.HttpFilter {
	for _, sourceFile := range serviceInfo.ServiceConfig().GetSourceInfo().GetSourceFiles() {
		configFile := &sm.ConfigFile{}
		ptypes.UnmarshalAny(sourceFile, configFile)
		glog.Infof("got proto descriptor: %v", string(configFile.GetFileContents()))

		if configFile.GetFileType() == sm.ConfigFile_FILE_DESCRIPTOR_SET_PROTO {
			configContent := configFile.GetFileContents()
			transcodeConfig := &tc.GrpcJsonTranscoder{
				DescriptorSet: &tc.GrpcJsonTranscoder_ProtoDescriptorBin{
					ProtoDescriptorBin: configContent,
				},
				Services:               []string{serviceInfo.ApiName},
				IgnoredQueryParameters: []string{"api_key", "key", "access_token"},
			}
			transcodeConfigStruct, _ := util.MessageToStruct(transcodeConfig)
			transcodeFilter := &hcm.HttpFilter{
				Name:       util.GRPCJSONTranscoder,
				ConfigType: &hcm.HttpFilter_Config{transcodeConfigStruct},
			}
			return transcodeFilter
		}
	}
	return nil
}

func makeBackendAuthFilter(serviceInfo *sc.ServiceInfo) *hcm.HttpFilter {
	rules := []*bapb.BackendAuthRule{}
	for _, rule := range serviceInfo.ServiceConfig().GetBackend().GetRules() {
		if rule.GetSelector() == "" || rule.GetJwtAudience() == "" {
			continue
		}
		rule.GetJwtAudience()
		rules = append(rules,
			&bapb.BackendAuthRule{
				Operation:    rule.GetSelector(),
				JwtAudience:  rule.GetJwtAudience(),
				TokenCluster: ut.TokenCluster,
			})
	}
	backendAuthConfig := &bapb.FilterConfig{Rules: rules}
	backendAuthConfigStruct, _ := util.MessageToStruct(backendAuthConfig)
	backendAuthFilter := &hcm.HttpFilter{
		Name:       ut.BackendAuth,
		ConfigType: &hcm.HttpFilter_Config{backendAuthConfigStruct},
	}
	return backendAuthFilter
}

func makeBackendRoutingFilter(serviceInfo *sc.ServiceInfo) *hcm.HttpFilter {
	rules := []*brpb.BackendRoutingRule{}
	for _, v := range serviceInfo.BackendRoutingInfos {
		rules = append(rules, &brpb.BackendRoutingRule{
			Operation:      v.Selector,
			IsConstAddress: v.TranslationType == conf.BackendRule_CONSTANT_ADDRESS,
			PathPrefix:     v.Uri,
		})
	}
	backendRoutingConfig := &brpb.FilterConfig{Rules: rules}
	backendRoutingConfigStruct, _ := util.MessageToStruct(backendRoutingConfig)
	backendRoutingFilter := &hcm.HttpFilter{
		Name:       ut.BackendRouting,
		ConfigType: &hcm.HttpFilter_Config{backendRoutingConfigStruct},
	}
	return backendRoutingFilter
}
