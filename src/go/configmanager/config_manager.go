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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloudesf.googlesource.com/gcpproxy/src/go/flags"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/envoyproxy/go-control-plane/pkg/util"
	"github.com/gogo/protobuf/types"
	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/duration"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/genproto/protobuf/api"

	bapb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/backend_auth"
	commonpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/common"
	pmpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/path_matcher"
	scpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/service_control"
	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	ac "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/jwt_authn/v2alpha"
	tc "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/transcoder/v2"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	sm "github.com/google/go-genproto/googleapis/api/servicemanagement/v1"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

const (
	statPrefix                = "ingress_http"
	routeName                 = "local_route"
	virtualHostName           = "backend"
	fetchConfigSuffix         = "/v1/services/$serviceName/configs/$configId?view=FULL"
	fetchRolloutsSuffix       = "/v1/services/$serviceName/rollouts?filter=status=SUCCESS"
	serviceControlClusterName = "service-control-cluster"
)

var (
	fetchConfigURL = func(serviceName, configID string) string {
		path := *flags.ServiceManagementURL + fetchConfigSuffix
		path = strings.Replace(path, "$serviceName", serviceName, 1)
		path = strings.Replace(path, "$configId", configID, 1)
		return path
	}
	fetchRolloutsURL = func(serviceName string) string {
		path := *flags.ServiceManagementURL + fetchRolloutsSuffix
		path = strings.Replace(path, "$serviceName", serviceName, 1)
		return path
	}
	checkNewRolloutInterval = 60 * time.Second
)

// ConfigManager handles service configuration fetching and updating.
// TODO(jilinxia): handles multi service name.
type ConfigManager struct {
	serviceName       string
	curRolloutID      string
	curConfigID       string
	serviceControlURI string
	serviceConfig     *conf.Service
	// httpPathMap stores all operations to http path pairs.
	httpPathMap            map[string]*httpRule
	httpPathWithOptionsSet map[string]bool
	backendRoutingInfos    []backendRoutingInfo
	client                 *http.Client
	cache                  cache.SnapshotCache
	checkRolloutsTicker    *time.Ticker
	gcpAttributes          *scpb.GcpAttributes
}

type httpRule struct {
	path   string
	method string
}

type backendRoutingInfo struct {
	selector        string
	translationType conf.BackendRule_PathTranslation
	backend         backendInfo
}

type backendInfo struct {
	name     string
	hostname string
	port     uint32
}

