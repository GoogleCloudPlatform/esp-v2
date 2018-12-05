// Copyright 2018 Google Cloud Platform Proxy Authors
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

package configmanager

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"cloudesf.googlesource.com/gcpproxy/src/go/flags"
	commonpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/common"
	scpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/service_control"
	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	ac "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/jwt_authn/v2alpha"
	tc "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/transcoder/v2"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/envoyproxy/go-control-plane/pkg/util"
	"github.com/gogo/protobuf/types"
	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/google/go-genproto/googleapis/api/servicemanagement/v1"
	"google.golang.org/genproto/googleapis/api/annotations"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/genproto/protobuf/api"
)

const (
	statPrefix        = "ingress_http"
	routeName         = "local_route"
	virtualHostName   = "backend"
	fetchConfigSuffix = "/v1/services/$serviceName/configs/$configId?view=FULL"
	tokenUri          = "http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
	serviceControlUri = "https://servicecontrol.googleapis.com/v1/services/"
)

var (
	fetchConfigURL = func(serviceName, configID string) string {
		path := *flags.ServiceManagementURL + fetchConfigSuffix
		path = strings.Replace(path, "$serviceName", serviceName, 1)
		path = strings.Replace(path, "$configId", configID, 1)
		return path
	}
)

// ConfigManager handles service configuration fetching and updating.
// TODO(jilinxia): handles multi service name.
type ConfigManager struct {
	serviceName string
	configID    string
	client      *http.Client
	cache       cache.SnapshotCache
}

// NewConfigManager creates new instance of ConfigManager.
func NewConfigManager(name, configID string) (*ConfigManager, error) {
	var err error
	if name == "" {
		name, err = fetchServiceName()
		if name == "" || err != nil {
			return nil, fmt.Errorf("failed to read metadata with key endpoints-service-name from metadata server")
		}
	}

	if configID == "" {
		configID, err = fetchConfigId()
		if configID == "" || err != nil {
			return nil, fmt.Errorf("failed to read metadata with key endpoints-service-version from metadata server")
		}
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	m := &ConfigManager{
		serviceName: name,
		client:      &http.Client{Transport: tr},
		configID:    configID,
	}
	m.cache = cache.NewSnapshotCache(true, m, m)
	if err := m.init(); err != nil {
		return nil, err
	}
	return m, nil
}

// init should be called when starting up the server.
// It calls ServiceManager Server to fetch the service configuration in order
// to dynamically configure Envoy.
func (m *ConfigManager) init() error {
	serviceConfig, err := m.fetchConfig(m.configID)
	if err != nil {
		// TODO(jilinxia): changes error generation
		return fmt.Errorf("fail to initialize config manager, %s", err)
	}

	snapshot, err := m.makeSnapshot(serviceConfig)
	if err != nil {
		return fmt.Errorf("fail to make a snapshot, %s", err)
	}
	m.cache.SetSnapshot(*flags.Node, *snapshot)
	return nil
}

func (m *ConfigManager) makeSnapshot(serviceConfig *conf.Service) (*cache.Snapshot, error) {
	if len(serviceConfig.GetApis()) == 0 {
		return nil, fmt.Errorf("service config must have one api at least")
	}
	// TODO(jilinxia): supports multi apis.
	if len(serviceConfig.GetApis()) > 1 {
		return nil, fmt.Errorf("not support multi apis yet")
	}
	endpointApi := serviceConfig.Apis[0]
	var backendProtocol ut.BackendProtocol
	switch strings.ToLower(*flags.BackendProtocol) {
	case "http1":
		backendProtocol = ut.HTTP1
	case "http2":
		backendProtocol = ut.HTTP2
	case "grpc":
		backendProtocol = ut.GRPC
	default:
		return nil, fmt.Errorf("unknown backend protocol")
	}

	var endpoints, routes []cache.Resource
	serverlistener, httpManager, err := m.makeListener(serviceConfig, endpointApi, backendProtocol)
	if err != nil {
		return nil, err
	}
	// HTTP filter configuration
	httpFilterConfig, err := util.MessageToStruct(httpManager)
	if err != nil {
		return nil, err
	}
	serverlistener.FilterChains = []listener.FilterChain{{
		Filters: []listener.Filter{{
			Name:   util.HTTPConnectionManager,
			Config: httpFilterConfig,
		}}}}
	cluster := &v2.Cluster{
		Name:           endpointApi.Name,
		LbPolicy:       v2.Cluster_ROUND_ROBIN,
		ConnectTimeout: *flags.ClusterConnectTimeout,
		Hosts: []*core.Address{
			{Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address: *flags.ClusterAddress,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: uint32(*flags.ClusterPort),
					},
				},
			},
			},
		},
	}
	// gRPC and HTTP/2 need this configuration.
	if backendProtocol != ut.HTTP1 {
		cluster.Http2ProtocolOptions = &core.Http2ProtocolOptions{}
	}

	snapshot := cache.NewSnapshot(m.configID, endpoints, []cache.Resource{cluster}, routes, []cache.Resource{serverlistener})
	return &snapshot, nil
}

