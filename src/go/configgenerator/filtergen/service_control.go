// Copyright 2021 Google LLC
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
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	commonpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/common"
	scpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/service_control"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	"github.com/golang/glog"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	// ServiceControlFilterName is the Envoy filter name for debug logging.
	ServiceControlFilterName = "com.google.espv2.filters.http.service_control"
)

type ServiceControlGenerator struct {
	ServiceName       string
	ServiceConfigID   string
	ProducerProjectID string

	// ServiceConfig is the full OP service config.
	ServiceConfig *confpb.Service

	// GRPCSupportRequired indicates if any backends use gRPC.
	GRPCSupportRequired bool

	ServiceControlURI url.URL
	CallCredentials   *options.IAMCredentialsOptions
	AccessToken       *helpers.FilterAccessTokenConfiger

	// General options below.

	DisableTracing          bool
	TracingProjectID        string
	HttpRequestTimeout      time.Duration
	GeneratedHeaderPrefix   string
	IAMURL                  string
	DependencyErrorBehavior string

	// Service Control filter options below.

	ClientIPFromForwardedHeader bool
	LogRequestHeaders           string
	LogResponseHeaders          string
	LogJwtPayloads              string
	MinStreamReportIntervalMs   uint64
	ComputePlatformOverride     string

	// Service control configs.
	MethodRequirements       []*scpb.Requirement
	CallingConfig            *scpb.ServiceControlCallingConfig
	GCPAttributes            *scpb.GcpAttributes
	EnableApiKeyUidReporting bool

	NoopFilterGenerator
}

// ServiceControlOPFactoryParams are extra params that don't fit within OP
// service config, but needed for construction.
type ServiceControlOPFactoryParams struct {
	GCPAttributes *scpb.GcpAttributes
}

// NewServiceControlFilterGensFromOPConfig creates a ServiceControlGenerator from
// OP service config + descriptor + ESPv2 options.
func NewServiceControlFilterGensFromOPConfig(serviceConfig *confpb.Service, opts options.ConfigGeneratorOptions, params ServiceControlOPFactoryParams) ([]FilterGenerator, error) {
	if opts.SkipServiceControlFilter {
		glog.Infof("Not adding service control (v1) filter gen because the feature is disabled by option.")
		return nil, nil
	}

	if serviceConfig.GetControl().GetEnvironment() == "" {
		glog.Infof("Not adding service control (v1) filter gen because the service control URL is not set in OP config.")
		return nil, nil
	}

	grpcSupportRequired, err := IsGRPCSupportRequiredForOPConfig(serviceConfig, opts)
	if err != nil {
		return nil, err
	}

	scURL, err := ParseServiceControlURLFromOPConfig(serviceConfig, opts)
	if err != nil {
		return nil, err
	}

	requirements, err := MakeMethodRequirementsFromOPConfig(serviceConfig, opts)
	if err != nil {
		return nil, err
	}

	return []FilterGenerator{
		&ServiceControlGenerator{
			ServiceName:                 serviceConfig.GetName(),
			ServiceConfigID:             serviceConfig.GetId(),
			ProducerProjectID:           serviceConfig.GetProducerProjectId(),
			ServiceConfig:               serviceConfig,
			GRPCSupportRequired:         grpcSupportRequired,
			ServiceControlURI:           scURL,
			CallCredentials:             opts.ServiceControlCredentials,
			AccessToken:                 helpers.NewFilterAccessTokenConfigerFromOPConfig(opts),
			DisableTracing:              opts.CommonOptions.TracingOptions.DisableTracing,
			TracingProjectID:            opts.CommonOptions.TracingOptions.ProjectId,
			HttpRequestTimeout:          opts.HttpRequestTimeout,
			GeneratedHeaderPrefix:       opts.GeneratedHeaderPrefix,
			IAMURL:                      opts.IamURL,
			DependencyErrorBehavior:     opts.DependencyErrorBehavior,
			ClientIPFromForwardedHeader: opts.ClientIPFromForwardedHeader,
			LogRequestHeaders:           opts.LogRequestHeaders,
			LogResponseHeaders:          opts.LogResponseHeaders,
			LogJwtPayloads:              opts.LogJwtPayloads,
			MinStreamReportIntervalMs:   opts.MinStreamReportIntervalMs,
			ComputePlatformOverride:     opts.ComputePlatformOverride,
			MethodRequirements:          requirements,
			CallingConfig:               MakeSCCallingConfigFromOPConfig(opts),
			GCPAttributes:               params.GCPAttributes,
			EnableApiKeyUidReporting:    opts.ServiceControlEnableApiKeyUidReporting,
		},
	}, nil
}

func (g *ServiceControlGenerator) FilterName() string {
	return ServiceControlFilterName
}