// NewConfigManager creates new instance of ConfigManager.
func NewConfigManager() (*ConfigManager, error) {
	var err error
	name := *flags.ServiceName
	if name == "" && *flags.CheckMetadata {
		name, err = fetchServiceName()
		if name == "" || err != nil {
			return nil, fmt.Errorf("failed to read metadata with key endpoints-service-name from metadata server")
		}
	} else if name == "" && !*flags.CheckMetadata {
		return nil, fmt.Errorf("service name is not specified")
	}
	var rolloutStrategy string
	rolloutStrategy = *flags.RolloutStrategy
	// try to fetch from metadata, if not found, set to fixed instead of throwing an error
	if rolloutStrategy == "" && *flags.CheckMetadata {
		rolloutStrategy, err = fetchRolloutStrategy()
	}
	if rolloutStrategy == "" {
		rolloutStrategy = ut.FixedRolloutStrategy
	}
	if !(rolloutStrategy == ut.FixedRolloutStrategy || rolloutStrategy == ut.ManagedRolloutStrategy) {
		return nil, fmt.Errorf(`failed to set rollout strategy. It must be either "managed" or "fixed"`)
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	m := &ConfigManager{
		serviceName:   name,
		client:        &http.Client{Transport: tr},
		gcpAttributes: fetchGCPAttributes(),
	}

	m.cache = cache.NewSnapshotCache(true, m, m)

	if rolloutStrategy == ut.ManagedRolloutStrategy {
		// try to fetch rollouts and get newest config, if failed, NewConfigManager exits with failure
		if err := m.loadConfigFromRollouts(); err != nil {
			return nil, err
		}
	} else {
		// rollout strategy is fixed mode
		configID := *flags.ConfigID
		if configID == "" {
			if *flags.CheckMetadata {
				configID, err = fetchConfigId()
				if configID == "" || err != nil {
					return nil, fmt.Errorf("failed to read metadata with key endpoints-service-version from metadata server")
				}
			} else {
				return nil, fmt.Errorf("service config id is not specified")
			}
		}
		m.curConfigID = configID
		if err := m.updateSnapshot(); err != nil {
			return nil, err
		}
	}
	glog.Infof("create new ConfigManager for service (%v) with configuration id (%v), %v rollout strategy",
		m.serviceName, m.curConfigID, rolloutStrategy)

	if rolloutStrategy == ut.ManagedRolloutStrategy {
		go func() {
			glog.Infof("start checking new rollouts every %v seconds", checkNewRolloutInterval)
			m.checkRolloutsTicker = time.NewTicker(checkNewRolloutInterval)
			for range m.checkRolloutsTicker.C {
				m.Infof("check new rollouts for service %v", m.serviceName)
				// only log error and keep checking when fetching rollouts and getting newest config fail
				if err := m.loadConfigFromRollouts(); err != nil {
					glog.Errorf("error occurred when checking new rollouts, %v", err)
				}
			}
		}()
	}
	return m, nil
}

func (m *ConfigManager) loadConfigFromRollouts() error {
	var err error
	var listServiceRolloutsResponse *sm.ListServiceRolloutsResponse
	listServiceRolloutsResponse, err = m.fetchRollouts()
	if err != nil {
		return fmt.Errorf("fail to get rollouts, %s", err)
	}
	m.Infof("get rollouts %v", listServiceRolloutsResponse)
	if len(listServiceRolloutsResponse.Rollouts) == 0 {
		return fmt.Errorf("no active rollouts")
	}
	newRolloutID := listServiceRolloutsResponse.Rollouts[0].RolloutId
	if m.curRolloutID == newRolloutID {
		return nil
	}
	m.curRolloutID = newRolloutID
	m.Infof("found new rollout id %v for service %v, %v", m.curRolloutID, m.serviceName)

	trafficPercentStrategy := listServiceRolloutsResponse.Rollouts[0].GetTrafficPercentStrategy()
	trafficPercentMap := trafficPercentStrategy.GetPercentages()
	if len(trafficPercentMap) == 0 {
		return fmt.Errorf("no active rollouts")
	}
	var newConfigID string
	currentMaxPercent := 0.0
	// take config ID with max traffic percent as new config ID
	for k, v := range trafficPercentMap {
		if v > currentMaxPercent {
			newConfigID = k
			currentMaxPercent = v
		}
	}
	if newConfigID == m.curConfigID {
		m.Infof("no new configuration to load for service %v, current configuration id %v", m.serviceName, m.curConfigID)
		return nil
	}
	if !(math.Abs(100.0-currentMaxPercent) < 1e-9) {
		glog.Warningf("though traffic percentage of configuration %v is %v%%, set it to 100%%", newConfigID, currentMaxPercent)
	}
	m.curConfigID = newConfigID
	m.Infof("found new configuration id %v for service %v", m.curConfigID, m.serviceName)
	return m.updateSnapshot()
}

// updateSnapshot should be called when starting up the server.
// It calls ServiceManager Server to fetch the service configuration in order
// to dynamically configure Envoy.
func (m *ConfigManager) updateSnapshot() error {
	var err error
	m.serviceConfig, err = m.fetchConfig(m.curConfigID)
	if err != nil {
		return fmt.Errorf("fail to initialize config manager, %s", err)
	}
	// This should be always true for the one fetched from production servicemanagment
	// But it may not be so for the ones fetched from mock servicemanagement for integation tests.
	// Hence use the one from serviceConfig to override it.
	m.serviceName = m.serviceConfig.GetName()
	m.Infof("got service configuration: %v", m.serviceConfig)
	snapshot, err := m.makeSnapshot()
	if err != nil {
		return fmt.Errorf("fail to make a snapshot, %s", err)
	}
	m.cache.SetSnapshot(*flags.Node, *snapshot)
	return nil
}

func (m *ConfigManager) makeSnapshot() (*cache.Snapshot, error) {
	if m.serviceConfig == nil {
		return nil, fmt.Errorf("unexpected empty service config")
	}
	if len(m.serviceConfig.GetApis()) == 0 {
		return nil, fmt.Errorf("service config must have one api at least")
	}
	// TODO(jilinxia): supports multi apis.
	if len(m.serviceConfig.GetApis()) > 1 {
		return nil, fmt.Errorf("not support multi apis yet")
	}

	endpointApi := m.serviceConfig.Apis[0]
	m.Infof("making configuration for api: %v", endpointApi.GetName())
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

	m.initHttpPathMap()

	var endpoints, routes, clusters []cache.Resource
	backendCluster, err := m.makeBackendCluster(endpointApi, backendProtocol)
	if err != nil {
		return nil, err
	}
	if backendCluster != nil {
		clusters = append(clusters, backendCluster)
	}

	// Note: makeServiceControlCluster should be called before makeListener
	// as makeServiceControlFilter is using m.serviceControlURI assigned by
	// makeServiceControlCluster
	scCluster, err := m.makeServiceControlCluster()
	if err != nil {
		return nil, err
	}
	if scCluster != nil {
		clusters = append(clusters, scCluster)
	}

	if *flags.EnableBackendRouting {
		dynamicRoutingBackendMap, err := m.processBackendRoutingInfo()
		if err != nil {
			return nil, err
		}
		m.addBackendRoutingClusters(dynamicRoutingBackendMap, &clusters)
	}

	m.Infof("adding Listener configuration for api: %v", endpointApi.GetName())
	serverlistener, httpManager, err := m.makeListener(endpointApi, backendProtocol)
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
			Name:       util.HTTPConnectionManager,
			ConfigType: &listener.Filter_Config{httpFilterConfig},
		}}}}

	snapshot := cache.NewSnapshot(m.curConfigID, endpoints, clusters, routes, []cache.Resource{serverlistener})
	glog.Infof("Envoy Dynamic Configuration is cached for service: %v", m.serviceName)
	return &snapshot, nil
}

