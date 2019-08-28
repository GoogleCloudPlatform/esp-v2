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

	"cloudesf.googlesource.com/gcpproxy/src/go/metadata"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/wrappers"

	sc "cloudesf.googlesource.com/gcpproxy/src/go/configinfo"
	bapb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/backend_auth"
	brpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/backend_routing"
	commonpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/common"
	pmpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/path_matcher"
	scpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/service_control"
	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
	addresspb "github.com/envoyproxy/data-plane-api/api/address"
	hcm "github.com/envoyproxy/data-plane-api/api/http_connection_manager"
	httpuripb "github.com/envoyproxy/data-plane-api/api/http_uri"
	ac "github.com/envoyproxy/data-plane-api/api/jwt_authn"
	ldspb "github.com/envoyproxy/data-plane-api/api/lds"
	listenerpb "github.com/envoyproxy/data-plane-api/api/listener"
	rt "github.com/envoyproxy/data-plane-api/api/router"
	tc "github.com/envoyproxy/data-plane-api/api/transcoder"
	structpb "github.com/golang/protobuf/ptypes/struct"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
	sm "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

const (
	statPrefix = "ingress_http"
)

// MakeListener provides a dynamic listener for Envoy
func MakeListener(serviceInfo *sc.ServiceInfo) (*ldspb.Listener, error) {
	httpFilters := []*hcm.HttpFilter{}

	if serviceInfo.Options.CorsPreset == "basic" || serviceInfo.Options.CorsPreset == "cors_with_regex" {
		corsFilter := &hcm.HttpFilter{
			Name: ut.CORS,
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
	if !serviceInfo.Options.SkipJwtAuthnFilter {
		jwtAuthnFilter := makeJwtAuthnFilter(serviceInfo)
		if jwtAuthnFilter != nil {
			httpFilters = append(httpFilters, jwtAuthnFilter)
			glog.Infof("adding JWT Authn Filter config: %v", jwtAuthnFilter)
		}
	}

	// Add Service Control filter if needed.
	if !serviceInfo.Options.SkipServiceControlFilter {
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
			Name:       ut.GRPCWeb,
			ConfigType: &hcm.HttpFilter_Config{Config: &structpb.Struct{}},
		}
		httpFilters = append(httpFilters, grpcWebFilter)
	}

	// Add Backend Auth filter and Backend Routing if needed.
	if serviceInfo.Options.EnableBackendRouting {
		if serviceInfo.Options.ServiceAccountKey != "" {
			return nil, fmt.Errorf("ServiceAccountKey is set(proxy runs on Non-GCP) while backendRouting is not allowed on Non-GCP")
		}
		backendAuthFilter := makeBackendAuthFilter(serviceInfo)
		httpFilters = append(httpFilters, backendAuthFilter)
		glog.Infof("adding Backend Auth Filter config: %v", backendAuthFilter)
		backendRoutingFilter := makeBackendRoutingFilter(serviceInfo)
		httpFilters = append(httpFilters, backendRoutingFilter)
		glog.Infof("adding Backend Routing Filter config: %v", backendRoutingFilter)
	}

	// Add Envoy Router filter so requests are routed upstream.
	// Router filter should be the last.

	routerFilter := makeRouterFilter(serviceInfo.Options)
	httpFilters = append(httpFilters, routerFilter)

	route, err := MakeRouteConfig(serviceInfo)

	if err != nil {
		return nil, fmt.Errorf("makeHttpConnectionManagerRouteConfig got err: %s", err)
	}

	httpConMgr := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: statPrefix,
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: route,
		},

		UseRemoteAddress:  &wrappers.BoolValue{Value: serviceInfo.Options.EnvoyUseRemoteAddress},
		XffNumTrustedHops: uint32(serviceInfo.Options.EnvoyXffNumTrustedHops),
	}
	if serviceInfo.Options.EnableTracing {
		httpConMgr.Tracing = &hcm.HttpConnectionManager_Tracing{}
	}

	glog.Infof("adding Http Connection Manager config: %v", httpConMgr)
	httpConMgr.HttpFilters = httpFilters

	// HTTP filter configuration
	httpFilterConfig, err := ut.MessageToStruct(httpConMgr)
	if err != nil {
		return nil, err
	}

	return &ldspb.Listener{
		Address: &addresspb.Address{
			Address: &addresspb.Address_SocketAddress{
				SocketAddress: &addresspb.SocketAddress{
					Address: serviceInfo.Options.ListenerAddress,
					PortSpecifier: &addresspb.SocketAddress_PortValue{
						PortValue: uint32(serviceInfo.Options.ListenerPort),
					},
				},
			},
		},
		FilterChains: []*listenerpb.FilterChain{
			{
				Filters: []*listenerpb.Filter{
					{
						Name:       ut.HTTPConnectionManager,
						ConfigType: &listenerpb.Filter_Config{httpFilterConfig},
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
		for _, httpRule := range method.HttpRule {
			if httpRule.UriTemplate != "" && httpRule.HttpMethod != "" {
				newHttpRule := &pmpb.PathMatcherRule{
					Operation: operation,
					Pattern:   httpRule,
				}
				if method.BackendRule.TranslationType == conf.BackendRule_CONSTANT_ADDRESS && hasPathParameter(newHttpRule.Pattern.UriTemplate) {
					newHttpRule.ExtractPathParameters = true
				}
				rules = append(rules, newHttpRule)
			}
		}
	}

	if len(rules) == 0 {
		return nil
	}

	pathMathcherConfig := &pmpb.FilterConfig{Rules: rules}
	if len(serviceInfo.SegmentNames) > 0 {
		pathMathcherConfig.SegmentNames = serviceInfo.SegmentNames
	}

	pathMathcherConfigStruct, _ := ut.MessageToStruct(pathMathcherConfig)
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
					HttpUri: &httpuripb.HttpUri{
						Uri: provider.GetJwksUri(),
						HttpUpstreamType: &httpuripb.HttpUri_Cluster{
							Cluster: provider.GetIssuer(),
						},
					},
					CacheDuration: &duration.Duration{
						Seconds: int64(serviceInfo.Options.JwksCacheDurationInS),
					},
				},
			},
			FromHeaders: []*ac.JwtHeader{
				{
					Name:        "Authorization",
					ValuePrefix: "Bearer ",
				},
				{
					Name: "X-Goog-Iap-Jwt-Assertion",
				},
			},
			FromParams: []string{
				"access_token",
			},
			ForwardPayloadHeader: "X-Endpoint-API-UserInfo",
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

	jas, _ := ut.MessageToStruct(jwtAuthentication)
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

func makeServiceControlCallingConfig(options sc.EnvoyConfigOptions) *scpb.ServiceControlCallingConfig {
	setting := &scpb.ServiceControlCallingConfig{}
	setting.NetworkFailOpen = &wrappers.BoolValue{Value: options.ServiceControlNetworkFailOpen}

	if options.ScCheckTimeoutMs > 0 {
		setting.CheckTimeoutMs = &wrappers.UInt32Value{Value: uint32(options.ScCheckTimeoutMs)}
	}
	if options.ScQuotaTimeoutMs > 0 {
		setting.QuotaTimeoutMs = &wrappers.UInt32Value{Value: uint32(options.ScQuotaTimeoutMs)}
	}
	if options.ScReportTimeoutMs > 0 {
		setting.ReportTimeoutMs = &wrappers.UInt32Value{Value: uint32(options.ScReportTimeoutMs)}
	}

	if options.ScCheckRetries > -1 {
		setting.CheckRetries = &wrappers.UInt32Value{Value: uint32(options.ScCheckRetries)}
	}
	if options.ScQuotaRetries > -1 {
		setting.QuotaRetries = &wrappers.UInt32Value{Value: uint32(options.ScQuotaRetries)}
	}
	if options.ScReportRetries > -1 {
		setting.ReportRetries = &wrappers.UInt32Value{Value: uint32(options.ScReportRetries)}
	}
	return setting
}

func makeServiceControlFilter(serviceInfo *sc.ServiceInfo) *hcm.HttpFilter {
	if serviceInfo == nil || serviceInfo.ServiceConfig().GetControl().GetEnvironment() == "" {
		return nil
	}

	lowercaseProtocol := strings.ToLower(serviceInfo.Options.BackendProtocol)
	serviceName := serviceInfo.ServiceConfig().GetName()
	service := &scpb.Service{
		ServiceName:       serviceName,
		ServiceConfigId:   serviceInfo.ConfigID,
		ProducerProjectId: serviceInfo.ServiceConfig().GetProducerProjectId(),
		ServiceConfig:     copyServiceConfigForReportMetrics(serviceInfo.ServiceConfig()),
		BackendProtocol:   lowercaseProtocol,
	}

	if serviceInfo.Options.LogRequestHeaders != "" {
		service.LogRequestHeaders = strings.Split(serviceInfo.Options.LogRequestHeaders, ",")
		for i := range service.LogRequestHeaders {
			service.LogRequestHeaders[i] = strings.TrimSpace(service.LogRequestHeaders[i])
		}
	}
	if serviceInfo.Options.LogResponseHeaders != "" {
		service.LogResponseHeaders = strings.Split(serviceInfo.Options.LogResponseHeaders, ",")
		for i := range service.LogResponseHeaders {
			service.LogResponseHeaders[i] = strings.TrimSpace(service.LogResponseHeaders[i])
		}
	}
	if serviceInfo.Options.LogJwtPayloads != "" {
		service.LogJwtPayloads = strings.Split(serviceInfo.Options.LogJwtPayloads, ",")
		for i := range service.LogJwtPayloads {
			service.LogJwtPayloads[i] = strings.TrimSpace(service.LogJwtPayloads[i])
		}
	}
	service.JwtPayloadMetadataName = ut.JwtPayloadMetadataName

	filterConfig := &scpb.FilterConfig{
		Services:        []*scpb.Service{service},
		AccessToken:     serviceInfo.AccessToken,
		ScCallingConfig: makeServiceControlCallingConfig(serviceInfo.Options),
		ServiceControlUri: &commonpb.HttpUri{
			Uri:     serviceInfo.ServiceControlURI,
			Cluster: ut.ServiceControlClusterName,
			Timeout: &duration.Duration{Seconds: 5},
		},
	}

	if serviceInfo.GcpAttributes != nil {
		filterConfig.GcpAttributes = serviceInfo.GcpAttributes
	}

	for _, operation := range serviceInfo.Operations {
		method := serviceInfo.Methods[operation]
		requirement := &scpb.Requirement{
			ServiceName:        serviceName,
			OperationName:      operation,
			SkipServiceControl: method.SkipServiceControl,
			MetricCosts:        method.MetricCosts,
		}

		// For these OPTIONS methods, auth should be disabled and AllowWithoutApiKey
		// should be true for each CORS.
		if method.IsGeneratedOption || method.AllowUnregisteredCalls {
			requirement.ApiKey = &scpb.APIKeyRequirement{
				AllowWithoutApiKey: true,
			}
		}

		if method.APIKeyLocations != nil {
			if requirement.ApiKey == nil {
				requirement.ApiKey = &scpb.APIKeyRequirement{}
			}
			requirement.ApiKey.Locations = method.APIKeyLocations
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
			transcodeConfigStruct, _ := ut.MessageToStruct(transcodeConfig)
			transcodeFilter := &hcm.HttpFilter{
				Name:       ut.GRPCJSONTranscoder,
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
				Operation:   operation,
				JwtAudience: method.BackendRule.JwtAudience,
			})
	}
	backendAuthConfig := &bapb.FilterConfig{
		Rules: rules,
		AccessToken: &commonpb.AccessToken{
			TokenType: &commonpb.AccessToken_RemoteToken{
				RemoteToken: &commonpb.HttpUri{
					Uri:     fmt.Sprintf("%s%s", *metadata.MetadataURL, ut.IdentityTokenSuffix),
					Cluster: ut.MetadataServerClusterName,
					// TODO(taoxuy): make token_subscriber use this timeout
					Timeout: &duration.Duration{Seconds: 5},
				},
			},
		},
	}
	backendAuthConfigStruct, _ := ut.MessageToStruct(backendAuthConfig)
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
	backendRoutingConfigStruct, _ := ut.MessageToStruct(backendRoutingConfig)
	backendRoutingFilter := &hcm.HttpFilter{
		Name:       ut.BackendRouting,
		ConfigType: &hcm.HttpFilter_Config{backendRoutingConfigStruct},
	}
	return backendRoutingFilter
}

func makeRouterFilter(options sc.EnvoyConfigOptions) *hcm.HttpFilter {
	router, _ := ut.MessageToStruct(&rt.Router{
		SuppressEnvoyHeaders: options.SuppressEnvoyHeaders,
		StartChildSpan:       options.EnableTracing,
	})
	routerFilter := &hcm.HttpFilter{
		Name:       ut.Router,
		ConfigType: &hcm.HttpFilter_Config{Config: router},
	}
	return routerFilter
}
