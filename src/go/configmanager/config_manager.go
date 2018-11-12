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
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

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
	"github.com/google/go-genproto/googleapis/api/servicemanagement/v1"
	"google.golang.org/genproto/googleapis/api/annotations"
	api "google.golang.org/genproto/googleapis/api/serviceconfig"
)

var (
	listenerAddress = flag.String("listener_address", "0.0.0.0", "listener socket ip address")
	clusterAddress  = flag.String("cluster_address", "127.0.0.1", "cluster socket ip address")

	listenerPort = flag.Int("listener_port", 8080, "listener port")
	clusterPort  = flag.Int("cluster_port", 8082, "cluster port")

	clusterConnectTimeout = flag.Duration("cluster_connect_imeout", 20*time.Second, "cluster connect timeout in seconds")

	fetchConfigURL = "https://servicemanagement.googleapis.com/v1/services/$serviceName/configs/$configId?view=FULL"
	node           = "api_proxy"
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
	m := &ConfigManager{
		serviceName: name,
		client:      http.DefaultClient,
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
		return errors.New("fail to make a snapshot")
	}
	m.cache.SetSnapshot(node, *snapshot)
	return nil
}

func (m *ConfigManager) makeSnapshot(serviceConfig *api.Service) (*cache.Snapshot, error) {
	var endpoints, routes []cache.Resource
	serverlistener, httpManager := m.makeListener(serviceConfig)
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
		Name:                 serviceConfig.Apis[0].Name,
		LbPolicy:             v2.Cluster_ROUND_ROBIN,
		ConnectTimeout:       *clusterConnectTimeout,
		Http2ProtocolOptions: &core.Http2ProtocolOptions{},
		Hosts: []*core.Address{
			{Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address: *clusterAddress,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: uint32(*clusterPort),
					},
				},
			},
			},
		},
	}

	snapshot := cache.NewSnapshot(m.configID, endpoints, []cache.Resource{cluster}, routes, []cache.Resource{serverlistener})
	return &snapshot, nil
}