func (m *ConfigManager) addBackendRoutingClusters(dynamicRoutingBackendMap map[string]backendInfo, clusters *[]cache.Resource) {
	for _, v := range dynamicRoutingBackendMap {
		c := &v2.Cluster{
			Name:           v.name,
			LbPolicy:       v2.Cluster_ROUND_ROBIN,
			ConnectTimeout: *flags.ClusterConnectTimeout,
			Type:           v2.Cluster_LOGICAL_DNS,
			Hosts: []*core.Address{
				{Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Address: v.hostname,
						PortSpecifier: &core.SocketAddress_PortValue{
							PortValue: v.port,
						},
					},
				},
				},
			},
			TlsContext: &auth.UpstreamTlsContext{
				Sni: v.hostname,
			},
		}
		*clusters = append(*clusters, c)
		m.Infof("Add backend routing cluster configuration for %v: %v", v.name, c)
	}
}

func (m *ConfigManager) makeBackendCluster(endpointApi *api.Api, backendProtocol ut.BackendProtocol) (*v2.Cluster, error) {
	c := &v2.Cluster{
		Name:           endpointApi.Name,
		LbPolicy:       v2.Cluster_ROUND_ROBIN,
		ConnectTimeout: *flags.ClusterConnectTimeout,
		Type:           v2.Cluster_STRICT_DNS,
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
		c.Http2ProtocolOptions = &core.Http2ProtocolOptions{}
	}
	m.Infof("Backend cluster configuration for service %s: %v", endpointApi.GetName(), c)
	return c, nil
}

func (m *ConfigManager) makeServiceControlCluster() (*v2.Cluster, error) {
	uri := m.serviceConfig.GetControl().GetEnvironment()
	if uri == "" {
		return nil, nil
	}

	// The assumption about control.environment field. Its format:
	//   [scheme://] +  host + [:port]
	// * It should not have any path part
	// * If scheme is missed, https is the default

	// Default is https
	scheme := "https"
	host := uri
	arr := strings.Split(uri, "://")
	if len(arr) == 2 {
		scheme = arr[0]
		host = arr[1]
	}

	// This is used in service_control_uri.uri in service control fitler.
	// Not path part, append /v1/services/ directly on host
	m.serviceControlURI = scheme + "://" + host + "/v1/services/"

	arr = strings.Split(host, ":")
	var port int
	if len(arr) == 2 {
		var err error
		port, err = strconv.Atoi(arr[1])
		if err != nil {
			return nil, fmt.Errorf("Invalid port: %s, got err: %s", arr[1], err)
		}
		host = arr[0]
	} else {
		if scheme == "http" {
			port = 80
		} else {
			port = 443
		}
	}

	c := &v2.Cluster{
		Name:            serviceControlClusterName,
		LbPolicy:        v2.Cluster_ROUND_ROBIN,
		ConnectTimeout:  5 * time.Second,
		DnsLookupFamily: v2.Cluster_V4_ONLY,
		Type:            v2.Cluster_LOGICAL_DNS,
		Hosts: []*core.Address{
			{Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address: host,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: uint32(port),
					},
				},
			},
			},
		},
	}

	if scheme == "https" {
		c.TlsContext = &auth.UpstreamTlsContext{
			Sni: host,
		}
	}
	m.Infof("adding cluster Configuration for uri: %s: %v", uri, c)
	return c, nil
}

func (m *ConfigManager) initHttpPathMap() {
	m.httpPathMap = make(map[string]*httpRule)
	m.httpPathWithOptionsSet = make(map[string]bool)
	for _, r := range m.serviceConfig.GetHttp().GetRules() {
		var rule *httpRule
		switch r.GetPattern().(type) {
		case *annotations.HttpRule_Get:
			rule = &httpRule{
				path:   r.GetGet(),
				method: ut.GET,
			}
		case *annotations.HttpRule_Put:
			rule = &httpRule{
				path:   r.GetPut(),
				method: ut.PUT,
			}
		case *annotations.HttpRule_Post:
			rule = &httpRule{
				path:   r.GetPost(),
				method: ut.POST,
			}
		case *annotations.HttpRule_Delete:
			rule = &httpRule{
				path:   r.GetDelete(),
				method: ut.DELETE,
			}
		case *annotations.HttpRule_Patch:
			rule = &httpRule{
				path:   r.GetPatch(),
				method: ut.PATCH,
			}
		case *annotations.HttpRule_Custom:
			rule = &httpRule{
				path:   r.GetCustom().GetPath(),
				method: r.GetCustom().GetKind(),
			}
		default:
			glog.Warning("unsupported http method")
		}

		if rule.method == ut.OPTIONS {
			m.httpPathWithOptionsSet[rule.path] = true
		}
		m.httpPathMap[r.GetSelector()] = rule
	}
}

func (m *ConfigManager) extractBackendAddress(address string) (string, uint32, error) {
	backendUrl, err := url.Parse(address)
	if err != nil {
		return "", 0, err
	}
	if backendUrl.Scheme != "https" {
		return "", 0, fmt.Errorf("dynamic routing only supports HTTPS")
	}
	hostname := backendUrl.Hostname()
	if net.ParseIP(hostname) != nil {
		return "", 0, fmt.Errorf("dynamic routing only supports domain name, got IP address: %v", hostname)
	}
	var port uint32 = 443
	if backendUrl.Port() != "" {
		// for cases like "https://example.org:8080"
		var port64 uint64
		var err error
		if port64, err = strconv.ParseUint(backendUrl.Port(), 10, 32); err != nil {
			return "", 0, err
		}
		port = uint32(port64)
	}
	return hostname, port, nil
}

