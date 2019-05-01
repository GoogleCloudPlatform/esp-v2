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
	"strings"

	"cloudesf.googlesource.com/gcpproxy/src/go/flags"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/pkg/util"
	"github.com/gogo/protobuf/types"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/duration"

	sc "cloudesf.googlesource.com/gcpproxy/src/go/configinfo"
	bapb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/backend_auth"
	brpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/backend_routing"
	commonpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/common"
	pmpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/path_matcher"
	scpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/service_control"
	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	ac "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/jwt_authn/v2alpha"
	rt "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/router/v2"
	tc "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/transcoder/v2"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	sm "github.com/google/go-genproto/googleapis/api/servicemanagement/v1"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

const (
	statPrefix = "ingress_http"
)

func MakeListener(serviceInfo *sc.ServiceInfo) (*v2.Listener, error) {
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
	// * Jwt Authentication filter
	// * Service Control filter
	// * Backend Authentication filter
	// * Backend Routing filter
	pathMathcherFilter := makePathMatcherFilter(serviceInfo)
	if pathMathcherFilter != nil {
		httpFilters = append(httpFilters, pathMathcherFilter)
		glog.Infof("adding Path Matcher Filter config: %v", pathMathcherFilter)
	}

	// Add JWT Authn filter if needed.
	if !*flags.SkipJwtAuthnFilter {
		jwtAuthnFilter := makeJwtAuthnFilter(serviceInfo)
		if jwtAuthnFilter != nil {
			httpFilters = append(httpFilters, jwtAuthnFilter)
			glog.Infof("adding JWT Authn Filter config: %v", jwtAuthnFilter)
		}
	}

	// Add Service Control filter if needed.
	if !*flags.SkipServiceControlFilter {
		serviceControlFilter := makeServiceControlFilter(serviceInfo)
		if serviceControlFilter != nil {
			httpFilters = append(httpFilters, serviceControlFilter)
			glog.Infof("adding Service Control Filter config: %v", serviceControlFilter)
		}
	}

	// Add gRPC Transcoder filter and gRPCWeb filter configs for gRPC backend.
	if serviceInfo.BackendProtocol == ut.GRPC {
		transcoderFilter := makeTranscoderFilter(serviceInfo)
		if transcoderFilter != nil {
			httpFilters = append(httpFilters, transcoderFilter)
			glog.Infof("adding Transcoder Filter config...")
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

	routerFilter := makeRouterFilter()
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

		UseRemoteAddress:  &types.BoolValue{Value: *flags.EnvoyUseRemoteAddress},
		XffNumTrustedHops: uint32(*flags.EnvoyXffNumTrustedHops),
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

func makePathMatcherFilter(serviceInfo *sc.ServiceInfo) *hcm.HttpFilter {
	rules := []*pmpb.PathMatcherRule{}
	for _, operation := range serviceInfo.Operations {
		method := serviceInfo.Methods[operation]
		// Adds PathMatcherRule for gRPC method.
		if serviceInfo.BackendProtocol == ut.GRPC {
			newGrpcRule := &pmpb.PathMatcherRule{
				Operation: operation,
				Pattern: &commonpb.Pattern{
					UriTemplate: fmt.Sprintf("/%s/%s", serviceInfo.ApiName, method.ShortName),
					HttpMethod:  ut.POST,
				},
			}
			rules = append(rules, newGrpcRule)
		}

		// Adds PathMatcherRule for HTTP method, whose HttpRule is not empty.
		if method.HttpRule.UriTemplate != "" && method.HttpRule.HttpMethod != "" {
			newHttpRule := &pmpb.PathMatcherRule{
				Operation: operation,
				Pattern:   &method.HttpRule,
			}
			if method.BackendRule.TranslationType == conf.BackendRule_CONSTANT_ADDRESS && hasPathParameter(newHttpRule.Pattern.UriTemplate) {
				newHttpRule.ExtractPathParameters = true
			}
			rules = append(rules, newHttpRule)
		}
	}

	if len(rules) == 0 {
		return nil
	}

	pathMathcherConfig := &pmpb.FilterConfig{Rules: rules}
	if len(serviceInfo.SegmentNames) > 0 {
		pathMathcherConfig.SegmentNames = serviceInfo.SegmentNames
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

func makeJwtAuthnFilter(serviceInfo *sc.ServiceInfo) *hcm.HttpFilter {
	auth := serviceInfo.ServiceConfig().GetAuthentication()
	if len(auth.GetProviders()) == 0 {
		return nil
	}
	providers := make(map[string]*ac.JwtProvider)
	for _, provider := range auth.GetProviders() {
		jp := &ac.JwtProvider{
			Issuer: provider.GetIssuer(),
			JwksSourceSpecifier: &ac.JwtProvider_RemoteJwks{
				RemoteJwks: &ac.RemoteJwks{
					HttpUri: &core.HttpUri{
						Uri: provider.GetJwksUri(),
						HttpUpstreamType: &core.HttpUri_Cluster{
							Cluster: provider.GetIssuer(),
						},
					},
					CacheDuration: &types.Duration{
						Seconds: int64(300),
					},
				},
			},
		}
		if len(provider.GetAudiences()) != 0 {
			for _, a := range strings.Split(provider.GetAudiences(), ",") {
				jp.Audiences = append(jp.Audiences, strings.TrimSpace(a))
			}
		}
		// TODO(taoxuy): add unit test
		// the JWT Payload will be send to metadata by envoy and it will be used by service control filter
		// for logging and setting credential_id
		jp.PayloadInMetadata = ut.JwtPayloadMetadataName
		providers[provider.GetId()] = jp
	}

	if len(providers) == 0 {
		return nil
	}

	requirements := make(map[string]*ac.JwtRequirement)
	for _, rule := range auth.GetRules() {
		if len(rule.GetRequirements()) > 0 {
			requirements[rule.GetSelector()] = makeJwtRequirement(rule.GetRequirements())
		}
	}

	jwtAuthentication := &ac.JwtAuthentication{
		Providers: providers,
		FilterStateRules: &ac.FilterStateRule{
			Name:     "envoy.filters.http.path_matcher.operation",
			Requires: requirements,
		},
	}

	jas, _ := util.MessageToStruct(jwtAuthentication)
	jwtAuthnFilter := &hcm.HttpFilter{
		Name:       ut.JwtAuthn,
		ConfigType: &hcm.HttpFilter_Config{jas},
	}
	return jwtAuthnFilter
}

func makeJwtRequirement(requirements []*conf.AuthRequirement) *ac.JwtRequirement {
	// By default, if there are multi requirements, treat it as RequireAny.
	requires := &ac.JwtRequirement{
		RequiresType: &ac.JwtRequirement_RequiresAny{
			RequiresAny: &ac.JwtRequirementOrList{},
		},
	}

	for _, r := range requirements {
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
		if len(requirements) == 1 {
			requires = require
		} else {
			requires.GetRequiresAny().Requirements = append(requires.GetRequiresAny().GetRequirements(), require)
		}
	}

	return requires
}

func makeServiceControlFilter(serviceInfo *sc.ServiceInfo) *hcm.HttpFilter {
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
	if *flags.LogRequestHeaders != "" {
		service.LogRequestHeaders = strings.Split(*flags.LogRequestHeaders, ",")
		for i := range service.LogRequestHeaders {
			service.LogRequestHeaders[i] = strings.TrimSpace(service.LogRequestHeaders[i])
		}
	}
	if *flags.LogResponseHeaders != "" {
		service.LogResponseHeaders = strings.Split(*flags.LogResponseHeaders, ",")
		for i := range service.LogResponseHeaders {
			service.LogResponseHeaders[i] = strings.TrimSpace(service.LogResponseHeaders[i])
		}
	}
	if *flags.LogJwtPayloads != "" {
		service.LogJwtPayloads = strings.Split(*flags.LogJwtPayloads, ",")
		for i := range service.LogJwtPayloads {
			service.LogJwtPayloads[i] = strings.TrimSpace(service.LogJwtPayloads[i])
		}
	}
	service.JwtPayloadMetadataName = ut.JwtPayloadMetadataName

	filterConfig := &scpb.FilterConfig{
		Services: []*scpb.Service{service},
	}
	if serviceInfo.GcpAttributes != nil {
		filterConfig.GcpAttributes = serviceInfo.GcpAttributes
	}

	for _, operation := range serviceInfo.Operations {
		method := serviceInfo.Methods[operation]
		requirement := &scpb.Requirement{
			ServiceName:   serviceName,
			OperationName: operation,
		}

		// For these OPTIONS methods, auth should be disabled and AllowWithoutApiKey
		// should be true for each CORS.
		if method.IsGeneratedOption || method.AllowUnregisteredCalls {
			requirement.ApiKey = &scpb.APIKeyRequirement{
				AllowWithoutApiKey: true,
			}
		}

		filterConfig.Requirements = append(filterConfig.Requirements, requirement)
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
	for _, operation := range serviceInfo.Operations {
		method := serviceInfo.Methods[operation]
		if method.BackendRule.JwtAudience == "" {
			continue
		}
		rules = append(rules,
			&bapb.BackendAuthRule{
				Operation:    operation,
				JwtAudience:  method.BackendRule.JwtAudience,
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
	for _, operation := range serviceInfo.Operations {
		method := serviceInfo.Methods[operation]
		if method.BackendRule.TranslationType != conf.BackendRule_PATH_TRANSLATION_UNSPECIFIED {
			rules = append(rules, &brpb.BackendRoutingRule{
				Operation:      operation,
				IsConstAddress: method.BackendRule.TranslationType == conf.BackendRule_CONSTANT_ADDRESS,
				PathPrefix:     method.BackendRule.Uri,
			})
		}
	}

	backendRoutingConfig := &brpb.FilterConfig{Rules: rules}
	backendRoutingConfigStruct, _ := util.MessageToStruct(backendRoutingConfig)
	backendRoutingFilter := &hcm.HttpFilter{
		Name:       ut.BackendRouting,
		ConfigType: &hcm.HttpFilter_Config{backendRoutingConfigStruct},
	}
	return backendRoutingFilter
}

func makeRouterFilter() *hcm.HttpFilter {
	router, _ := util.MessageToStruct(&rt.Router{
		SuppressEnvoyHeaders: *flags.SuppressEnvoyHeaders,
	})
	routerFilter := &hcm.HttpFilter{
		Name:       util.Router,
		ConfigType: &hcm.HttpFilter_Config{router},
	}
	return routerFilter
}
