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

	"cloudesf.googlesource.com/gcpproxy/src/go/flags"
	"github.com/golang/glog"
	"google.golang.org/genproto/googleapis/api/annotations"

	scpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/service_control"
	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// ServiceInfo contains service level information.
type ServiceInfo struct {
	Name string
	// TODO(jilinxia): update this when supporting multi apis.
	ApiName  string
	ConfigID string

	serviceConfig *conf.Service
	GcpAttributes *scpb.GcpAttributes

	ServiceControlURI string
	// TODO(jilinxia): move all the following maps into MethodInfo.
	DynamicRoutingBackendMap map[string]backendInfo
	// All non-generated operations
	OperationSet map[string]bool
	// HttpPathMap stores all operations to http path pairs.
	HttpPathMap            map[string]*HttpRule
	HttpPathWithOptionsSet map[string]bool
	// Generated OPTIONS operation for CORS.
	GeneratedOptionsOperations []string
	BackendRoutingInfos        []backendRoutingInfo
}

// HttpRule includes information for HTTP rules.
type HttpRule struct {
	Path   string
	Method string
}

type backendInfo struct {
	Name     string
	Hostname string
	Port     uint32
}

type backendRoutingInfo struct {
	Selector        string
	TranslationType conf.BackendRule_PathTranslation
	Backend         backendInfo
	Uri             string
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

	serviceInfo := &ServiceInfo{
		Name:          serviceConfig.GetName(),
		ConfigID:      id,
		serviceConfig: serviceConfig,
	}

	serviceInfo.processHttpRule()

	if *flags.EnableBackendRouting {
		if err := serviceInfo.processBackendRule(); err != nil {
			return nil, err
		}
	}

	serviceInfo.ApiName = serviceConfig.GetApis()[0].GetName()
	return serviceInfo, nil
}

// Returns the pointer of the ServiceConfig that this API belongs to.
func (s *ServiceInfo) ServiceConfig() *conf.Service {
	return s.serviceConfig
}

func (s *ServiceInfo) processHttpRule() {
	s.HttpPathMap = make(map[string]*HttpRule)
	s.HttpPathWithOptionsSet = make(map[string]bool)
	for _, r := range s.ServiceConfig().GetHttp().GetRules() {
		var rule *HttpRule
		switch r.GetPattern().(type) {
		case *annotations.HttpRule_Get:
			rule = &HttpRule{
				Path:   r.GetGet(),
				Method: ut.GET,
			}
		case *annotations.HttpRule_Put:
			rule = &HttpRule{
				Path:   r.GetPut(),
				Method: ut.PUT,
			}
		case *annotations.HttpRule_Post:
			rule = &HttpRule{
				Path:   r.GetPost(),
				Method: ut.POST,
			}
		case *annotations.HttpRule_Delete:
			rule = &HttpRule{
				Path:   r.GetDelete(),
				Method: ut.DELETE,
			}
		case *annotations.HttpRule_Patch:
			rule = &HttpRule{
				Path:   r.GetPatch(),
				Method: ut.PATCH,
			}
		case *annotations.HttpRule_Custom:
			rule = &HttpRule{
				Path:   r.GetCustom().GetPath(),
				Method: r.GetCustom().GetKind(),
			}
		default:
			glog.Warning("unsupported http method")
		}

		if rule.Method == ut.OPTIONS {
			s.HttpPathWithOptionsSet[rule.Path] = true
		}
		s.HttpPathMap[r.GetSelector()] = rule
	}
}

func (s *ServiceInfo) processBackendRule() error {
	s.DynamicRoutingBackendMap = make(map[string]backendInfo)
	for _, r := range s.ServiceConfig().Backend.GetRules() {
		// for CONSTANT_ADDRESS and APPEND_PATH_TO_ADDRESS
		if r.PathTranslation != conf.BackendRule_PATH_TRANSLATION_UNSPECIFIED {
			hostname, port, uri, err := ut.ParseURL(r.Address)
			if err != nil {
				return err
			}
			address := fmt.Sprintf("%v:%v", hostname, port)
			if _, exist := s.DynamicRoutingBackendMap[address]; !exist {
				backendSelector := fmt.Sprintf("DynamicRouting.%v", len(s.DynamicRoutingBackendMap))
				s.DynamicRoutingBackendMap[address] = backendInfo{
					Name:     backendSelector,
					Hostname: hostname,
					Port:     port,
				}
			}
			s.BackendRoutingInfos = append(s.BackendRoutingInfos, backendRoutingInfo{
				Selector:        r.Selector,
				TranslationType: r.PathTranslation,
				Backend:         s.DynamicRoutingBackendMap[address],
				Uri:             uri,
			})
		}
	}
	return nil
}

// TODO(jilinxia): this should be stored a bit.
func (s *ServiceInfo) GetEndpointAllowCorsFlag() bool {
	for _, endpoint := range s.ServiceConfig().GetEndpoints() {
		if endpoint.GetName() == s.ServiceConfig().GetName() && endpoint.GetAllowCors() {
			return true
		}
	}
	return false
}