func (m *ConfigManager) processBackendRoutingInfo() (map[string]backendInfo, error) {
	dynamicRoutingBackendMap := make(map[string]backendInfo)
	for _, r := range m.serviceConfig.Backend.GetRules() {
		// for CONSTANT_ADDRESS and APPEND_PATH_TO_ADDRESS
		if r.PathTranslation != conf.BackendRule_PATH_TRANSLATION_UNSPECIFIED {
			var err error
			var hostname string
			var port uint32
			hostname, port, err = m.extractBackendAddress(r.Address)
			if err != nil {
				return nil, err
			}
			address := fmt.Sprintf("%v:%v", hostname, port)
			if _, exist := dynamicRoutingBackendMap[address]; !exist {
				backendSelector := fmt.Sprintf("DynamicRouting.%v", len(dynamicRoutingBackendMap))
				dynamicRoutingBackendMap[address] = backendInfo{
					name:     backendSelector,
					hostname: hostname,
					port:     port,
				}
			}
			m.backendRoutingInfos = append(m.backendRoutingInfos, backendRoutingInfo{
				selector:        r.Selector,
				translationType: r.PathTranslation,
				backend:         dynamicRoutingBackendMap[address],
			})
		}
	}
	return dynamicRoutingBackendMap, nil
}

func (m *ConfigManager) makeListener(endpointApi *api.Api, backendProtocol ut.BackendProtocol) (*v2.Listener, *hcm.HttpConnectionManager, error) {
	httpFilters := []*hcm.HttpFilter{}

	if *flags.CorsPreset == "basic" || *flags.CorsPreset == "cors_with_regex" {
		corsFilter := &hcm.HttpFilter{
			Name: util.CORS,
		}
		httpFilters = append(httpFilters, corsFilter)
		m.Infof("adding CORS Filter config: %v", corsFilter)
	}

	// TODO(kyuc): once we verify that path matcher works as intended. Path
	// Mathcher filter can be always enabled since the following filters will
	// depend on it.
	//  * Service Control (not using dynamic metadata yet, but will be using it)
	//  * JWT Auth filter
	//  * Backend Auth Filter (currently uses dynamic metadata)
	//  * Dynamic Routing Filter (name TBD -- will be using dynamic metadata )
	if *flags.EnableBackendRouting {
		pathMathcherFilter := m.makePathMatcherFilter(endpointApi, backendProtocol)
		if pathMathcherFilter != nil {
			httpFilters = append(httpFilters, pathMathcherFilter)
			m.Infof("adding Path Matcher Filter config: %v", pathMathcherFilter)
		}
	}

	// Add JWT Authn filter if needed.
	if !*flags.SkipJwtAuthnFilter {
		jwtAuthnFilter := m.makeJwtAuthnFilter(endpointApi, backendProtocol)
		if jwtAuthnFilter != nil {
			httpFilters = append(httpFilters, jwtAuthnFilter)
			m.Infof("adding JWT Authn Filter config: %v", jwtAuthnFilter)
		}
	}

	// Add Service Control filter if needed.
	if !*flags.SkipServiceControlFilter {
		serviceControlFilter := m.makeServiceControlFilter(endpointApi, backendProtocol)
		if serviceControlFilter != nil {
			httpFilters = append(httpFilters, serviceControlFilter)
			m.Infof("adding Service Control Filter config: %v", serviceControlFilter)
		}
	}

	// Add gRPC Transcoder filter and gRPCWeb filter configs for gRPC backend.
	if backendProtocol == ut.GRPC {
		transcoderFilter := m.makeTranscoderFilter(endpointApi)
		if transcoderFilter != nil {
			httpFilters = append(httpFilters, transcoderFilter)
			m.Infof("adding Transcoder Filter config: %v", transcoderFilter)
		}

		grpcWebFilter := &hcm.HttpFilter{
			Name:       util.GRPCWeb,
			ConfigType: &hcm.HttpFilter_Config{&types.Struct{}},
		}
		httpFilters = append(httpFilters, grpcWebFilter)
	}

	// Add Backend Auth filter if needed.
	if *flags.EnableBackendRouting {
		backendAuthFilter := m.makeBackendAuthFilter()
		httpFilters = append(httpFilters, backendAuthFilter)
	}

	// Add Envoy Router filter so requests are routed upstream.
	// Router filter should be the last.
	routerFilter := &hcm.HttpFilter{
		Name:       util.Router,
		ConfigType: &hcm.HttpFilter_Config{&types.Struct{}},
	}
	httpFilters = append(httpFilters, routerFilter)

	route, err := makeRouteConfig(endpointApi)
	if err != nil {
		return nil, nil, fmt.Errorf("makeHttpConnectionManagerRouteConfig got err: %s", err)
	}

	if *flags.EnableBackendRouting {
		if err := m.addDynamicRoutingConfig(route); err != nil {
			return nil, nil, err
		}
	}
	httpConMgr := &hcm.HttpConnectionManager{
		CodecType:  hcm.AUTO,
		StatPrefix: statPrefix,
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: route,
		},
	}

	m.Infof("adding Http Connection Manager config: %v", httpConMgr)
	httpConMgr.HttpFilters = httpFilters

	return &v2.Listener{
		Address: core.Address{Address: &core.Address_SocketAddress{SocketAddress: &core.SocketAddress{
			Address:       *flags.ListenerAddress,
			PortSpecifier: &core.SocketAddress_PortValue{PortValue: uint32(*flags.ListenerPort)}}}},
	}, httpConMgr, nil
}