func (m *ConfigManager) makeListener(serviceConfig *conf.Service, endpointApi *api.Api, backendProtocol ut.BackendProtocol) (*v2.Listener, *hcm.HttpConnectionManager, error) {
	httpFilters := []*hcm.HttpFilter{}

	// Add JWT Authn filter if needed.
	if !*flags.SkipJwtAuthnFilter {
		jwtAuthnFilter := m.makeJwtAuthnFilter(serviceConfig, endpointApi)
		if jwtAuthnFilter != nil {
			httpFilters = append(httpFilters, jwtAuthnFilter)
		}
	}

	// Add service control filter if needed
	if !*flags.SkipServiceControlFilter {
		serviceControlFilter := m.makeServiceControlFilter(serviceConfig)
		if serviceControlFilter != nil {
			httpFilters = append(httpFilters, serviceControlFilter)
		}
	}

	// Add gRPC transcode filter config for gRPC backend.
	if backendProtocol == ut.GRPC {
		transcoderFilter := m.makeTranscoderFilter(serviceConfig, endpointApi)
		if transcoderFilter != nil {
			httpFilters = append(httpFilters, transcoderFilter)
		}
	}

	// Add Envoy Router filter so requests are routed upstream.
	// Router filter should be the last.
	routerFilter := &hcm.HttpFilter{
		Name:   util.Router,
		Config: &types.Struct{},
	}
	httpFilters = append(httpFilters, routerFilter)

	httpConMgr := &hcm.HttpConnectionManager{
		CodecType:  hcm.AUTO,
		StatPrefix: statPrefix,
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: &v2.RouteConfiguration{
				Name: routeName,
				VirtualHosts: []route.VirtualHost{
					{
						Name:    virtualHostName,
						Domains: []string{"*"},
						Routes: []route.Route{
							{
								Match: route.RouteMatch{
									PathSpecifier: &route.RouteMatch_Prefix{
										Prefix: "/",
									},
								},
								Action: &route.Route_Route{
									Route: &route.RouteAction{
										ClusterSpecifier: &route.RouteAction_Cluster{
											Cluster: endpointApi.Name},
									},
								},
							},
						},
					},
				},
			},
		},
		HttpFilters: httpFilters,
	}

	return &v2.Listener{
		Address: core.Address{Address: &core.Address_SocketAddress{SocketAddress: &core.SocketAddress{
			Address:       *flags.ListenerAddress,
			PortSpecifier: &core.SocketAddress_PortValue{PortValue: uint32(*flags.ListenerPort)}}}},
	}, httpConMgr, nil
}

func (m *ConfigManager) makeTranscoderFilter(serviceConfig *conf.Service, endpointApi *api.Api) *hcm.HttpFilter {
	for _, sourceFile := range serviceConfig.GetSourceInfo().GetSourceFiles() {
		configFile := &servicemanagement.ConfigFile{}
		ptypes.UnmarshalAny(sourceFile, configFile)
		if configFile.GetFileType() == servicemanagement.ConfigFile_FILE_DESCRIPTOR_SET_PROTO {
			configContent := configFile.GetFileContents()
			transcodeConfig := &tc.GrpcJsonTranscoder{
				DescriptorSet: &tc.GrpcJsonTranscoder_ProtoDescriptorBin{
					ProtoDescriptorBin: configContent,
				},
				Services: []string{endpointApi.Name},
			}
			transcodeConfigStruct, _ := util.MessageToStruct(transcodeConfig)
			transcodeFilter := &hcm.HttpFilter{
				Name:   util.GRPCJSONTranscoder,
				Config: transcodeConfigStruct,
			}
			return transcodeFilter
		}
	}
	return nil
}

