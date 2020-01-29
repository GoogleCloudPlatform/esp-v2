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

package configinfo

import (
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/glog"

	commonpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/common"
	pmpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/path_matcher"
	scpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/service_control"
	durationpb "github.com/golang/protobuf/ptypes/duration"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// ServiceInfo contains service level information.
type ServiceInfo struct {
	Name     string
	ConfigID string

	// An array to store all the api names
	ApiNames []string
	// A sorted array to store all the method name for this service.
	// Should always iterate this array to avoid test fail due to order issue.
	Operations []string
	// Stores all methods info for this service, using selector as key.
	Methods map[string]*methodInfo
	// Stores information about backend clusters for re-routing.
	BackendRoutingClusters []*BackendRoutingCluster
	// Stores url segment names, mapping snake name to Json name.
	SegmentNames []*pmpb.SegmentName

	AllowCors         bool
	BackendIsGrpc     bool
	ServiceControlURI string
	CatchAllBackend   *BackendRoutingCluster
	GcpAttributes     *scpb.GcpAttributes
	// Keep a pointer to original service config. Should always process rules
	// inside ServiceInfo.
	serviceConfig *confpb.Service
	AccessToken   *commonpb.AccessToken
	Options       options.ConfigGeneratorOptions
}

type BackendRoutingCluster struct {
	ClusterName string
	Hostname    string
	Port        uint32
	UseTLS      bool
	Protocol    util.BackendProtocol
}

// NewServiceInfoFromServiceConfig returns an instance of ServiceInfo.
func NewServiceInfoFromServiceConfig(serviceConfig *confpb.Service, id string, opts options.ConfigGeneratorOptions) (*ServiceInfo, error) {
	if serviceConfig == nil {
		return nil, fmt.Errorf("unexpected empty service config")
	}
	if len(serviceConfig.GetApis()) == 0 {
		return nil, fmt.Errorf("service config must have one api at least")
	}

	serviceInfo := &ServiceInfo{
		Name:          serviceConfig.GetName(),
		ConfigID:      id,
		serviceConfig: serviceConfig,
		Options:       opts,
		Methods:       make(map[string]*methodInfo),
	}

	// Calling order is required due to following variable usage
	// * AllowCors:
	//    set by: processEndpoints
	//    used by: processHttpRule
	// * BackendInfo map to MethodInfo
	//    set by processApi
	//    used by processBackendRule
	// * BackendIsGrpc:
	//     set by processBackendRule, buildCatchAllBackend
	//     used by addGrpcHttpRules
	if err := serviceInfo.buildCatchAllBackend(); err != nil {
		return nil, err
	}
	serviceInfo.processEndpoints()
	serviceInfo.processApis()
	serviceInfo.processQuota()
	if err := serviceInfo.processBackendRule(); err != nil {
		return nil, err
	}
	if err := serviceInfo.processHttpRule(); err != nil {
		return nil, err
	}
	if err := serviceInfo.processUsageRule(); err != nil {
		return nil, err
	}
	if err := serviceInfo.processSystemParameters(); err != nil {
		return nil, err
	}
	serviceInfo.processAccessToken()
	serviceInfo.processTypes()
	serviceInfo.addGrpcHttpRules()

	if err := serviceInfo.processEmptyJwksUriByOpenID(); err != nil {
		return nil, err
	}

	// Sort Methods according to name.
	for operation := range serviceInfo.Methods {
		serviceInfo.Operations = append(serviceInfo.Operations, operation)
	}
	sort.Strings(serviceInfo.Operations)

	return serviceInfo, nil
}

func (s *ServiceInfo) buildCatchAllBackend() error {
	protocol, tls, err := util.ParseBackendProtocol(s.Options.BackendProtocol)
	if err != nil {
		return err
	}
	if protocol == util.GRPC {
		s.BackendIsGrpc = true
	}

	s.CatchAllBackend = &BackendRoutingCluster{
		UseTLS:      tls,
		Protocol:    protocol,
		ClusterName: s.BackendClusterName(),
		Hostname:    s.Options.ClusterAddress,
		Port:        uint32(s.Options.ClusterPort),
	}
	return nil
}

// Returns the pointer of the ServiceConfig that this API belongs to.
func (s *ServiceInfo) ServiceConfig() *confpb.Service {
	return s.serviceConfig
}

func (s *ServiceInfo) processEmptyJwksUriByOpenID() error {
	authn := s.serviceConfig.GetAuthentication()
	for _, provider := range authn.GetProviders() {
		jwksUri := provider.GetJwksUri()

		// Note: When jwksUri is empty, proxy will try to find jwksUri using the
		// OpenID Connect Discovery protocol.
		if jwksUri == "" {
			glog.Infof("jwks_uri is empty, using OpenID Connect Discovery protocol")
			jwksUriByOpenID, err := util.ResolveJwksUriUsingOpenID(provider.GetIssuer())
			if err != nil {
				return fmt.Errorf("failed OpenID Connect Discovery protocol: %v", err)
			} else {
				jwksUri = jwksUriByOpenID
			}
			provider.JwksUri = jwksUri
		}
	}

	return nil
}

func (s *ServiceInfo) processApis() {
	for _, api := range s.serviceConfig.GetApis() {
		s.ApiNames = append(s.ApiNames, api.Name)

		for _, method := range api.GetMethods() {
			selector := fmt.Sprintf("%s.%s", api.GetName(), method.GetName())
			mi, _ := s.getOrCreateMethod(selector)
			// Keep track of non-unary gRPC methods.
			if method.RequestStreaming || method.ResponseStreaming {
				mi.IsStreaming = true
			}
		}
	}
}

func (s *ServiceInfo) addGrpcHttpRules() {
	// If there is not grpc backend, not to add grpc HttpRules
	if !s.BackendIsGrpc {
		return
	}

	for _, api := range s.serviceConfig.GetApis() {
		for _, method := range api.GetMethods() {
			selector := fmt.Sprintf("%s.%s", api.GetName(), method.GetName())
			mi, _ := s.getOrCreateMethod(selector)
			mi.HttpRule = append(mi.HttpRule, &commonpb.Pattern{
				UriTemplate: fmt.Sprintf("/%s/%s", api.GetName(), method.GetName()),
				HttpMethod:  util.POST,
			})
		}
	}
}

func (s *ServiceInfo) processAccessToken() {
	if s.Options.ServiceAccountKey != "" {
		data, _ := ioutil.ReadFile(s.Options.ServiceAccountKey)
		s.AccessToken = &commonpb.AccessToken{
			TokenType: &commonpb.AccessToken_ServiceAccountSecret{
				ServiceAccountSecret: &commonpb.DataSource{
					Specifier: &commonpb.DataSource_InlineString{
						InlineString: string(data),
					},
				},
			},
		}
		return
	}
	s.AccessToken = &commonpb.AccessToken{
		TokenType: &commonpb.AccessToken_RemoteToken{
			RemoteToken: &commonpb.HttpUri{
				Uri:     fmt.Sprintf("%s%s", s.Options.MetadataURL, util.AccessTokenSuffix),
				Cluster: util.MetadataServerClusterName,
				// TODO(taoxuy): make token_subscriber use this timeout
				Timeout: &durationpb.Duration{Seconds: 5},
			},
		},
	}
}

func (s *ServiceInfo) processQuota() {
	for _, metricRule := range s.ServiceConfig().GetQuota().GetMetricRules() {
		var metricCosts []*scpb.MetricCost
		for name, cost := range metricRule.GetMetricCosts() {
			metricCosts = append(metricCosts, &scpb.MetricCost{
				Name: name,
				Cost: cost,
			})
		}
		s.Methods[metricRule.GetSelector()].MetricCosts = metricCosts
	}
}

func (s *ServiceInfo) processEndpoints() {
	for _, endpoint := range s.ServiceConfig().GetEndpoints() {
		if endpoint.GetName() == s.ServiceConfig().GetName() && endpoint.GetAllowCors() {
			s.AllowCors = true
		}
	}
}

func addHttpRule(method *methodInfo, r *annotationspb.HttpRule, httpPathWithOptionsSet map[string]bool) error {

	var httpRule *commonpb.Pattern
	switch r.GetPattern().(type) {
	case *annotationspb.HttpRule_Get:
		httpRule = &commonpb.Pattern{
			UriTemplate: r.GetGet(),
			HttpMethod:  util.GET,
		}
	case *annotationspb.HttpRule_Put:
		httpRule = &commonpb.Pattern{
			UriTemplate: r.GetPut(),
			HttpMethod:  util.PUT,
		}
	case *annotationspb.HttpRule_Post:
		httpRule = &commonpb.Pattern{
			UriTemplate: r.GetPost(),
			HttpMethod:  util.POST,
		}
	case *annotationspb.HttpRule_Delete:
		httpRule = &commonpb.Pattern{
			UriTemplate: r.GetDelete(),
			HttpMethod:  util.DELETE,
		}
	case *annotationspb.HttpRule_Patch:
		httpRule = &commonpb.Pattern{
			UriTemplate: r.GetPatch(),
			HttpMethod:  util.PATCH,
		}
	case *annotationspb.HttpRule_Custom:
		httpRule = &commonpb.Pattern{
			UriTemplate: r.GetCustom().GetPath(),
			HttpMethod:  r.GetCustom().GetKind(),
		}
		httpPathWithOptionsSet[r.GetCustom().GetPath()] = true
	default:
		return fmt.Errorf("unsupported http method %T", r.GetPattern())
	}
	method.HttpRule = append(method.HttpRule, httpRule)

	return nil
}

func (s *ServiceInfo) processHttpRule() error {
	// An temporary map to record generated OPTION methods, to avoid duplication.
	httpPathWithOptionsSet := make(map[string]bool)

	for _, rule := range s.ServiceConfig().GetHttp().GetRules() {
		method, err := s.getOrCreateMethod(rule.GetSelector())
		if err != nil {
			return err
		}
		if err := addHttpRule(method, rule, httpPathWithOptionsSet); err != nil {
			return err
		}

		// additional_bindings cannot be nested inside themselves according to
		// https://aip.dev/127. Service Management will enforce this restriction
		// when interpret the httprules from the descriptor. Therefore, no need to
		// check for nested additional_bindings.
		for _, additionalRule := range rule.AdditionalBindings {
			if err := addHttpRule(method, additionalRule, httpPathWithOptionsSet); err != nil {
				return err
			}
		}
	}

	// In order to support CORS. HTTP method OPTIONS needs to be added to all
	// urls except the ones already with options.
	if s.AllowCors {
		for _, r := range s.ServiceConfig().GetHttp().GetRules() {
			method := s.Methods[r.GetSelector()]
			for _, httpRule := range method.HttpRule {
				if httpRule.HttpMethod != "OPTIONS" {
					if _, exist := httpPathWithOptionsSet[httpRule.UriTemplate]; !exist {
						s.addOptionMethod(method.ApiName, httpRule.UriTemplate, method.BackendInfo)
						httpPathWithOptionsSet[httpRule.UriTemplate] = true
					}

				}
			}
		}
	}

	// Add HttpRule for HealthCheck method
	if s.Options.Healthz != "" {
		hcMethod, err := s.getOrCreateMethod("ESPv2.HealthCheck")
		if err != nil {
			return err
		}
		if !strings.HasPrefix(s.Options.Healthz, "/") {
			s.Options.Healthz = fmt.Sprintf("/%s", s.Options.Healthz)
		}

		hcMethod.HttpRule = append(hcMethod.HttpRule, &commonpb.Pattern{
			UriTemplate: s.Options.Healthz,
			HttpMethod:  util.GET,
		})
		hcMethod.SkipServiceControl = true
		hcMethod.IsGenerated = true
	}

	return nil
}

func (s *ServiceInfo) addOptionMethod(apiName string, path string, backendInfo *backendInfo) {
	// All options have their operation as the following format: CORS_${suffix}.
	// Appends ${suffix} to make sure it is not used by any http rules.
	//
	// b/145622434 changes ${suffix} to contain a url-safe path by replacing
	// OpenAPI-specific characters from the UriTemplate. Spec here:
	// https://swagger.io/docs/specification/paths-and-operations/
	// This will ensure other services can correctly parse/display the operation name.
	corsOperationBase := "CORS"
	formattedPath := strings.TrimPrefix(path, "/")
	formattedPath = strings.ReplaceAll(formattedPath, "/", "_")
	formattedPath = strings.ReplaceAll(formattedPath, "{", "")
	formattedPath = strings.ReplaceAll(formattedPath, "}", "")
	corsOperation := fmt.Sprintf("%s_%s", corsOperationBase, formattedPath)
	genOperation := fmt.Sprintf("%s.%s", apiName, corsOperation)

	s.Methods[genOperation] = &methodInfo{
		ShortName: corsOperation,
		ApiName:   apiName,
		HttpRule: []*commonpb.Pattern{
			{
				UriTemplate: path,
				HttpMethod:  util.OPTIONS,
			},
		},
		IsGenerated: true,
		BackendInfo: backendInfo,
	}
}

func (s *ServiceInfo) processBackendRule() error {
	backendRoutingClustersMap := make(map[string]string)

	for _, r := range s.ServiceConfig().Backend.GetRules() {
		if r.Address != "" {
			scheme, hostname, port, uri, err := util.ParseURI(r.Address)
			if err != nil {
				return err
			}
			if net.ParseIP(hostname) != nil {
				return fmt.Errorf("dynamic routing only supports domain name, got IP address: %v", hostname)
			}
			address := fmt.Sprintf("%v:%v", hostname, port)

			if _, exist := backendRoutingClustersMap[address]; !exist {
				protocol, tls, err := util.ParseBackendProtocol(scheme)
				if err != nil {
					return err
				}
				if protocol == util.GRPC {
					s.BackendIsGrpc = true
				}

				backendSelector := address
				s.BackendRoutingClusters = append(s.BackendRoutingClusters,
					&BackendRoutingCluster{
						ClusterName: backendSelector,
						UseTLS:      tls,
						Protocol:    protocol,
						Hostname:    hostname,
						Port:        port,
					})
				backendRoutingClustersMap[address] = backendSelector
			}

			clusterName := backendRoutingClustersMap[address]

			method, err := s.getOrCreateMethod(r.GetSelector())
			if err != nil {
				return err
			}

			// For CONSTANT_ADDRESS, an empty uri will generate an empty path header.
			// It is an invalid Http header if path is empty.
			if uri == "" && r.PathTranslation == confpb.BackendRule_CONSTANT_ADDRESS {
				uri = "/"
			}

			var deadline time.Duration
			if r.Deadline == 0 {
				// If no deadline specified by the user, explicitly use default.
				deadline = util.DefaultResponseDeadline
			} else if r.Deadline < 0 {
				glog.Warningf("Negative deadline of %v specified for method %v. "+
					"Using default deadline %v instead.", r.Deadline, address, util.DefaultResponseDeadline)
				deadline = util.DefaultResponseDeadline
			} else {
				// The backend deadline from the BackendRule is a float64 that represents seconds.
				// But float64 has a large precision, so we must explicitly lower the precision.
				// For the purposes of a network proxy, round the deadline to the nearest millisecond.
				deadlineMs := int64(math.Round(r.Deadline * 1000))
				deadline = time.Duration(deadlineMs) * time.Millisecond
			}

			method.BackendInfo = &backendInfo{
				ClusterName:     clusterName,
				Uri:             uri,
				Hostname:        hostname,
				TranslationType: r.PathTranslation,
				JwtAudience:     r.GetJwtAudience(),
				Deadline:        deadline,
			}
		}
	}
	return nil
}

func (s *ServiceInfo) processUsageRule() error {
	for _, r := range s.ServiceConfig().GetUsage().GetRules() {
		method, err := s.getOrCreateMethod(r.GetSelector())
		if err != nil {
			return err
		}
		method.AllowUnregisteredCalls = r.GetAllowUnregisteredCalls()
		method.SkipServiceControl = r.GetSkipServiceControl()
	}
	return nil
}

func (s *ServiceInfo) processSystemParameters() error {
	for _, rule := range s.ServiceConfig().GetSystemParameters().GetRules() {
		apiKeyLocationParameters := []*confpb.SystemParameter{}
		for _, parameter := range rule.GetParameters() {
			if parameter.GetName() == util.APIKeyParameterName {
				apiKeyLocationParameters = append(apiKeyLocationParameters, parameter)
			}
		}
		method, err := s.getOrCreateMethod(rule.GetSelector())
		if err != nil {
			return err
		}
		extractAPIKeyLocations(method, apiKeyLocationParameters)
	}
	return nil
}

func extractAPIKeyLocations(method *methodInfo, parameters []*confpb.SystemParameter) {
	var urlQueryNames, headerNames []*scpb.APIKeyLocation
	for _, parameter := range parameters {
		if urlQueryName := parameter.GetUrlQueryParameter(); urlQueryName != "" {
			urlQueryNames = append(urlQueryNames, &scpb.APIKeyLocation{
				Key: &scpb.APIKeyLocation_Query{
					Query: urlQueryName,
				},
			})
		}
		if headerName := parameter.GetHttpHeader(); headerName != "" {
			headerNames = append(headerNames, &scpb.APIKeyLocation{
				Key: &scpb.APIKeyLocation_Header{
					Header: headerName,
				},
			})
		}
	}
	method.APIKeyLocations = append(method.APIKeyLocations, urlQueryNames...)
	method.APIKeyLocations = append(method.APIKeyLocations, headerNames...)
}

func (s *ServiceInfo) processTypes() {
	// Create snake name to JSON name mapping.
	for _, t := range s.ServiceConfig().GetTypes() {
		for _, f := range t.GetFields() {
			if strings.ContainsRune(f.GetName(), '_') {
				s.SegmentNames = append(s.SegmentNames, &pmpb.SegmentName{
					SnakeName: f.GetName(),
					JsonName:  f.GetJsonName(),
				})
			}
		}
	}
}

// get the methodInfo by full name, and create a new one if not exists.
// Ideally, all selector name in service config rules should exist in the api
// methods.
func (s *ServiceInfo) getOrCreateMethod(name string) (*methodInfo, error) {
	if s.Methods[name] == nil {
		names := strings.Split(name, ".")
		if len(names) <= 1 {
			return nil, fmt.Errorf("method %s should be in the format of apiName.methodShortName", name)
		}
		shortName := names[len(names)-1]
		s.Methods[name] = &methodInfo{
			ShortName: shortName,
			ApiName:   name[:len(name)-len(shortName)-1],
		}
	}
	return s.Methods[name], nil
}

func (s *ServiceInfo) BackendClusterName() string {
	return fmt.Sprintf("%s_local", s.Name)
}