func (m *ConfigManager) addDynamicRoutingConfig(routeConfig *v2.RouteConfiguration) error {
	var backendRoutes []route.Route
	for _, v := range m.backendRoutingInfos {
		var routeMatcher *route.RouteMatch
		if routeMatcher = m.makeHttpRouteMatcher(v.selector); routeMatcher == nil {
			return fmt.Errorf("error making HTTP route matcher for selector: %v", v.selector)
		}
		r := route.Route{
			Match: *routeMatcher,
			Action: &route.Route_Route{
				Route: &route.RouteAction{
					ClusterSpecifier: &route.RouteAction_Cluster{
						Cluster: v.backend.name,
					},
					HostRewriteSpecifier: &route.RouteAction_HostRewrite{
						HostRewrite: v.backend.hostname,
					},
				},
			},
		}
		backendRoutes = append(backendRoutes, r)
	}
	// has to be backend routing first because the first route that matches will be used in envoy route filter
	routeConfig.VirtualHosts[0].Routes = append(backendRoutes, routeConfig.VirtualHosts[0].Routes...)
	return nil
}

func makeRouteConfig(endpointApi *api.Api) (*v2.RouteConfiguration, error) {
	var virtualHosts []route.VirtualHost
	host := route.VirtualHost{
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
	}

	switch *flags.CorsPreset {
	case "basic":
		org := *flags.CorsAllowOrigin
		if org == "" {
			return nil, fmt.Errorf("cors_allow_origin cannot be empty when cors_preset=basic")
		}
		host.Cors = &route.CorsPolicy{
			AllowOrigin: []string{org},
		}
	case "cors_with_regex":
		orgReg := *flags.CorsAllowOriginRegex
		if orgReg == "" {
			return nil, fmt.Errorf("cors_allow_origin_regex cannot be empty when cors_preset=cors_with_regex")
		}
		host.Cors = &route.CorsPolicy{
			AllowOriginRegex: []string{orgReg},
		}
	case "":
		if *flags.CorsAllowMethods != "" || *flags.CorsAllowHeaders != "" || *flags.CorsExposeHeaders != "" || *flags.CorsAllowCredentials {
			return nil, fmt.Errorf("cors_preset must be set in order to enable CORS support")
		}
	default:
		return nil, fmt.Errorf(`cors_preset must be either "basic" or "cors_with_regex"`)
	}

	if host.GetCors() != nil {
		host.GetCors().AllowMethods = *flags.CorsAllowMethods
		host.GetCors().AllowHeaders = *flags.CorsAllowHeaders
		host.GetCors().ExposeHeaders = *flags.CorsExposeHeaders
		host.GetCors().AllowCredentials = &types.BoolValue{Value: *flags.CorsAllowCredentials}
	}

	virtualHosts = append(virtualHosts, host)
	return &v2.RouteConfiguration{
		Name:         routeName,
		VirtualHosts: virtualHosts,
	}, nil
}

func hasPathParameter(httpPattern string) bool {
	return strings.ContainsRune(httpPattern, '{')
}

