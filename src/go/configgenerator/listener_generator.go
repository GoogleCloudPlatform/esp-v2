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
	"sort"
	"strings"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	sc "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	bapb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/backend_auth"
	brpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/backend_routing"
	commonpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/common"
	pmpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/path_matcher"
	scpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/service_control"
	v2pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	listenerpb "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	routepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	acpb "github.com/envoyproxy/go-control-plane/envoy/config/filter/accesslog/v2"
	gspb "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/grpc_stats/v2alpha"
	hcpb "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/health_check/v2"
	jwtpb "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/jwt_authn/v2alpha"
	routerpb "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/router/v2"
	transcoderpb "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/transcoder/v2"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	facpb "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	anypb "github.com/golang/protobuf/ptypes/any"
	durationpb "github.com/golang/protobuf/ptypes/duration"
	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

const (
	statPrefix = "ingress_http"
)

// MakeListeners provides dynamic listeners for Envoy
func MakeListeners(serviceInfo *sc.ServiceInfo) ([]*v2pb.Listener, error) {
	listener, err := makeListener(serviceInfo)
	if err != nil {
		return nil, err
	}
	return []*v2pb.Listener{listener}, nil
}

// makeListener provides a dynamic listener for Envoy
func makeListener(serviceInfo *sc.ServiceInfo) (*v2pb.Listener, error) {
	httpFilters := []*hcmpb.HttpFilter{}

	if serviceInfo.Options.CorsPreset == "basic" || serviceInfo.Options.CorsPreset == "cors_with_regex" {
		corsFilter := &hcmpb.HttpFilter{
			Name: util.CORS,
		}
		httpFilters = append(httpFilters, corsFilter)
		jsonStr, _ := util.ProtoToJson(corsFilter)
		glog.Infof("adding CORS Filter config: %v", jsonStr)
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
		jsonStr, _ := util.ProtoToJson(pathMathcherFilter)
		glog.Infof("adding Path Matcher Filter config: %v", jsonStr)
	}

	// Add Health Check filter if needed. It must behind Path Matcher filter, since Service Control
	// filter needs to get the corresponding rule for health check calls, in order to skip Report
	if serviceInfo.Options.Healthz != "" {
		hcFilter, err := makeHealthCheckFilter(serviceInfo)
		if err != nil {
			return nil, err
		}
		httpFilters = append(httpFilters, hcFilter)
		jsonStr, _ := util.ProtoToJson(hcFilter)
		glog.V(1).Infof("adding Healthz filter config: %v", jsonStr)
	}

	// Add JWT Authn filter if needed.
	if !serviceInfo.Options.SkipJwtAuthnFilter {
		jwtAuthnFilter := makeJwtAuthnFilter(serviceInfo)
		if jwtAuthnFilter != nil {
			httpFilters = append(httpFilters, jwtAuthnFilter)
			jsonStr, _ := util.ProtoToJson(jwtAuthnFilter)
			glog.Infof("adding JWT Authn Filter config: %v", jsonStr)
		}
	}

	// Add Service Control filter if needed.
	if !serviceInfo.Options.SkipServiceControlFilter {
		serviceControlFilter := makeServiceControlFilter(serviceInfo)
		if serviceControlFilter != nil {
			httpFilters = append(httpFilters, serviceControlFilter)
			jsonStr, _ := util.ProtoToJson(serviceControlFilter)
			glog.Infof("adding Service Control Filter config: %v", jsonStr)
		}
	}

	// Add gRPC Transcoder filter and gRPCWeb filter configs for gRPC backend.
	if serviceInfo.GrpcSupportRequired {
		transcoderFilter := makeTranscoderFilter(serviceInfo)
		if transcoderFilter != nil {
			httpFilters = append(httpFilters, transcoderFilter)
			jsonStr, _ := util.ProtoToJson(transcoderFilter)
			glog.Infof("adding Transcoder Filter config: %v", jsonStr)
		}

		grpcWebFilter := &hcmpb.HttpFilter{
			Name: util.GRPCWeb,
		}
		httpFilters = append(httpFilters, grpcWebFilter)

		// GrpcStats filter is used to count gRPC frames.
		// The data is stored in filterState and used by ServiceControl
		// filter in the final report call.
		httpFilters = append(httpFilters, makeGrpcStatsFilter())
	}

	// Add Backend Auth filter and Backend Routing if needed.
	backendAuthFilter := makeBackendAuthFilter(serviceInfo)
	if backendAuthFilter != nil {
		httpFilters = append(httpFilters, backendAuthFilter)
		jsonStr, _ := util.ProtoToJson(backendAuthFilter)
		glog.Infof("adding Backend Auth Filter config: %v", jsonStr)
	}

	backendRoutingFilter, err := makeBackendRoutingFilter(serviceInfo)
	if err != nil {
		return nil, err
	}

	if backendRoutingFilter != nil {
		httpFilters = append(httpFilters, backendRoutingFilter)
		jsonStr, _ := util.ProtoToJson(backendRoutingFilter)
		glog.Infof("adding Backend Routing Filter config: %v", jsonStr)
	}

	// Add Envoy Router filter so requests are routed upstream.
	// Router filter should be the last.
	routerFilter := makeRouterFilter(serviceInfo.Options)
	httpFilters = append(httpFilters, routerFilter)

	route, err := MakeRouteConfig(serviceInfo)

	if err != nil {
		return nil, fmt.Errorf("makeHttpConnectionManagerRouteConfig got err: %s", err)
	}

	httpConMgr := makeHttpConMgr(&serviceInfo.Options, route)

	jsonStr, _ := util.ProtoToJson(httpConMgr)
	glog.Infof("adding Http Connection Manager config: %v", jsonStr)
	httpConMgr.HttpFilters = httpFilters

	// HTTP filter configuration
	httpFilterConfig, err := ptypes.MarshalAny(httpConMgr)
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

	listenerName := "http_listener"

	if serviceInfo.Options.SslServerCertPath != "" {
		listenerName = "https_listener"
		transportSocket, err := util.CreateDownstreamTransportSocket(
			serviceInfo.Options.SslServerCertPath,
			serviceInfo.Options.SslMinimumProtocol,
			serviceInfo.Options.SslMaximumProtocol,
		)
		if err != nil {
			return nil, err
		}
		filterChain.TransportSocket = transportSocket
	}

	return &v2pb.Listener{
		Name: listenerName,
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
	}, nil
}