func (m *ConfigManager) makeListener(serviceConfig *api.Service) (*v2.Listener, *hcm.HttpConnectionManager) {
	if len(serviceConfig.GetApis()) == 0 {
		return nil, nil
	}
	httpFilters := []*hcm.HttpFilter{}
	// Add gRPC transcode filter config.
	for _, sourceFile := range serviceConfig.GetSourceInfo().GetSourceFiles() {
		configFile := &servicemanagement.ConfigFile{}
		ptypes.UnmarshalAny(sourceFile, configFile)
		if configFile.GetFileType() == servicemanagement.ConfigFile_FILE_DESCRIPTOR_SET_PROTO {
			configContent := configFile.GetFileContents()
			transcodeConfig := &tc.GrpcJsonTranscoder{
				DescriptorSet: &tc.GrpcJsonTranscoder_ProtoDescriptorBin{
					ProtoDescriptorBin: configContent,
				},
				Services: []string{serviceConfig.Apis[0].Name},
			}
			transcodeConfigStruct, _ := util.MessageToStruct(transcodeConfig)
			transcodeFilter := &hcm.HttpFilter{
				Name:   util.GRPCJSONTranscoder,
				Config: transcodeConfigStruct,
			}
			httpFilters = append(httpFilters, transcodeFilter)
			break
		}
	}
	// TODO(jilinxia): Add Service control filter config.
	// Add JWT Authn filter.
	httpFilters = append(httpFilters, m.makeJwtAuthnFilter(serviceConfig))
	// Add Envoy Router filter so requests are routed upstream.
	routerFilter := &hcm.HttpFilter{
		Name:   util.Router,
		Config: &types.Struct{},
	}
	httpFilters = append(httpFilters, routerFilter)
	return &v2.Listener{
			Address: core.Address{Address: &core.Address_SocketAddress{SocketAddress: &core.SocketAddress{
				Address:       *listenerAddress,
				PortSpecifier: &core.SocketAddress_PortValue{PortValue: uint32(*listenerPort)}}}},
		}, &hcm.HttpConnectionManager{
			CodecType:  hcm.AUTO,
			StatPrefix: "ingress_http",
			RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
				RouteConfig: &v2.RouteConfiguration{
					Name: "local_route",
					VirtualHosts: []route.VirtualHost{
						{
							Name:    "backend",
							Domains: []string{"*"},
							Routes: []route.Route{
								{
									Match: route.RouteMatch{
										PathSpecifier: &route.RouteMatch_Prefix{fmt.Sprintf("/%s", serviceConfig.Apis[0].Name)},
									},
									Action: &route.Route_Route{
										Route: &route.RouteAction{
											ClusterSpecifier: &route.RouteAction_Cluster{serviceConfig.Apis[0].Name},
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
}

func (m *ConfigManager) makeJwtAuthnFilter(serviceConfig *api.Service) *hcm.HttpFilter {
	auth := serviceConfig.GetAuthentication()
	providers := make(map[string]*ac.JwtProvider)
	for _, provider := range auth.GetProviders() {
		jwk, err := fetchJwk(provider.GetJwksUri(), m.client)
		if err != nil {
			glog.Warningf("fetch jwk from issuer got error: %s", err)
			break
		}
		jp := &ac.JwtProvider{
			Issuer:    provider.GetIssuer(),
			Audiences: []string{provider.GetAudiences()},
			// TODO(jilinxia): fetch local token.
			JwksSourceSpecifier: &ac.JwtProvider_LocalJwks{
				LocalJwks: &core.DataSource{
					Specifier: &core.DataSource_InlineString{string(jwk)},
				},
			},
		}
		providers[provider.GetId()] = jp
	}
	rules := []*ac.RequirementRule{}
	// TODO(jilinxia): supports multi rules with RequireAll, RequireAny.
	for _, rule := range auth.GetRules() {
		var require *ac.JwtRequirement
		for _, r := range rule.GetRequirements() {
			audiences := strings.Split(r.GetAudiences(), ",")
			// TODO(jilinxia): adds unit tests when audiences is empty.
			if len(audiences) == 0 {
				require = &ac.JwtRequirement{
					RequiresType: &ac.JwtRequirement_ProviderName{
						ProviderName: r.GetProviderId(),
					},
				}
			} else {
				require = &ac.JwtRequirement{
					RequiresType: &ac.JwtRequirement_ProviderAndAudiences{
						ProviderAndAudiences: &ac.ProviderWithAudiences{
							ProviderName: r.GetProviderId(),
							Audiences:    strings.Split(r.GetAudiences(), ","),
						},
					},
				}
			}
			// TODO(jilinxia): make requirement rule work for open API style.
			m := strings.Split(rule.GetSelector(), ".")
			ruleConfig := &ac.RequirementRule{
				Match: &route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{fmt.Sprintf("/%s/%s", serviceConfig.Apis[0].Name, m[len(m)-1])},
				},
				Requires: require,
			}
			rules = append(rules, ruleConfig)
		}
	}

	jwtAuthentication := &ac.JwtAuthentication{
		Providers: providers,
		Rules:     rules,
	}
	jas, _ := util.MessageToStruct(jwtAuthentication)
	jwtAuthnFilter := &hcm.HttpFilter{
		Name:   "envoy.filters.http.jwt_authn",
		Config: jas,
	}
	return jwtAuthnFilter
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

func (m *ConfigManager) fetchConfig(configId string) (*api.Service, error) {
	token, err := fetchAccessToken()
	if err != nil {
		return nil, fmt.Errorf("fail to get access token")
	}
	path := strings.Replace(fetchConfigURL, "$configId", configId, -1)
	return callServiceManagement(path, m.serviceName, token, m.client)
}

// Helper to convert Json string to protobuf.Any.
type funcResolver func(url string) (proto.Message, error)

func (fn funcResolver) Resolve(url string) (proto.Message, error) {
	return fn(url)
}

var callServiceManagement = func(path, serviceName, token string, client *http.Client) (*api.Service, error) {
	path = strings.Replace(path, "$serviceName", serviceName, -1)
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
	var serviceConfig api.Service
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
