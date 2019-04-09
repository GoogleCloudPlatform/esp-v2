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

package configinfo

import (
	"fmt"
	"sort"
	"strings"

	"cloudesf.googlesource.com/gcpproxy/src/go/flags"
	"github.com/golang/glog"
	"google.golang.org/genproto/googleapis/api/annotations"

	commonpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/common"
	pmpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/path_matcher"
	scpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/service_control"
	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
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
	BackendProtocol   ut.BackendProtocol
	GcpAttributes     *scpb.GcpAttributes
	// Keep a pointer to original service config. Should always process rules
	// inside ServiceInfo.
	serviceConfig *conf.Service
}

type backendRoutingCluster struct {
	ClusterName string
	Hostname    string
	Port        uint32
}

// NewServiceInfoFromServiceConfig returns an instance of ServiceInfo.
func NewServiceInfoFromServiceConfig(serviceConfig *conf.Service, id string) (*ServiceInfo, error) {
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

	var backendProtocol ut.BackendProtocol
	switch strings.ToLower(*flags.BackendProtocol) {
	case "http1":
		backendProtocol = ut.HTTP1
	case "http2":
		backendProtocol = ut.HTTP2
	case "grpc":
		backendProtocol = ut.GRPC
	default:
		return nil, fmt.Errorf(`unknown backend protocol, should be one of "grpc", "http1" or "http2"`)
	}

	serviceInfo := &ServiceInfo{
		Name:            serviceConfig.GetName(),
		ApiName:         serviceConfig.GetApis()[0].GetName(),
		ConfigID:        id,
		serviceConfig:   serviceConfig,
		BackendProtocol: backendProtocol,
	}

	// Order matters.
	serviceInfo.processEndpoints()
	serviceInfo.processApis()
	serviceInfo.processHttpRule()
	serviceInfo.processUsageRule()
	if err := serviceInfo.processBackendRule(); err != nil {
		return nil, err
	}
	serviceInfo.processTypes()

	// Sort Methods according to name.
	for operation := range serviceInfo.Methods {
		serviceInfo.Operations = append(serviceInfo.Operations, operation)
	}
	sort.Strings(serviceInfo.Operations)

	return serviceInfo, nil
}

// Returns the pointer of the ServiceConfig that this API belongs to.
func (s *ServiceInfo) ServiceConfig() *conf.Service {
	return s.serviceConfig
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
		switch r.GetPattern().(type) {
		case *annotations.HttpRule_Get:
			method.HttpRule = commonpb.Pattern{
				UriTemplate: r.GetGet(),
				HttpMethod:  ut.GET,
			}
		case *annotations.HttpRule_Put:
			method.HttpRule = commonpb.Pattern{
				UriTemplate: r.GetPut(),
				HttpMethod:  ut.PUT,
			}
		case *annotations.HttpRule_Post:
			method.HttpRule = commonpb.Pattern{
				UriTemplate: r.GetPost(),
				HttpMethod:  ut.POST,
			}
		case *annotations.HttpRule_Delete:
			method.HttpRule = commonpb.Pattern{
				UriTemplate: r.GetDelete(),
				HttpMethod:  ut.DELETE,
			}
		case *annotations.HttpRule_Patch:
			method.HttpRule = commonpb.Pattern{
				UriTemplate: r.GetPatch(),
				HttpMethod:  ut.PATCH,
			}
		case *annotations.HttpRule_Custom:
			method.HttpRule = commonpb.Pattern{
				UriTemplate: r.GetCustom().GetPath(),
				HttpMethod:  r.GetCustom().GetKind(),
			}
			httpPathWithOptionsSet[r.GetCustom().GetPath()] = true
		default:
			glog.Warning("unsupported http method")
		}
	}

	// In order to support CORS. HTTP method OPTIONS needs to be added to all
	// urls except the ones already with options.
	if s.AllowCors {
		index := 0
		for _, r := range s.ServiceConfig().GetHttp().GetRules() {
			method := s.Methods[r.GetSelector()]
			if method.HttpRule.HttpMethod != "OPTIONS" {
				if _, exist := httpPathWithOptionsSet[method.HttpRule.UriTemplate]; !exist {
					s.addOptionMethod(index, method.HttpRule.UriTemplate)
					httpPathWithOptionsSet[method.HttpRule.UriTemplate] = true
					index++
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
		HttpRule: commonpb.Pattern{
			UriTemplate: path,
			HttpMethod:  ut.OPTIONS,
		},
		IsGeneratedOption: true,
	}
}

func (s *ServiceInfo) processBackendRule() error {
	if !*flags.EnableBackendRouting {
		return nil
	}
	backendRoutingClustersMap := make(map[string]string)

	for _, r := range s.ServiceConfig().Backend.GetRules() {
		if r.PathTranslation != conf.BackendRule_PATH_TRANSLATION_UNSPECIFIED {
			hostname, port, uri, err := ut.ParseURL(r.Address)
			if err != nil {
				return err
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
	}
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