func (g *ServiceControlGenerator) GenPerRouteConfig(selector string, httpRule *httppattern.Pattern) (proto.Message, error) {
	return &scpb.PerRouteFilterConfig{
		OperationName: selector,
	}, nil
}

func (g *ServiceControlGenerator) GenFilterConfig() (proto.Message, error) {
	// TODO(b/148638212): Clean up this hacky way of specifying the protocol for Service Control report.
	// This is safe (for now) as our Service Control filter only differentiates between gRPC or non-gRPC.
	var protocol string
	if g.GRPCSupportRequired {
		protocol = "grpc"
	} else {
		// TODO(b/148638212): Must be http1 (not http) for current filter implementation.
		protocol = "http1"
	}

	service := &scpb.Service{
		ServiceName:                 g.ServiceName,
		ServiceConfigId:             g.ServiceConfigID,
		ProducerProjectId:           g.ProducerProjectID,
		ServiceConfig:               copyServiceConfigForReportMetrics(g.ServiceConfig),
		BackendProtocol:             protocol,
		ClientIpFromForwardedHeader: g.ClientIPFromForwardedHeader,
		TracingProjectId:            g.TracingProjectID,
		TracingDisabled:             g.DisableTracing,
	}

	if g.LogRequestHeaders != "" {
		service.LogRequestHeaders = strings.Split(g.LogRequestHeaders, ",")
		for i := range service.LogRequestHeaders {
			service.LogRequestHeaders[i] = strings.TrimSpace(service.LogRequestHeaders[i])
		}
	}
	if g.LogResponseHeaders != "" {
		service.LogResponseHeaders = strings.Split(g.LogResponseHeaders, ",")
		for i := range service.LogResponseHeaders {
			service.LogResponseHeaders[i] = strings.TrimSpace(service.LogResponseHeaders[i])
		}
	}
	if g.LogJwtPayloads != "" {
		service.LogJwtPayloads = strings.Split(g.LogJwtPayloads, ",")
		for i := range service.LogJwtPayloads {
			service.LogJwtPayloads[i] = strings.TrimSpace(service.LogJwtPayloads[i])
		}
	}
	if g.MinStreamReportIntervalMs != 0 {
		service.MinStreamReportIntervalMs = g.MinStreamReportIntervalMs
	}
	service.JwtPayloadMetadataName = util.JwtPayloadMetadataName
	filterConfig := &scpb.FilterConfig{
		Services:        []*scpb.Service{service},
		ScCallingConfig: g.CallingConfig,
		ServiceControlUri: &commonpb.HttpUri{
			Uri:     g.ServiceControlURI.String() + "/v1/services",
			Cluster: clustergen.ServiceControlClusterName,
			Timeout: durationpb.New(g.HttpRequestTimeout),
		},
		GeneratedHeaderPrefix:    g.GeneratedHeaderPrefix,
		Requirements:             g.MethodRequirements,
		EnableApiKeyUidReporting: g.EnableApiKeyUidReporting,
	}

	accessTokenConfig := g.AccessToken.MakeAccessTokenConfig()
	if g.CallCredentials != nil {
		// Use access token fetched from Google Cloud IAM Server to talk to Service Controller
		filterConfig.AccessToken = &scpb.FilterConfig_IamToken{
			IamToken: &commonpb.IamTokenInfo{
				IamUri: &commonpb.HttpUri{
					Uri:     fmt.Sprintf("%s%s", g.IAMURL, util.IamAccessTokenPath(g.CallCredentials.ServiceAccountEmail)),
					Cluster: clustergen.IAMServerClusterName,
					Timeout: durationpb.New(g.HttpRequestTimeout),
				},
				ServiceAccountEmail: g.CallCredentials.ServiceAccountEmail,
				Delegates:           g.CallCredentials.Delegates,
				AccessToken:         accessTokenConfig,
			},
		}
	} else {
		filterConfig.AccessToken = &scpb.FilterConfig_ImdsToken{
			ImdsToken: accessTokenConfig.GetRemoteToken(),
		}

	}

	if g.GCPAttributes != nil {
		filterConfig.GcpAttributes = g.GCPAttributes
	}
	if g.ComputePlatformOverride != "" {
		if filterConfig.GcpAttributes == nil {
			filterConfig.GcpAttributes = &scpb.GcpAttributes{}
		}
		filterConfig.GcpAttributes.Platform = g.ComputePlatformOverride
	}

	depErrorBehaviorEnum, err := ParseDepErrorBehavior(g.DependencyErrorBehavior)
	if err != nil {
		return nil, err
	}

	filterConfig.DepErrorBehavior = depErrorBehaviorEnum

	return filterConfig, nil
}