func (m *ConfigManager) makePathMatcherFilter(endpointApi *api.Api, backendProtocol ut.BackendProtocol) *hcm.HttpFilter {
	rules := []*pmpb.PathMatcherRule{}
	if backendProtocol == ut.GRPC {
		for _, method := range endpointApi.GetMethods() {
			selector := fmt.Sprintf("%s.%s", endpointApi.GetName(), method.GetName())
			rules = append(rules, &pmpb.PathMatcherRule{
				Operation: selector,
				Pattern: &commonpb.Pattern{
					UriTemplate: fmt.Sprintf("/%s/%s", endpointApi.GetName(), method.GetName()),
					HttpMethod:  ut.POST,
				},
			})
		}
	}

	constantAddressRules := make(map[string]bool)
	for _, rule := range m.serviceConfig.GetBackend().GetRules() {
		if rule.GetPathTranslation() == conf.BackendRule_CONSTANT_ADDRESS {
			constantAddressRules[rule.GetSelector()] = true
		}
	}

	for _, httpRule := range m.serviceConfig.GetHttp().GetRules() {
		var newPattern *commonpb.Pattern
		switch httpPattern := httpRule.GetPattern().(type) {
		case *annotations.HttpRule_Get:
			newPattern = &commonpb.Pattern{
				UriTemplate: httpPattern.Get,
				HttpMethod:  ut.GET,
			}
		case *annotations.HttpRule_Put:
			newPattern = &commonpb.Pattern{
				UriTemplate: httpPattern.Put,
				HttpMethod:  ut.PUT,
			}
		case *annotations.HttpRule_Post:
			newPattern = &commonpb.Pattern{
				UriTemplate: httpPattern.Post,
				HttpMethod:  ut.POST,
			}
		case *annotations.HttpRule_Delete:
			newPattern = &commonpb.Pattern{
				UriTemplate: httpPattern.Delete,
				HttpMethod:  ut.DELETE,
			}
		case *annotations.HttpRule_Patch:
			newPattern = &commonpb.Pattern{
				UriTemplate: httpPattern.Patch,
				HttpMethod:  ut.PATCH,
			}
		}

		newRule := &pmpb.PathMatcherRule{
			Operation: httpRule.GetSelector(),
			Pattern:   newPattern,
		}

		isConstantAddress := constantAddressRules[httpRule.GetSelector()]
		if isConstantAddress && hasPathParameter(newPattern.UriTemplate) {
			newRule.ExtractPathParameters = true
		}

		rules = append(rules, newRule)
	}

	// Create snake name to JSON name mapping.
	var segmentNames []*pmpb.SegmentName
	for _, t := range m.serviceConfig.GetTypes() {
		for _, f := range t.GetFields() {
			if strings.ContainsRune(f.GetName(), '_') {
				segmentNames = append(segmentNames, &pmpb.SegmentName{
					SnakeName: f.GetName(),
					JsonName:  f.GetJsonName(),
				})
			}
		}
	}

	pathMathcherConfig := &pmpb.FilterConfig{Rules: rules}
	if len(segmentNames) > 0 {
		pathMathcherConfig.SegmentNames = segmentNames
	}

	pathMathcherConfigStruct, _ := util.MessageToStruct(pathMathcherConfig)
	pathMatcherFilter := &hcm.HttpFilter{
		Name:       ut.PathMatcher,
		ConfigType: &hcm.HttpFilter_Config{pathMathcherConfigStruct},
	}
	return pathMatcherFilter
}

func (m *ConfigManager) makeTranscoderFilter(endpointApi *api.Api) *hcm.HttpFilter {
	for _, sourceFile := range m.serviceConfig.GetSourceInfo().GetSourceFiles() {
		configFile := &sm.ConfigFile{}
		ptypes.UnmarshalAny(sourceFile, configFile)
		m.Infof("got proto descriptor: %v", string(configFile.GetFileContents()))

		if configFile.GetFileType() == sm.ConfigFile_FILE_DESCRIPTOR_SET_PROTO {
			configContent := configFile.GetFileContents()
			transcodeConfig := &tc.GrpcJsonTranscoder{
				DescriptorSet: &tc.GrpcJsonTranscoder_ProtoDescriptorBin{
					ProtoDescriptorBin: configContent,
				},
				Services:               []string{endpointApi.Name},
				IgnoredQueryParameters: []string{"api_key", "key"},
			}
			transcodeConfigStruct, _ := util.MessageToStruct(transcodeConfig)
			transcodeFilter := &hcm.HttpFilter{
				Name:       util.GRPCJSONTranscoder,
				ConfigType: &hcm.HttpFilter_Config{transcodeConfigStruct},
			}
			return transcodeFilter
		}
	}
	return nil
}

func (m *ConfigManager) makeJwtAuthnFilter(endpointApi *api.Api, backendProtocol ut.BackendProtocol) *hcm.HttpFilter {
	auth := m.serviceConfig.GetAuthentication()
	if len(auth.GetProviders()) == 0 {
		return nil
	}
	providers := make(map[string]*ac.JwtProvider)
	for _, provider := range auth.GetProviders() {
		jwk, err := fetchJwk(provider.GetJwksUri(), m.client)
		if err != nil {
			glog.Warningf("fetch jwk from issuer %s got error: %s", provider.GetIssuer(), err)
			continue
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
			continue
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

		if routeMatcher := m.makeHttpRouteMatcher(rule.GetSelector()); routeMatcher != nil {
			ruleConfig := &ac.RequirementRule{
				Match:    routeMatcher,
				Requires: requires,
			}
			rules = append(rules, ruleConfig)
		}

		s := strings.Split(rule.GetSelector(), ".")
		// For gRPC protocol, needs to add extra match rule for grpc client.
		if backendProtocol == ut.GRPC {
			rules = append(rules, &ac.RequirementRule{
				Match: &route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Path{
						Path: fmt.Sprintf("/%s/%s", endpointApi.Name, s[len(s)-1]),
					},
				},
				Requires: requires,
			})
		}
	}

	jwtAuthentication := &ac.JwtAuthentication{
		Providers: providers,
		Rules:     rules,
	}

	jas, _ := util.MessageToStruct(jwtAuthentication)
	jwtAuthnFilter := &hcm.HttpFilter{
		Name:       ut.JwtAuthn,
		ConfigType: &hcm.HttpFilter_Config{jas},
	}
	return jwtAuthnFilter
}