func makeHttpConMgr(opts *options.ConfigGeneratorOptions, route *v2pb.RouteConfiguration) *hcmpb.HttpConnectionManager {
	httpConMgr := &hcmpb.HttpConnectionManager{
		UpgradeConfigs: []*hcmpb.HttpConnectionManager_UpgradeConfig{
			{
				UpgradeType: "websocket",
			},
		},
		CodecType:  hcmpb.HttpConnectionManager_AUTO,
		StatPrefix: statPrefix,
		RouteSpecifier: &hcmpb.HttpConnectionManager_RouteConfig{
			RouteConfig: route,
		},
		UseRemoteAddress:  &wrapperspb.BoolValue{Value: opts.EnvoyUseRemoteAddress},
		XffNumTrustedHops: uint32(opts.EnvoyXffNumTrustedHops),
	}

	if opts.AccessLog != "" {
		fileAccessLog := &facpb.FileAccessLog{
			Path: opts.AccessLog,
		}

		if opts.AccessLogFormat != "" {
			fileAccessLog.AccessLogFormat = &facpb.FileAccessLog_Format{
				Format: opts.AccessLogFormat,
			}
		}

		serialized, _ := ptypes.MarshalAny(fileAccessLog)

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
		httpConMgr.Tracing = &hcmpb.HttpConnectionManager_Tracing{}
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

	return httpConMgr
}

func makePathMatcherFilter(serviceInfo *sc.ServiceInfo) *hcmpb.HttpFilter {
	rules := []*pmpb.PathMatcherRule{}
	for _, operation := range serviceInfo.Operations {
		method := serviceInfo.Methods[operation]
		// Adds PathMatcherRule for HTTP method, whose HttpRule is not empty.
		for _, httpRule := range method.HttpRule {
			if httpRule.UriTemplate != "" && httpRule.HttpMethod != "" {
				newHttpRule := &pmpb.PathMatcherRule{
					Operation: operation,
					Pattern:   httpRule,
				}
				if method.BackendInfo != nil && method.BackendInfo.TranslationType == confpb.BackendRule_CONSTANT_ADDRESS && hasPathParameter(newHttpRule.Pattern.UriTemplate) {
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

	pathMathcherConfigStruct, _ := ptypes.MarshalAny(pathMathcherConfig)
	pathMatcherFilter := &hcmpb.HttpFilter{
		Name:       util.PathMatcher,
		ConfigType: &hcmpb.HttpFilter_TypedConfig{pathMathcherConfigStruct},
	}
	return pathMatcherFilter
}

func makeGrpcStatsFilter() *hcmpb.HttpFilter {
	cfg := &gspb.FilterConfig{
		EmitFilterState: true,
	}
	cfg_any, _ := ptypes.MarshalAny(cfg)

	return &hcmpb.HttpFilter{
		Name:       util.GrpcStatsFilterName,
		ConfigType: &hcmpb.HttpFilter_TypedConfig{cfg_any},
	}
}

func hasPathParameter(httpPattern string) bool {
	return strings.ContainsRune(httpPattern, '{')
}

func defaultJwtLocations() ([]*jwtpb.JwtHeader, []string) {
	return []*jwtpb.JwtHeader{
			{
				Name:        util.DefaultJwtHeaderNameAuthorization,
				ValuePrefix: util.DefaultJwtHeaderValuePrefixBearer,
			},
			{
				Name: util.DefaultJwtHeaderNameXGoogleIapJwtAssertion,
			},
		}, []string{
			util.DefaultJwtQueryParamAccessToken,
		}
}

func processJwtLocations(provider *confpb.AuthProvider) ([]*jwtpb.JwtHeader, []string) {
	if len(provider.JwtLocations) == 0 {
		return defaultJwtLocations()
	}

	jwtHeaders := []*jwtpb.JwtHeader{}
	jwtParams := []string{}

	for _, jwtLocation := range provider.JwtLocations {
		switch x := jwtLocation.In.(type) {
		case *confpb.JwtLocation_Header:
			jwtHeaders = append(jwtHeaders, &jwtpb.JwtHeader{
				Name:        jwtLocation.GetHeader(),
				ValuePrefix: jwtLocation.GetValuePrefix(),
			})
		case *confpb.JwtLocation_Query:
			jwtParams = append(jwtParams, jwtLocation.GetQuery())
		default:
			fmt.Errorf("provider.JwtLocations as unexpected type %T", x)
		}
	}
	return jwtHeaders, jwtParams
}

func makeJwtAuthnFilter(serviceInfo *sc.ServiceInfo) *hcmpb.HttpFilter {
	auth := serviceInfo.ServiceConfig().GetAuthentication()
	if len(auth.GetProviders()) == 0 {
		return nil
	}
	providers := make(map[string]*jwtpb.JwtProvider)
	for _, provider := range auth.GetProviders() {
		clusterName, err := util.ExtraAddressFromURI(provider.GetJwksUri())
		if err != nil {
			return nil
		}

		fromHeaders, fromParams := processJwtLocations(provider)

		jp := &jwtpb.JwtProvider{
			Issuer: provider.GetIssuer(),
			JwksSourceSpecifier: &jwtpb.JwtProvider_RemoteJwks{
				RemoteJwks: &jwtpb.RemoteJwks{
					HttpUri: &corepb.HttpUri{
						Uri: provider.GetJwksUri(),
						HttpUpstreamType: &corepb.HttpUri_Cluster{
							Cluster: clusterName,
						},
						Timeout: ptypes.DurationProto(serviceInfo.Options.HttpRequestTimeout),
					},
					CacheDuration: &durationpb.Duration{
						Seconds: int64(serviceInfo.Options.JwksCacheDurationInS),
					},
				},
			},
			FromHeaders:          fromHeaders,
			FromParams:           fromParams,
			ForwardPayloadHeader: "X-Endpoint-API-UserInfo",
			Forward:              true,
		}

		if len(provider.GetAudiences()) != 0 {
			for _, a := range strings.Split(provider.GetAudiences(), ",") {
				jp.Audiences = append(jp.Audiences, strings.TrimSpace(a))
			}
		} else {
			// No providers specified by user.
			// For backwards-compatibility with ESPv1, auto-generate audiences.
			// See b/147834348 for more information on this default behavior.
			defaultAudience := fmt.Sprintf("https://%v", serviceInfo.Name)
			jp.Audiences = append(jp.Audiences, defaultAudience)
		}

		// TODO(taoxuy): add unit test
		// the JWT Payload will be send to metadata by envoy and it will be used by service control filter
		// for logging and setting credential_id
		jp.PayloadInMetadata = util.JwtPayloadMetadataName
		providers[provider.GetId()] = jp
	}

	if len(providers) == 0 {
		return nil
	}

	requirements := make(map[string]*jwtpb.JwtRequirement)
	for _, rule := range auth.GetRules() {
		if len(rule.GetRequirements()) > 0 {
			requirements[rule.GetSelector()] = makeJwtRequirement(rule.GetRequirements())
		}
	}

	jwtAuthentication := &jwtpb.JwtAuthentication{
		Providers: providers,
		FilterStateRules: &jwtpb.FilterStateRule{
			Name:     "envoy.filters.http.path_matcher.operation",
			Requires: requirements,
		},
	}

	jas, _ := ptypes.MarshalAny(jwtAuthentication)
	jwtAuthnFilter := &hcmpb.HttpFilter{
		Name:       util.JwtAuthn,
		ConfigType: &hcmpb.HttpFilter_TypedConfig{jas},
	}
	return jwtAuthnFilter
}

func makeJwtRequirement(requirements []*confpb.AuthRequirement) *jwtpb.JwtRequirement {
	// By default, if there are multi requirements, treat it as RequireAny.
	requires := &jwtpb.JwtRequirement{
		RequiresType: &jwtpb.JwtRequirement_RequiresAny{
			RequiresAny: &jwtpb.JwtRequirementOrList{},
		},
	}

	for _, r := range requirements {
		var require *jwtpb.JwtRequirement
		if r.GetAudiences() == "" {
			require = &jwtpb.JwtRequirement{
				RequiresType: &jwtpb.JwtRequirement_ProviderName{
					ProviderName: r.GetProviderId(),
				},
			}
		} else {
			// Note: Audiences in requirements is deprecated.
			// But if it's specified, we should override the audiences for the provider.
			var audiences []string
			for _, a := range strings.Split(r.GetAudiences(), ",") {
				audiences = append(audiences, strings.TrimSpace(a))
			}
			require = &jwtpb.JwtRequirement{
				RequiresType: &jwtpb.JwtRequirement_ProviderAndAudiences{
					ProviderAndAudiences: &jwtpb.ProviderWithAudiences{
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

func makeServiceControlCallingConfig(opts options.ConfigGeneratorOptions) *scpb.ServiceControlCallingConfig {
	setting := &scpb.ServiceControlCallingConfig{}
	setting.NetworkFailOpen = &wrapperspb.BoolValue{Value: opts.ServiceControlNetworkFailOpen}

	if opts.ScCheckTimeoutMs > 0 {
		setting.CheckTimeoutMs = &wrapperspb.UInt32Value{Value: uint32(opts.ScCheckTimeoutMs)}
	}
	if opts.ScQuotaTimeoutMs > 0 {
		setting.QuotaTimeoutMs = &wrapperspb.UInt32Value{Value: uint32(opts.ScQuotaTimeoutMs)}
	}
	if opts.ScReportTimeoutMs > 0 {
		setting.ReportTimeoutMs = &wrapperspb.UInt32Value{Value: uint32(opts.ScReportTimeoutMs)}
	}

	if opts.ScCheckRetries > -1 {
		setting.CheckRetries = &wrapperspb.UInt32Value{Value: uint32(opts.ScCheckRetries)}
	}
	if opts.ScQuotaRetries > -1 {
		setting.QuotaRetries = &wrapperspb.UInt32Value{Value: uint32(opts.ScQuotaRetries)}
	}
	if opts.ScReportRetries > -1 {
		setting.ReportRetries = &wrapperspb.UInt32Value{Value: uint32(opts.ScReportRetries)}
	}
	return setting
}

func makeServiceControlFilter(serviceInfo *sc.ServiceInfo) *hcmpb.HttpFilter {
	if serviceInfo == nil || serviceInfo.ServiceConfig().GetControl().GetEnvironment() == "" {
		return nil
	}

	// TODO(b/148638212): Clean up this hacky way of specifying the protocol for Service Control report.
	// This is safe (for now) as our Service Control filter only differentiates between gRPC or non-gRPC.
	var protocol string
	if serviceInfo.GrpcSupportRequired {
		protocol = "grpc"
	} else {
		// TODO(b/148638212): Must be http1 (not http) for current filter implementation.
		protocol = "http1"
	}

	serviceName := serviceInfo.ServiceConfig().GetName()
	service := &scpb.Service{
		ServiceName:       serviceName,
		ServiceConfigId:   serviceInfo.ConfigID,
		ProducerProjectId: serviceInfo.ServiceConfig().GetProducerProjectId(),
		ServiceConfig:     copyServiceConfigForReportMetrics(serviceInfo.ServiceConfig()),
		BackendProtocol:   protocol,
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
	if serviceInfo.Options.MinStreamReportIntervalMs != 0 {
		service.MinStreamReportIntervalMs = serviceInfo.Options.MinStreamReportIntervalMs
	}
	service.JwtPayloadMetadataName = util.JwtPayloadMetadataName

	filterConfig := &scpb.FilterConfig{
		Services:        []*scpb.Service{service},
		ScCallingConfig: makeServiceControlCallingConfig(serviceInfo.Options),
		ServiceControlUri: &commonpb.HttpUri{
			Uri:     serviceInfo.ServiceControlURI,
			Cluster: util.ServiceControlClusterName,
			Timeout: ptypes.DurationProto(serviceInfo.Options.HttpRequestTimeout),
		},
	}

	if serviceInfo.Options.ServiceControlCredentials != nil {
		// Use access token fetched from Google Cloud IAM Server to talk to Service Controller
		filterConfig.AccessToken = &scpb.FilterConfig_IamToken{
			IamToken: &commonpb.IamTokenInfo{
				IamUri: &commonpb.HttpUri{
					Uri:     fmt.Sprintf("%s%s", serviceInfo.Options.IamURL, util.IamAccessTokenSuffix(serviceInfo.Options.ServiceControlCredentials.ServiceAccountEmail)),
					Cluster: util.IamServerClusterName,
					Timeout: ptypes.DurationProto(serviceInfo.Options.HttpRequestTimeout),
				},
				ServiceAccountEmail: serviceInfo.Options.ServiceControlCredentials.ServiceAccountEmail,
				Delegates:           serviceInfo.Options.ServiceControlCredentials.Delegates,
				AccessToken:         serviceInfo.AccessToken,
			},
		}
	} else {
		// Use access token from fetched the Instance Metadata Server to talk to Service Controller
		switch serviceInfo.AccessToken.TokenType.(type) {
		case *commonpb.AccessToken_RemoteToken:
			filterConfig.AccessToken = &scpb.FilterConfig_ImdsToken{
				ImdsToken: serviceInfo.AccessToken.GetRemoteToken(),
			}
			break
		case *commonpb.AccessToken_ServiceAccountSecret:
			filterConfig.AccessToken = &scpb.FilterConfig_ServiceAccountSecret{
				ServiceAccountSecret: serviceInfo.AccessToken.GetServiceAccountSecret(),
			}
			break
		default:
			break
		}
	}

	if serviceInfo.GcpAttributes != nil {
		filterConfig.GcpAttributes = serviceInfo.GcpAttributes
	}
	if serviceInfo.Options.ComputePlatformOverride != "" {
		if filterConfig.GcpAttributes == nil {
			filterConfig.GcpAttributes = &scpb.GcpAttributes{}
		}
		filterConfig.GcpAttributes.Platform = serviceInfo.Options.ComputePlatformOverride
	}

	for _, operation := range serviceInfo.Operations {
		method := serviceInfo.Methods[operation]
		requirement := &scpb.Requirement{
			ServiceName:        serviceName,
			OperationName:      operation,
			ApiName:            method.ApiName,
			ApiVersion:         method.ApiVersion,
			SkipServiceControl: method.SkipServiceControl,
			MetricCosts:        method.MetricCosts,
		}

		// For these OPTIONS methods, auth should be disabled and AllowWithoutApiKey
		// should be true for each CORS.
		if method.IsGenerated || method.AllowUnregisteredCalls {
			requirement.ApiKey = &scpb.ApiKeyRequirement{
				AllowWithoutApiKey: true,
			}
		}

		if method.ApiKeyLocations != nil {
			if requirement.ApiKey == nil {
				requirement.ApiKey = &scpb.ApiKeyRequirement{}
			}
			requirement.ApiKey.Locations = method.ApiKeyLocations
		}

		filterConfig.Requirements = append(filterConfig.Requirements, requirement)
	}

	scs, err := ptypes.MarshalAny(filterConfig)
	if err != nil {
		glog.Warningf("failed to convert message to struct: %v", err)
	}
	filter := &hcmpb.HttpFilter{
		Name:       util.ServiceControl,
		ConfigType: &hcmpb.HttpFilter_TypedConfig{scs},
	}
	return filter
}

func copyServiceConfigForReportMetrics(src *confpb.Service) *anypb.Any {
	// Logs and metrics fields are needed by the Envoy HTTP filter
	// to generate proper Metrics for Report calls.
	serviceConfigCopy := &confpb.Service{
		Logs:               src.GetLogs(),
		Metrics:            src.GetMetrics(),
		MonitoredResources: src.GetMonitoredResources(),
		Monitoring:         src.GetMonitoring(),
		Logging:            src.GetLogging(),
	}
	a, err := ptypes.MarshalAny(serviceConfigCopy)
	if err != nil {
		glog.Warningf("failed to copy certain service config, error: %v", err)
		return nil
	}
	return a
}

func makeTranscoderFilter(serviceInfo *sc.ServiceInfo) *hcmpb.HttpFilter {
	for _, sourceFile := range serviceInfo.ServiceConfig().GetSourceInfo().GetSourceFiles() {
		configFile := &smpb.ConfigFile{}
		ptypes.UnmarshalAny(sourceFile, configFile)

		if configFile.GetFileType() == smpb.ConfigFile_FILE_DESCRIPTOR_SET_PROTO {
			ignoredQueryParameterList := []string{}
			for IgnoredQueryParameter, _ := range serviceInfo.AllTranscodingIgnoredQueryParams {
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
				PrintOptions: &transcoderpb.GrpcJsonTranscoder_PrintOptions{
					AlwaysPrintPrimitiveFields: serviceInfo.Options.TranscodingAlwaysPrintPrimitiveFields,
					AlwaysPrintEnumsAsInts:     serviceInfo.Options.TranscodingAlwaysPrintEnumsAsInts,
					PreserveProtoFieldNames:    serviceInfo.Options.TranscodingPreserveProtoFieldNames,
				},
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

func makeBackendAuthFilter(serviceInfo *sc.ServiceInfo) *hcmpb.HttpFilter {
	var rules []*bapb.BackendAuthRule
	for _, operation := range serviceInfo.Operations {
		method := serviceInfo.Methods[operation]
		if method.BackendInfo == nil || method.BackendInfo.JwtAudience == "" {
			continue
		}
		rules = append(rules,
			&bapb.BackendAuthRule{
				Operation:   operation,
				JwtAudience: method.BackendInfo.JwtAudience,
			})
	}
	// If none of BackendRules need auth, rules will be empty, not need to add the filter.
	if len(rules) == 0 {
		return nil
	}

	backendAuthConfig := &bapb.FilterConfig{
		Rules: rules,
	}
	if serviceInfo.Options.BackendAuthCredentials != nil {
		backendAuthConfig.IdTokenInfo = &bapb.FilterConfig_IamToken{
			IamToken: &commonpb.IamTokenInfo{
				IamUri: &commonpb.HttpUri{
					Uri:     fmt.Sprintf("%s%s", serviceInfo.Options.IamURL, util.IamIdentityTokenSuffix(serviceInfo.Options.BackendAuthCredentials.ServiceAccountEmail)),
					Cluster: util.IamServerClusterName,
					Timeout: ptypes.DurationProto(serviceInfo.Options.HttpRequestTimeout),
				},
				// Currently only support fetching access token from instance metadata
				// server, not by service account file.
				AccessToken:         serviceInfo.AccessToken,
				ServiceAccountEmail: serviceInfo.Options.BackendAuthCredentials.ServiceAccountEmail,
				Delegates:           serviceInfo.Options.BackendAuthCredentials.Delegates,
			}}
	} else {
		backendAuthConfig.IdTokenInfo = &bapb.FilterConfig_ImdsToken{
			ImdsToken: &commonpb.HttpUri{
				Uri:     fmt.Sprintf("%s%s", serviceInfo.Options.MetadataURL, util.IdentityTokenSuffix),
				Cluster: util.MetadataServerClusterName,
				Timeout: ptypes.DurationProto(serviceInfo.Options.HttpRequestTimeout),
			},
		}
	}
	backendAuthConfigStruct, _ := ptypes.MarshalAny(backendAuthConfig)
	backendAuthFilter := &hcmpb.HttpFilter{
		Name:       util.BackendAuth,
		ConfigType: &hcmpb.HttpFilter_TypedConfig{backendAuthConfigStruct},
	}
	return backendAuthFilter
}

func makeBackendRoutingFilter(serviceInfo *sc.ServiceInfo) (*hcmpb.HttpFilter, error) {
	rules := []*brpb.BackendRoutingRule{}
	for _, operation := range serviceInfo.Operations {
		method := serviceInfo.Methods[operation]
		if method.BackendInfo != nil && method.BackendInfo.TranslationType != confpb.BackendRule_PATH_TRANSLATION_UNSPECIFIED {
			newRule := &brpb.BackendRoutingRule{
				Operation:      operation,
				IsConstAddress: method.BackendInfo.TranslationType == confpb.BackendRule_CONSTANT_ADDRESS,
			}
			if method.BackendInfo != nil {
				newRule.PathPrefix = method.BackendInfo.Uri
			}
			rules = append(rules, newRule)
		}
	}
	// If none of BackendRules need path translation, rules will be empty, not need to add the filter.
	// For example, grpc backend rules don't need to do path translation.
	if len(rules) == 0 {
		return nil, nil
	}

	backendRoutingConfigStruct, err := ptypes.MarshalAny(&brpb.FilterConfig{Rules: rules})
	if err != nil {
		return nil, err
	}
	return &hcmpb.HttpFilter{
		Name:       util.BackendRouting,
		ConfigType: &hcmpb.HttpFilter_TypedConfig{backendRoutingConfigStruct},
	}, nil
}

func makeHealthCheckFilter(serviceInfo *sc.ServiceInfo) (*hcmpb.HttpFilter, error) {
	hcFilterConfig := &hcpb.HealthCheck{
		PassThroughMode: &wrapperspb.BoolValue{Value: false},

		Headers: []*routepb.HeaderMatcher{
			{
				Name: ":path",
				HeaderMatchSpecifier: &routepb.HeaderMatcher_ExactMatch{
					ExactMatch: serviceInfo.Options.Healthz,
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
