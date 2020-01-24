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
	"net"
	"sort"
	"strings"

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
	BackendRoutingClusters []*backendRoutingCluster
	// Stores url segment names, mapping snake name to Json name.
	SegmentNames []*pmpb.SegmentName

	AllowCors         bool
	ServiceControlURI string
	BackendProtocol   util.BackendProtocol
	GcpAttributes     *scpb.GcpAttributes
	// Keep a pointer to original service config. Should always process rules
	// inside ServiceInfo.
	serviceConfig *confpb.Service
	AccessToken   *commonpb.AccessToken
	Options       options.ConfigGeneratorOptions
}

type backendRoutingCluster struct {
	ClusterName string
	Hostname    string
	Port        uint32
}

// NewServiceInfoFromServiceConfig returns an instance of ServiceInfo.
func NewServiceInfoFromServiceConfig(serviceConfig *confpb.Service, id string, opts options.ConfigGeneratorOptions) (*ServiceInfo, error) {
	if serviceConfig == nil {
		return nil, fmt.Errorf("unexpected empty service config")
	}
	if len(serviceConfig.GetApis()) == 0 {
		return nil, fmt.Errorf("service config must have one api at least")
	}

	var backendProtocol util.BackendProtocol
	switch strings.ToLower(opts.BackendProtocol) {
	case "http1":
		backendProtocol = util.HTTP1
	case "http2":
		backendProtocol = util.HTTP2
	case "grpc":
		backendProtocol = util.GRPC
	default:
		return nil, fmt.Errorf(`unknown backend protocol, should be one of "grpc", "http1" or "http2"`)
	}

	serviceInfo := &ServiceInfo{
		Name:            serviceConfig.GetName(),
		ConfigID:        id,
		serviceConfig:   serviceConfig,
		BackendProtocol: backendProtocol,
		Options:         opts,
	}

	// check BackendRule to decide BackendProtocol
	if err := serviceInfo.checkBackendRuleForProtocol(); err != nil {
		return nil, err
	}

	// Order matters.
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
	s.Methods = make(map[string]*methodInfo)
	for _, api := range s.serviceConfig.GetApis() {
		s.ApiNames = append(s.ApiNames, api.Name)

		for _, method := range api.GetMethods() {
			mi := &methodInfo{
				ShortName: method.GetName(),
				ApiName:   api.GetName(),
			}
			if s.BackendProtocol == util.GRPC {
				mi.HttpRule = append(mi.HttpRule, &commonpb.Pattern{
					UriTemplate: fmt.Sprintf("/%s/%s", api.GetName(), method.GetName()),
					HttpMethod:  util.POST,
				})
			}
			s.Methods[fmt.Sprintf("%s.%s", api.GetName(), method.GetName())] = mi
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
	if s.BackendProtocol != util.GRPC && len(s.ServiceConfig().GetHttp().GetRules()) == 0 {
		return fmt.Errorf("no HttpRules specified for the Http service %v", s.Name)
	}

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

// check BackendRule to decide BackendProtocol
func (s *ServiceInfo) checkBackendRuleForProtocol() error {
	var firstScheme string
	for _, r := range s.ServiceConfig().Backend.GetRules() {
		if r.Address != "" {
			scheme, hostname, _, _, err := util.ParseURI(r.Address)
			if err != nil {
				return err
			}
			if net.ParseIP(hostname) != nil {
				return fmt.Errorf("dynamic routing only supports domain name, got IP address: %v", hostname)
			}
			if scheme != "https" && scheme != "grpc" && scheme != "http" {
				return fmt.Errorf("dynamic routing only supports https/http/grpc scheme, found: %v", scheme)
			}
			// Make sure all backends not to mix grpc with http/https scheme
			// https and http can use the same filter chain so they can be mixed
			if firstScheme == "" {
				firstScheme = scheme
			} else if (firstScheme == "grpc") != (scheme == "grpc") {
				return fmt.Errorf("dynamic routing could not mix grpc with http/https scheme, found: %v, %v", firstScheme, scheme)
			}
		}
	}
	if firstScheme == "grpc" {
		s.BackendProtocol = util.GRPC
	}
	return nil
}

func (s *ServiceInfo) processBackendRule() error {
	backendRoutingClustersMap := make(map[string]string)

	for _, r := range s.ServiceConfig().Backend.GetRules() {
		if r.Address != "" {
			_, hostname, port, uri, err := util.ParseURI(r.Address)
			address := fmt.Sprintf("%v:%v", hostname, port)

			// TODO(taoxuy): In order to support mixing http and https,
			// needs to pass scheme to backendRouteCluster so that
			// makeBackendRoutingClusters knows if TLS is needed.
			// Now TLS is always generated.

			if _, exist := backendRoutingClustersMap[address]; !exist {
				backendSelector := address
				s.BackendRoutingClusters = append(s.BackendRoutingClusters,
					&backendRoutingCluster{
						ClusterName: backendSelector,
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

			method.BackendInfo = &backendInfo{
				ClusterName:     clusterName,
				Uri:             uri,
				Hostname:        hostname,
				TranslationType: r.PathTranslation,
				JwtAudience:     r.GetJwtAudience(),
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