func (m *ConfigManager) getEndpointAllowCorsFlag() bool {
	if len(m.serviceConfig.Endpoints) == 0 {
		return false
	}
	for _, endpoint := range m.serviceConfig.Endpoints {
		if endpoint.GetName() == m.serviceName && endpoint.GetAllowCors() == true {
			return true
		}
	}
	return false
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

func (m *ConfigManager) makeServiceControlFilter(endpointApi *api.Api, backendProtocol ut.BackendProtocol) *hcm.HttpFilter {
	if m.serviceConfig.GetControl().GetEnvironment() == "" {
		return nil
	}

	service := &scpb.Service{
		ServiceName:       m.serviceName,
		ServiceConfigId:   m.curConfigID,
		ProducerProjectId: m.serviceConfig.GetProducerProjectId(),
		TokenCluster:      ut.TokenCluster,
		ServiceControlUri: &scpb.HttpUri{
			Uri:     m.serviceControlURI,
			Cluster: serviceControlClusterName,
			Timeout: &duration.Duration{Seconds: 5},
		},
		ServiceConfig: copyServiceConfigForReportMetrics(m.serviceConfig),
	}

	rulesMap := make(map[string][]*scpb.ServiceControlRule)
	if backendProtocol == ut.GRPC {
		for _, method := range endpointApi.GetMethods() {
			selector := fmt.Sprintf("%s.%s", endpointApi.GetName(), method.GetName())
			rulesMap[selector] = []*scpb.ServiceControlRule{
				&scpb.ServiceControlRule{
					Requires: &scpb.Requirement{
						ServiceName:   m.serviceName,
						OperationName: selector,
					},
					Pattern: &commonpb.Pattern{
						UriTemplate: fmt.Sprintf("/%s/%s", endpointApi.GetName(), method.GetName()),
						HttpMethod:  ut.POST,
					},
				},
			}
		}
	}

	for _, httpRule := range m.serviceConfig.GetHttp().GetRules() {
		var newPattern *commonpb.Pattern
		switch httpPattern := httpRule.GetPattern().(type) {
		case *annotations.HttpRule_Get:
			newPattern = &commonpb.Pattern{
				UriTemplate: httpPattern.Get,
				HttpMethod:  ut.GET,
			}
		case *annotations.HttpRule_Put:
			newPattern = &commonpb.Pattern{
				UriTemplate: httpPattern.Put,
				HttpMethod:  ut.PUT,
			}
		case *annotations.HttpRule_Post:
			newPattern = &commonpb.Pattern{
				UriTemplate: httpPattern.Post,
				HttpMethod:  ut.POST,
			}
		case *annotations.HttpRule_Delete:
			newPattern = &commonpb.Pattern{
				UriTemplate: httpPattern.Delete,
				HttpMethod:  ut.DELETE,
			}
		case *annotations.HttpRule_Patch:
			newPattern = &commonpb.Pattern{
				UriTemplate: httpPattern.Patch,
				HttpMethod:  ut.PATCH,
			}
		}

		rulesMap[httpRule.GetSelector()] = append(rulesMap[httpRule.GetSelector()],
			&scpb.ServiceControlRule{
				Requires: &scpb.Requirement{
					ServiceName:   m.serviceName,
					OperationName: httpRule.GetSelector(),
				},
				Pattern: newPattern,
			})
	}

	// In order to support CORS. Http method OPTIONS needs to be added to all urls except the ones
	// already with options. For these OPTIONS methods, auth should be disabled and
	// AllowWithoutApiKey should be true.
	if m.getEndpointAllowCorsFlag() {
		httpPathSet := make(map[string]bool)
		for _, httpRule := range m.httpPathMap {
			httpPathSet[httpRule.path] = true
		}

		// All options have same selector as format: CORS.suffix.
		// Appends suffix to make sure it is not used by any http rules.
		corsSelectorBase := "CORS"
		corsCount := 0
		for path := range httpPathSet {
			if _, exist := m.httpPathWithOptionsSet[path]; !exist {
				corsSelector := ""
				for {
					corsSelector = fmt.Sprintf("%s.%d", corsSelectorBase, corsCount)
					if _, exist := rulesMap[corsSelector]; !exist {
						break
					}
					corsCount++
				}
				optionsPattern := &commonpb.Pattern{
					UriTemplate: path,
					HttpMethod:  ut.OPTIONS,
				}
				rulesMap[corsSelector] = []*scpb.ServiceControlRule{
					{
						Requires: &scpb.Requirement{
							ServiceName:   m.serviceName,
							OperationName: corsSelector,
							ApiKey: &scpb.APIKeyRequirement{
								AllowWithoutApiKey: true,
							},
						},
						Pattern: optionsPattern,
					},
				}
			}
		}
	}

	for _, usageRule := range m.serviceConfig.GetUsage().GetRules() {
		scRules := rulesMap[usageRule.GetSelector()]
		for _, scRule := range scRules {
			scRule.Requires.ApiKey = &scpb.APIKeyRequirement{
				AllowWithoutApiKey: usageRule.GetAllowUnregisteredCalls(),
			}
		}
	}

	filterConfig := &scpb.FilterConfig{
		Services: []*scpb.Service{service},
	}

	if m.gcpAttributes != nil {
		filterConfig.GcpAttributes = m.gcpAttributes
	}

	// Map order is not deterministic, so sort by key here to make the filter
	// config rules order deterministic. Simply iterating map will introduce
	// flakiness to the tests.
	var keys []string
	for key := range rulesMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		for _, rule := range rulesMap[key] {
			filterConfig.Rules = append(filterConfig.Rules, rule)
		}
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

func (m *ConfigManager) makeBackendAuthFilter() *hcm.HttpFilter {
	rules := []*bapb.BackendAuthRule{}
	for _, rule := range m.serviceConfig.GetBackend().GetRules() {
		rule.GetJwtAudience()
		rules = append(rules,
			&bapb.BackendAuthRule{
				Operation:    rule.GetSelector(),
				JwtAudience:  rule.GetJwtAudience(),
				TokenCluster: ut.TokenCluster,
			})
	}
	backendAuthConfig := &bapb.FilterConfig{Rules: rules}
	backendAuthConfigStruct, _ := util.MessageToStruct(backendAuthConfig)
	backendAuthFilter := &hcm.HttpFilter{
		Name:       ut.BackendAuth,
		ConfigType: &hcm.HttpFilter_Config{backendAuthConfigStruct},
	}
	return backendAuthFilter
}

func (m *ConfigManager) makeHttpRouteMatcher(selector string) *route.RouteMatch {
	var routeMatcher route.RouteMatch
	httpRule, ok := m.httpPathMap[selector]
	if !ok {
		glog.Warningf("no corresponding http path found for selector %s", selector)
		return nil
	}

	re := regexp.MustCompile(`{[^{}]+}`)

	// Replacing query parameters inside "{}" by regex "[^\/]+", which means
	// any character except `/`, also adds `$` to match to the end of the string.
	if re.MatchString(httpRule.path) {
		routeMatcher = route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Regex{
				Regex: re.ReplaceAllString(httpRule.path, `[^\/]+`) + `$`,
			},
		}
	} else {
		// Match with HttpHeader method. Some methods may have same path.
		routeMatcher = route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Path{
				Path: httpRule.path,
			},
		}
	}
	routeMatcher.Headers = []*route.HeaderMatcher{
		{
			Name: ":method",
			HeaderMatchSpecifier: &route.HeaderMatcher_ExactMatch{
				ExactMatch: httpRule.method,
			},
		},
	}
	return &routeMatcher
}