func (m *ConfigManager) makeJwtAuthnFilter(serviceConfig *conf.Service, endpointApi *api.Api) *hcm.HttpFilter {
	if serviceConfig == nil {
		glog.Warning("unexpected empty service config")
		return nil
	}
	auth := serviceConfig.GetAuthentication()
	if len(auth.GetProviders()) == 0 {
		return nil
	}
	providers := make(map[string]*ac.JwtProvider)
	for _, provider := range auth.GetProviders() {
		jwk, err := fetchJwk(provider.GetJwksUri(), m.client)
		if err != nil {
			glog.Warningf("fetch jwk from issuer got error: %s", err)
			break
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
			break
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
		m := strings.Split(rule.GetSelector(), ".")
		ruleConfig := &ac.RequirementRule{
			Match: &route.RouteMatch{
				PathSpecifier: &route.RouteMatch_Prefix{
					Prefix: fmt.Sprintf("/%s/%s", endpointApi.Name, m[len(m)-1]),
				},
			},
			Requires: requires,
		}
		rules = append(rules, ruleConfig)
	}

	jwtAuthentication := &ac.JwtAuthentication{
		Providers: providers,
		Rules:     rules,
	}
	jas, _ := util.MessageToStruct(jwtAuthentication)
	jwtAuthnFilter := &hcm.HttpFilter{
		Name:   ut.JwtAuthn,
		Config: jas,
	}
	return jwtAuthnFilter
}

func (m *ConfigManager) makeServiceControlFilter(serviceConfig *conf.Service) *hcm.HttpFilter {
	if serviceConfig.GetName() == "" || serviceConfig.GetControl().GetEnvironment() == "" {
		return nil
	}

	service := &scpb.Service{
		ServiceName: serviceConfig.GetName(),
		TokenUri: &scpb.HttpUri{
			Uri:     tokenUri,
			Cluster: "gcp_metadata_cluster",
			Timeout: &duration.Duration{Seconds: 5},
		},
		ServiceControlUri: &scpb.HttpUri{
			Uri:     serviceControlUri,
			Cluster: "service_control_cluster",
			Timeout: &duration.Duration{Seconds: 5},
		},
	}

	rulesMap := make(map[string][]*scpb.ServiceControlRule)
	for _, api := range serviceConfig.GetApis() {
		for _, method := range api.GetMethods() {
			grpcUri := fmt.Sprintf("/%s/%s", api.GetName(), method.GetName())
			selector := fmt.Sprintf("%s.%s", api.GetName(), method.GetName())
			rulesMap[selector] = []*scpb.ServiceControlRule{
				&scpb.ServiceControlRule{
					Requires: &scpb.Requirement{
						ServiceName:   serviceConfig.GetName(),
						OperationName: selector,
					},
					Pattern: &commonpb.Pattern{
						UriTemplate: grpcUri,
						HttpMethod:  "POST",
					},
				},
			}
		}
	}

	for _, httpRule := range serviceConfig.GetHttp().GetRules() {
		scRules := rulesMap[httpRule.GetSelector()]
		switch httpPattern := httpRule.GetPattern().(type) {
		case *annotations.HttpRule_Get:
			scRules = append(scRules, &scpb.ServiceControlRule{
				Requires: &scpb.Requirement{
					ServiceName:   serviceConfig.GetName(),
					OperationName: httpRule.GetSelector(),
				},
				Pattern: &commonpb.Pattern{
					UriTemplate: httpPattern.Get,
					HttpMethod:  "GET",
				},
			})
		case *annotations.HttpRule_Put:
			scRules = append(scRules, &scpb.ServiceControlRule{
				Requires: &scpb.Requirement{
					ServiceName:   serviceConfig.GetName(),
					OperationName: httpRule.GetSelector(),
				},
				Pattern: &commonpb.Pattern{
					UriTemplate: httpPattern.Put,
					HttpMethod:  "PUT",
				},
			})
		case *annotations.HttpRule_Post:
			scRules = append(scRules, &scpb.ServiceControlRule{
				Requires: &scpb.Requirement{
					ServiceName:   serviceConfig.GetName(),
					OperationName: httpRule.GetSelector(),
				},
				Pattern: &commonpb.Pattern{
					UriTemplate: httpPattern.Post,
					HttpMethod:  "POST",
				},
			})
		case *annotations.HttpRule_Delete:
			scRules = append(scRules, &scpb.ServiceControlRule{
				Requires: &scpb.Requirement{
					ServiceName:   serviceConfig.GetName(),
					OperationName: httpRule.GetSelector(),
				},
				Pattern: &commonpb.Pattern{
					UriTemplate: httpPattern.Delete,
					HttpMethod:  "DELETE",
				},
			})
		case *annotations.HttpRule_Patch:
			scRules = append(scRules, &scpb.ServiceControlRule{
				Requires: &scpb.Requirement{
					ServiceName:   serviceConfig.GetName(),
					OperationName: httpRule.GetSelector(),
				},
				Pattern: &commonpb.Pattern{
					UriTemplate: httpPattern.Patch,
					HttpMethod:  "PATCH",
				},
			})
		}
		rulesMap[httpRule.GetSelector()] = scRules

	}

	for _, usageRule := range serviceConfig.GetUsage().GetRules() {
		scRules := rulesMap[usageRule.GetSelector()]
		for _, scRule := range scRules {
			scRule.Requires.ApiKey = &scpb.APIKeyRequirement{
				AllowWithoutApiKey: usageRule.GetAllowUnregisteredCalls(),
				ApiKeys: []*scpb.APIKey{
					&scpb.APIKey{
						Key: &scpb.APIKey_Query{ut.APIKeyQuery},
					},
					&scpb.APIKey{
						Key: &scpb.APIKey_Header{ut.APIKeyHeader},
					},
				},
			}
		}
	}

	filterConfig := &scpb.FilterConfig{
		Services:    []*scpb.Service{service},
		ServiceName: serviceConfig.GetName(),
		ServiceControlUri: &scpb.HttpUri{
			Uri:     serviceControlUri,
			Cluster: "service_control_cluster",
			Timeout: &duration.Duration{Seconds: 5},
		},
	}

	for _, rules := range rulesMap {
		for _, rule := range rules {
			filterConfig.Rules = append(filterConfig.Rules, rule)
		}
	}

	scs, _ := util.MessageToStruct(filterConfig)
	filter := &hcm.HttpFilter{
		Name:   ut.ServiceControl,
		Config: scs,
	}
	return filter
}

// Implements the ID method for HashNode interface.
func (m *ConfigManager) ID(node *core.Node) string {
	return node.GetId()
}

// Implements the Infof method for Log interface.
func (m *ConfigManager) Infof(format string, args ...interface{}) { glog.Infof(format, args...) }

// Implements the Errorf method for Log interface.
func (m *ConfigManager) Errorf(format string, args ...interface{}) { glog.Errorf(format, args...) }

func (m *ConfigManager) Cache() cache.Cache { return m.cache }

func (m *ConfigManager) fetchConfig(configId string) (*conf.Service, error) {
	token, _, err := fetchAccessToken()
	if err != nil {
		return nil, fmt.Errorf("fail to get access token")
	}

	return callServiceManagement(fetchConfigURL(m.serviceName, configId), token, m.client)
}

// Helper to convert Json string to protobuf.Any.
type funcResolver func(url string) (proto.Message, error)

func (fn funcResolver) Resolve(url string) (proto.Message, error) {
	return fn(url)
}

var callServiceManagement = func(path, token string, client *http.Client) (*conf.Service, error) {
	req, _ := http.NewRequest("GET", path, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http call to service management returns not 200 OK: %v", resp.Status)
	}
	defer resp.Body.Close()
	resolver := funcResolver(func(url string) (proto.Message, error) {
		switch url {
		case "type.googleapis.com/google.api.servicemanagement.v1.ConfigFile":
			return new(servicemanagement.ConfigFile), nil
		case "type.googleapis.com/google.api.HttpRule":
			return new(annotations.HttpRule), nil
		default:
			return nil, fmt.Errorf("unexpected protobuf.Any type")
		}
	})
	unmarshaler := &jsonpb.Unmarshaler{
		AllowUnknownFields: true,
		AnyResolver:        resolver,
	}
	var serviceConfig conf.Service
	if err = unmarshaler.Unmarshal(resp.Body, &serviceConfig); err != nil {
		return nil, fmt.Errorf("fail to unmarshal serviceConfig: %s", err)
	}
	return &serviceConfig, nil
}

var fetchJwk = func(path string, client *http.Client) ([]byte, error) {
	req, _ := http.NewRequest("GET", path, nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching JWK returns not 200 OK: %v", resp.Status)
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}