func MakeSCCallingConfigFromOPConfig(opts options.ConfigGeneratorOptions) *scpb.ServiceControlCallingConfig {
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

func copyServiceConfigForReportMetrics(src *confpb.Service) *confpb.Service {
	// Logs and metrics fields are needed by the Envoy HTTP filter
	// to generate proper Metrics for Report calls.
	return &confpb.Service{
		Logs:               src.GetLogs(),
		Metrics:            src.GetMetrics(),
		MonitoredResources: src.GetMonitoredResources(),
		Monitoring:         src.GetMonitoring(),
		Logging:            src.GetLogging(),
	}
}

// ParseServiceControlURLFromOPConfig will get and parse the Service Control URL.
func ParseServiceControlURLFromOPConfig(serviceConfig *confpb.Service, opts options.ConfigGeneratorOptions) (url.URL, error) {
	uri := GetServiceControlURLFromOPConfig(serviceConfig, opts)
	if uri == "" {
		return url.URL{}, nil
	}

	// The assumption about control.environment field. Its format:
	//   [scheme://] +  host + [:port]
	// * It should not have any path part
	// * If scheme is missed, https is the default

	scURL, err := util.ParseURIIntoURL(uri)
	if err != nil {
		return url.URL{}, fmt.Errorf("failed to parse uri %q into url: %v", uri, err)
	}
	if scURL.Path != "" {
		return url.URL{}, fmt.Errorf("error parsing service control url %+v: should not have path part: %s", scURL, scURL.Path)
	}

	return scURL, nil
}

// GetServiceControlURLFromOPConfig chooses the right data source to read the Service
// Control URL from.
func GetServiceControlURLFromOPConfig(serviceConfig *confpb.Service, opts options.ConfigGeneratorOptions) string {
	// Ignore value from ServiceConfig if flag is set
	if uri := opts.ServiceControlURL; uri != "" {
		return uri
	}

	return serviceConfig.GetControl().GetEnvironment()
}

// MakeMethodRequirementsFromOPConfig creates the method requirements config.
func MakeMethodRequirementsFromOPConfig(serviceConfig *confpb.Service, opts options.ConfigGeneratorOptions) ([]*scpb.Requirement, error) {
	var requirements []*scpb.Requirement

	quotaAndUsageReqs, err := GetQuotaAndUsageRequirementsFromOPConfig(serviceConfig, opts)
	if err != nil {
		return nil, err
	}
	for _, quotaAndUsageReq := range quotaAndUsageReqs {
		requirements = append(requirements, quotaAndUsageReq)
	}

	corsRequirements := GetAutoGeneratedCORSRequirementsFromOPConfig(serviceConfig, opts)
	for _, corsRequirement := range corsRequirements {
		requirements = append(requirements, corsRequirement)
	}

	healthzRequirement := GetHealthzRequirementFromOPConfig(serviceConfig, opts)
	if healthzRequirement != nil {
		requirements = append(requirements, healthzRequirement)
	}

	return requirements, nil
}

func GetQuotaAndUsageRequirementsFromOPConfig(serviceConfig *confpb.Service, opts options.ConfigGeneratorOptions) ([]*scpb.Requirement, error) {
	var requirements []*scpb.Requirement

	metricCostsBySelector := GetQuotaMetricCostsFromOPConfig(serviceConfig, opts)
	usageRulesBySelector := GetUsageRulesBySelectorFromOPConfig(serviceConfig, opts)
	apiKeySystemParamsBySelector := GetAPIKeySystemParametersBySelectorFromOPConfig(serviceConfig, opts)

	for _, api := range serviceConfig.GetApis() {
		for _, method := range api.GetMethods() {
			selector := MethodToSelector(api, method)
			if util.ShouldSkipOPDiscoveryAPI(selector, opts.AllowDiscoveryAPIs) {
				glog.Warningf("Skip method %q because discovery API is not supported.", selector)
				continue
			}

			requirement := &scpb.Requirement{
				ServiceName:   serviceConfig.GetName(),
				OperationName: selector,
				ApiName:       api.GetName(),
				ApiVersion:    api.GetVersion(),
			}

			metricCosts, ok := metricCostsBySelector[selector]
			if ok {
				requirement.MetricCosts = metricCosts
			}

			if usageRule, ok := usageRulesBySelector[selector]; ok {
				requirement.SkipServiceControl = usageRule.GetSkipServiceControl()

				if usageRule.GetAllowUnregisteredCalls() {
					requirement.ApiKey = &scpb.ApiKeyRequirement{
						AllowWithoutApiKey: true,
					}
				}
			}

			if apiKeySystemParams, ok := apiKeySystemParamsBySelector[selector]; ok {
				if requirement.ApiKey == nil {
					requirement.ApiKey = &scpb.ApiKeyRequirement{}
				}
				requirement.ApiKey.Locations = ExtractAPIKeyLocations(apiKeySystemParams)
			}

			requirements = append(requirements, requirement)
		}
	}

	return requirements, nil
}

// GetQuotaMetricCostsFromOPConfig pre-processes all quota rules into a map of
// metric costs by selector.
func GetQuotaMetricCostsFromOPConfig(serviceConfig *confpb.Service, opts options.ConfigGeneratorOptions) map[string][]*scpb.MetricCost {
	metricCostsBySelector := make(map[string][]*scpb.MetricCost)

	for _, metricRule := range serviceConfig.GetQuota().GetMetricRules() {
		selector := metricRule.GetSelector()
		if util.ShouldSkipOPDiscoveryAPI(selector, opts.AllowDiscoveryAPIs) {
			glog.Warningf("Skip quota metric rule %q because discovery API is not supported.", selector)
			continue
		}

		var metricCosts []*scpb.MetricCost
		for name, cost := range metricRule.GetMetricCosts() {
			metricCosts = append(metricCosts, &scpb.MetricCost{
				Name: name,
				Cost: cost,
			})
		}

		// To keep tests from breaking due to map ordering.
		sort.Slice(metricCosts, func(i, j int) bool {
			return metricCosts[i].GetName() < metricCosts[j].GetName()
		})

		metricCostsBySelector[selector] = metricCosts
	}

	return metricCostsBySelector
}

// GetAutoGeneratedCORSRequirementsFromOPConfig returns the Service Control requirements
// for all auto-generated CORS methods (if enabled).
//
// Replaces ServiceInfo::processHttpRule.
func GetAutoGeneratedCORSRequirementsFromOPConfig(serviceConfig *confpb.Service, opts options.ConfigGeneratorOptions) []*scpb.Requirement {
	var corsMethods []*scpb.Requirement
	if !IsAutoGenCORSRequiredForOPConfig(serviceConfig, opts) {
		return corsMethods
	}

	corsOperationDelimiter := opts.CorsOperationDelimiter

	for _, api := range serviceConfig.GetApis() {
		for _, method := range api.GetMethods() {
			selector := MethodToSelector(api, method)
			if util.ShouldSkipOPDiscoveryAPI(selector, opts.AllowDiscoveryAPIs) {
				glog.Warningf("Skip api/method %q because discovery API is not supported.", selector)
				continue
			}
			genOperation := MethodToCORSSelector(api, method, corsOperationDelimiter)
			corsMethod := &scpb.Requirement{
				ServiceName:   serviceConfig.GetName(),
				OperationName: genOperation,
				ApiName:       api.GetName(),
				ApiVersion:    api.GetVersion(),
				ApiKey: &scpb.ApiKeyRequirement{
					AllowWithoutApiKey: true,
				},
			}

			corsMethods = append(corsMethods, corsMethod)
		}
	}

	return corsMethods
}

// GetHealthzRequirementFromOPConfig returns the Service Control requirement for
// autogenerated healthz method (if enabled).
//
// Replaces ServiceInfo::processHttpRule.
func GetHealthzRequirementFromOPConfig(serviceConfig *confpb.Service, opts options.ConfigGeneratorOptions) *scpb.Requirement {
	if opts.Healthz == "" {
		return nil
	}

	methodName := fmt.Sprintf("%s.%s_HealthCheck", opts.HealthCheckOperation, opts.HealthCheckAutogeneratedOperationPrefix)
	return &scpb.Requirement{
		ServiceName:        serviceConfig.GetName(),
		OperationName:      methodName,
		ApiName:            util.EspOperation,
		SkipServiceControl: true,
		ApiKey: &scpb.ApiKeyRequirement{
			AllowWithoutApiKey: true,
		},
	}
}

// ExtractAPIKeyLocations extracts the locations of API Keys from the system parameters
// into the corresponding SC filter config proto.
//
// System parameters passed in must only be ones for API Key, no other system
// parameters allowed.
//
// Replaces ServiceInfo::extractApiKeyLocations.
func ExtractAPIKeyLocations(parameters []*confpb.SystemParameter) []*scpb.ApiKeyLocation {
	var locations []*scpb.ApiKeyLocation

	for _, parameter := range parameters {
		if urlQueryName := parameter.GetUrlQueryParameter(); urlQueryName != "" {
			location := &scpb.ApiKeyLocation{
				Key: &scpb.ApiKeyLocation_Query{
					Query: urlQueryName,
				},
			}
			locations = append(locations, location)
		}
		if headerName := parameter.GetHttpHeader(); headerName != "" {
			location := &scpb.ApiKeyLocation{
				Key: &scpb.ApiKeyLocation_Header{
					Header: headerName,
				},
			}
			locations = append(locations, location)
		}
	}

	return locations
}