// Implements the ID method for HashNode interface.
func (m *ConfigManager) ID(node *core.Node) string {
	return node.GetId()
}

// Implements the Infof method for Log interface.
func (m *ConfigManager) Infof(format string, args ...interface{}) {
	outputString, _ := json.MarshalIndent(args, "", "   ")
	glog.Infof(format, string(outputString))
}

// Implements the Errorf method for Log interface.
func (m *ConfigManager) Errorf(format string, args ...interface{}) { glog.Errorf(format, args...) }

func (m *ConfigManager) Cache() cache.Cache { return m.cache }

// TODO(jcwang) cleanup here. This function is redundant.
func (m *ConfigManager) fetchRollouts() (*sm.ListServiceRolloutsResponse, error) {
	token, _, err := fetchAccessToken()
	if err != nil {
		return nil, fmt.Errorf("fail to get access token")
	}

	return callServiceManagementRollouts(fetchRolloutsURL(m.serviceName), token, m.client)
}

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

var callServiceManagementRollouts = func(path, token string, client *http.Client) (*sm.ListServiceRolloutsResponse, error) {
	var err error
	var resp *http.Response
	if resp, err = callWithAccessToken(path, token, client); err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	unmarshaler := &jsonpb.Unmarshaler{}
	var rolloutsResponse sm.ListServiceRolloutsResponse
	if err = unmarshaler.Unmarshal(resp.Body, &rolloutsResponse); err != nil {
		return nil, fmt.Errorf("fail to unmarshal ListServiceRolloutsResponse: %s", err)
	}
	return &rolloutsResponse, nil
}

var callServiceManagement = func(path, token string, client *http.Client) (*conf.Service, error) {
	var err error
	var resp *http.Response
	if resp, err = callWithAccessToken(path, token, client); err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	resolver := funcResolver(func(url string) (proto.Message, error) {
		switch url {
		case "type.googleapis.com/google.api.servicemanagement.v1.ConfigFile":
			return new(sm.ConfigFile), nil
		case "type.googleapis.com/google.api.HttpRule":
			return new(annotations.HttpRule), nil
		case "type.googleapis.com/google.protobuf.BoolValue":
			return new(types.BoolValue), nil
		default:
			return nil, fmt.Errorf("unexpected protobuf.Any with url: %s", url)
		}
	})
	unmarshaler := &jsonpb.Unmarshaler{
		AnyResolver: resolver,
	}
	var serviceConfig conf.Service
	if err = unmarshaler.Unmarshal(resp.Body, &serviceConfig); err != nil {
		return nil, fmt.Errorf("fail to unmarshal serviceConfig: %s", err)
	}
	return &serviceConfig, nil
}

var callWithAccessToken = func(path, token string, client *http.Client) (*http.Response, error) {
	req, _ := http.NewRequest("GET", path, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("http call to %s returns not 200 OK: %v", path, resp.Status)
	}
	return resp, nil
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
