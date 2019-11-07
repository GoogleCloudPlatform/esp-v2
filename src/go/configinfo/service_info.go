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

	"github.com/GoogleCloudPlatform/api-proxy/src/go/options"
	"github.com/GoogleCloudPlatform/api-proxy/src/go/util"
	"github.com/golang/glog"

	commonpb "github.com/GoogleCloudPlatform/api-proxy/src/go/proto/api/envoy/http/common"
	pmpb "github.com/GoogleCloudPlatform/api-proxy/src/go/proto/api/envoy/http/path_matcher"
	scpb "github.com/GoogleCloudPlatform/api-proxy/src/go/proto/api/envoy/http/service_control"
	durationpb "github.com/golang/protobuf/ptypes/duration"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// ServiceInfo contains service level information.
type ServiceInfo struct {
	Name     string
	ApiName  string
	ConfigID string

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
	// TODO(jilinxia): supports multi apis.
	if len(serviceConfig.GetApis()) > 1 {
		return nil, fmt.Errorf("not support multi apis yet")
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
		ApiName:         serviceConfig.GetApis()[0].GetName(),
		ConfigID:        id,
		serviceConfig:   serviceConfig,
		BackendProtocol: backendProtocol,
		Options:         opts,
	}

	// Order matters.
	serviceInfo.processEndpoints()
	serviceInfo.processApis()
	serviceInfo.processQuota()
	serviceInfo.processHttpRule()
	serviceInfo.processUsageRule()
	serviceInfo.processSystemParameters()
	serviceInfo.processAccessToken()
	if err := serviceInfo.processBackendRule(); err != nil {
		return nil, err
	}
	serviceInfo.processTypes()
	serviceInfo.processEmptyJwksUriByOpenID()

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

func (s *ServiceInfo) processEmptyJwksUriByOpenID() {
	authn := s.serviceConfig.GetAuthentication()
	for _, provider := range authn.GetProviders() {
		jwksUri := provider.GetJwksUri()

		// Note: When jwksUri is empty, proxy will try to find jwksUri by openID
		// discovery. If error happens during this process, a fake and unaccessible
		// jwksUri will be filled instead.
		if jwksUri == "" {
			jwksUriByOpenID, err := util.ResolveJwksUriUsingOpenID(provider.GetIssuer())
			if err != nil {
				glog.Warning(err.Error())
				jwksUri = util.FakeJwksUri
			} else {
				jwksUri = jwksUriByOpenID
			}
			provider.JwksUri = jwksUri
		}
	}
}

func (s *ServiceInfo) processApis() {
	s.Methods = make(map[string]*methodInfo)
	api := s.serviceConfig.GetApis()[0]
	for _, method := range api.GetMethods() {
		s.Methods[fmt.Sprintf("%s.%s", api.GetName(), method.GetName())] =
			&methodInfo{
				ShortName: method.GetName(),
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

func (s *ServiceInfo) processHttpRule() {
	// An temporary map to record generated OPTION methods, to avoid duplication.
	httpPathWithOptionsSet := make(map[string]bool)

	for _, r := range s.ServiceConfig().GetHttp().GetRules() {
		method := s.getOrCreateMethod(r.GetSelector())
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
			glog.Warning("unsupported http method")
		}
		method.HttpRule = append(method.HttpRule, httpRule)
	}

	// In order to support CORS. HTTP method OPTIONS needs to be added to all
	// urls except the ones already with options.
	if s.AllowCors {
		index := 0
		for _, r := range s.ServiceConfig().GetHttp().GetRules() {
			method := s.Methods[r.GetSelector()]
			for _, httpRule := range method.HttpRule {
				if httpRule.HttpMethod != "OPTIONS" {
					if _, exist := httpPathWithOptionsSet[httpRule.UriTemplate]; !exist {
						s.addOptionMethod(index, httpRule.UriTemplate)
						httpPathWithOptionsSet[httpRule.UriTemplate] = true
						index++
					}

				}
			}
		}
	}
}

func (s *ServiceInfo) addOptionMethod(index int, path string) {
	// All options have their operation as the following format: CORS_suffix.
	// Appends suffix to make sure it is not used by any http rules.
	corsOperationBase := "CORS"
	corsOperation := fmt.Sprintf("%s_%d", corsOperationBase, index)
	s.Methods[fmt.Sprintf("%s.%s", s.ApiName, corsOperation)] = &methodInfo{
		ShortName: corsOperation,
		HttpRule: []*commonpb.Pattern{
			{
				UriTemplate: path,
				HttpMethod:  util.OPTIONS,
			},
		},
		IsGeneratedOption: true,
	}
}

func (s *ServiceInfo) processBackendRule() error {
	if !s.Options.EnableBackendRouting {
		return nil
	}
	backendRoutingClustersMap := make(map[string]string)

	for _, r := range s.ServiceConfig().Backend.GetRules() {
		if r.PathTranslation != confpb.BackendRule_PATH_TRANSLATION_UNSPECIFIED {
			scheme, hostname, port, uri, err := util.ParseURI(r.Address)
			if err != nil {
				return err
			}
			if scheme != "https" {
				return fmt.Errorf("dynamic routing only supports HTTPS")
			}
			if net.ParseIP(hostname) != nil {
				return fmt.Errorf("dynamic routing only supports domain name, got IP address: %v", hostname)
			}
			address := fmt.Sprintf("%v:%v", hostname, port)
			if _, exist := backendRoutingClustersMap[address]; !exist {
				backendSelector := fmt.Sprintf("DynamicRouting_%v", len(s.BackendRoutingClusters))
				s.BackendRoutingClusters = append(s.BackendRoutingClusters,
					&backendRoutingCluster{
						ClusterName: backendSelector,
						Hostname:    hostname,
						Port:        port,
					})
				backendRoutingClustersMap[address] = backendSelector
			}

			clusterName := backendRoutingClustersMap[address]

			method := s.getOrCreateMethod(r.GetSelector())

			// For CONSTANT_ADDRESS, an empty uri will generate an empty path header.
			// It is an invalid Http header if path is empty.
			if uri == "" && r.PathTranslation == confpb.BackendRule_CONSTANT_ADDRESS {
				uri = "/"
			}

			method.BackendRule = backendInfo{
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

func (s *ServiceInfo) processUsageRule() {
	for _, r := range s.ServiceConfig().GetUsage().GetRules() {
		method := s.getOrCreateMethod(r.GetSelector())
		method.AllowUnregisteredCalls = r.GetAllowUnregisteredCalls()
		method.SkipServiceControl = r.GetSkipServiceControl()
	}
}

func (s *ServiceInfo) processSystemParameters() {
	for _, rule := range s.ServiceConfig().GetSystemParameters().GetRules() {
		apiKeyLocationParameters := []*confpb.SystemParameter{}
		for _, parameter := range rule.GetParameters() {
			if parameter.GetName() == util.APIKeyParameterName {
				apiKeyLocationParameters = append(apiKeyLocationParameters, parameter)
			}
		}
		extractAPIKeyLocations(s.getOrCreateMethod(rule.GetSelector()), apiKeyLocationParameters)
	}
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
func (s *ServiceInfo) getOrCreateMethod(name string) *methodInfo {
	if s.Methods[name] == nil {
		names := strings.Split(name, ".")
		s.Methods[name] = &methodInfo{
			ShortName: names[len(names)-1],
		}
	}
	return s.Methods[name]
}
