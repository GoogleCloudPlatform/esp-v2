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

package env

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"cloudesf.googlesource.com/gcpproxy/tests/env/components"
	"cloudesf.googlesource.com/gcpproxy/tests/env/testdata"
	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"google.golang.org/genproto/googleapis/api/annotations"

	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

const (
	// Additional wait time after `TestEnv.Setup`
	setupWaitTime = time.Duration(1 * time.Second)
)

var (
	debugComponents = flag.String("debug_components", "", `display debug logs for components, can be "all", "envoy", "configmanager"`)
)

func init() {
	flag.Parse()
}

// A ServiceManagementServer is a HTTP server hosting mock service configs.
type ServiceManagementServer interface {
	Start(serviceConfig string) (URL string)
}

type TestEnv struct {
	enableDynamicRoutingBackend bool
	mockMetadata                bool
	enableScNetworkFailOpen     bool
	backendService              string
	mockJwtProviders            []string
	mockMetadataOverride        map[string]string
	bookstoreServer             *components.BookstoreGrpcServer
	configMgr                   *components.ConfigManagerServer
	dynamicRoutingBackend       *components.EchoHTTPServer
	echoBackend                 *components.EchoHTTPServer
	envoy                       *components.Envoy
	fakeServiceConfig           *conf.Service
	MockMetadataServer          *components.MockMetadataServer
	mockServiceManagementServer ServiceManagementServer
	ports                       *components.Ports

	ServiceControlServer *components.MockServiceCtrl
}

func NewTestEnv(name uint16, backendService string, jwtProviders []string) *TestEnv {
	fakeServiceConfig := proto.Clone(testdata.ConfigMap[backendService]).(*conf.Service)
	return &TestEnv{
		mockMetadata:                true,
		mockServiceManagementServer: components.NewMockServiceMrg(),
		backendService:              backendService,
		ports:                       components.NewPorts(name),
		fakeServiceConfig:           fakeServiceConfig,
		mockJwtProviders:            jwtProviders,
		ServiceControlServer:        components.NewMockServiceCtrl(fakeServiceConfig.GetName()),
	}
}

// OverrideMockMetadata overrides mock metadata values given path to response map.
func (e *TestEnv) OverrideMockMetadata(newMetdaData map[string]string) {
	e.mockMetadataOverride = newMetdaData
}

// OverrideMockServiceManagementServer replaces mock Service Management implementation by a custom server.
// Set s nil to turn off service management.
func (e *TestEnv) OverrideMockServiceManagementServer(s ServiceManagementServer) {
	e.mockServiceManagementServer = s
}

// EnableDynamicRoutingBackend enables dynamic routing backend server.
func (e *TestEnv) EnableDynamicRoutingBackend() {
	e.enableDynamicRoutingBackend = true
}

// Ports returns test environment ports.
func (e *TestEnv) Ports() *components.Ports {
	return e.ports
}

// OverrideAuthentication overrides Service.Authentication.
func (e *TestEnv) OverrideAuthentication(authentication *conf.Authentication) {
	e.fakeServiceConfig.Authentication = authentication
}

// OverrideSystemParameters overrides Service.SystemParameters.
func (e *TestEnv) OverrideSystemParameters(systemParameters *conf.SystemParameters) {
	e.fakeServiceConfig.SystemParameters = systemParameters
}

// OverrideQuota overrides Service.Quota.
func (e *TestEnv) OverrideQuota(quota *conf.Quota) {
	e.fakeServiceConfig.Quota = quota
}

// AppendHttpRules appends Service.Http.Rules.
func (e *TestEnv) AppendHttpRules(rules []*annotations.HttpRule) {
	e.fakeServiceConfig.Http.Rules = append(e.fakeServiceConfig.Http.Rules, rules...)
}

// AppendBackendRules appends Service.Backend.Rules.
func (e *TestEnv) AppendBackendRules(rules []*conf.BackendRule) {
	if e.fakeServiceConfig.Backend == nil {
		e.fakeServiceConfig.Backend = &conf.Backend{}
	}
	e.fakeServiceConfig.Backend.Rules = append(e.fakeServiceConfig.Backend.Rules, rules...)
}

// EnableScNetworkFailOpen sets enableScNetworkFailOpen to be true.
func (e *TestEnv) EnableScNetworkFailOpen() {
	e.enableScNetworkFailOpen = true
}

// AppendUsageRules appends Service.Usage.Rules.
func (e *TestEnv) AppendUsageRules(rules []*conf.UsageRule) {
	e.fakeServiceConfig.Usage.Rules = append(e.fakeServiceConfig.Usage.Rules, rules...)
}

// SetAllowCors Sets AllowCors in API endpoint to true.
func (e *TestEnv) SetAllowCors() {
	e.fakeServiceConfig.Endpoints[0].AllowCors = true
}

func addDynamicRoutingBackendPort(serviceConfig *conf.Service, port uint16) error {
	for _, v := range serviceConfig.Backend.GetRules() {
		if v.PathTranslation != conf.BackendRule_PATH_TRANSLATION_UNSPECIFIED {
			urlPrefix := "https://localhost:"
			i := strings.Index(v.Address, urlPrefix)
			if i == -1 {
				return fmt.Errorf("failed to find port number")
			}
			portAndPathStr := v.Address[i+len(urlPrefix):]
			pathIndex := strings.Index(portAndPathStr, "/")
			if pathIndex == -1 {
				v.Address = fmt.Sprintf("https://localhost:%v", port)
			} else {
				v.Address = fmt.Sprintf("https://localhost:%v%v", port, portAndPathStr[pathIndex:])
			}
		}
	}
	return nil
}

// Setup setups Envoy, ConfigManager, and Backend server for test.
func (e *TestEnv) Setup(confArgs []string) error {
	if e.mockServiceManagementServer != nil {
		if err := addDynamicRoutingBackendPort(e.fakeServiceConfig, e.ports.DynamicRoutingBackendPort); err != nil {
			return err
		}
		if len(e.mockJwtProviders) > 0 {
			// Add Mock Jwt Providers to the fake ServiceConfig.
			for _, id := range e.mockJwtProviders {
				provider, ok := testdata.MockJwtProviderMap[id]
				if !ok {
					return fmt.Errorf("not supported jwt provider id: %v", id)
				}
				auth := e.fakeServiceConfig.GetAuthentication()
				auth.Providers = append(auth.Providers, provider)
			}
		}

		e.ServiceControlServer.Setup()
		testdata.SetFakeControlEnvironment(e.fakeServiceConfig, e.ServiceControlServer.GetURL())
		testdata.AppendLogMetrics(e.fakeServiceConfig)

		marshaler := &jsonpb.Marshaler{}
		jsonStr, err := marshaler.MarshalToString(e.fakeServiceConfig)
		if err != nil {
			return fmt.Errorf("fail to unmarshal fakeServiceConfig: %v", err)
		}

		confArgs = append(confArgs, "--service_management_url="+e.mockServiceManagementServer.Start(jsonStr))
	}

	if !e.enableScNetworkFailOpen {
		confArgs = append(confArgs, "--service_control_network_fail_open=false")
	}

	if e.mockMetadata {
		e.MockMetadataServer = components.NewMockMetadata(e.mockMetadataOverride)
		confArgs = append(confArgs, "--metadata_url="+e.MockMetadataServer.GetURL())
	}

	confArgs = append(confArgs, fmt.Sprintf("--cluster_port=%v", e.ports.BackendServerPort))
	confArgs = append(confArgs, fmt.Sprintf("--listener_port=%v", e.ports.ListenerPort))
	confArgs = append(confArgs, fmt.Sprintf("--discovery_port=%v", e.ports.DiscoveryPort))

	// Starts XDS.
	var err error
	debugConfigMgr := *debugComponents == "all" || *debugComponents == "configmanager"
	e.configMgr, err = components.NewConfigManagerServer(debugConfigMgr, confArgs)
	if err != nil {
		return err
	}

	if err = e.configMgr.Start(); err != nil {
		return err
	}

	// Starts envoy.
	envoyConfPath := "/tmp/apiproxy-testdata-bootstrap.yaml"
	debugEnvoy := *debugComponents == "all" || *debugComponents == "envoy"
	e.envoy, err = components.NewEnvoy(debugEnvoy, envoyConfPath, e.ports)
	if err != nil {
		glog.Errorf("unable to create Envoy %v", err)
		return err
	}

	if err = e.envoy.StartAndWait(); err != nil {
		return err
	}

	switch e.backendService {
	case "echo", "echoForDynamicRouting":
		e.echoBackend, err = components.NewEchoHTTPServer(e.ports.BackendServerPort, false, false)
		if err != nil {
			return err
		}
		if err := e.echoBackend.StartAndWait(); err != nil {
			return err
		}
	case "bookstore":
		e.bookstoreServer, err = components.NewBookstoreGrpcServer(e.ports.BackendServerPort)
		if err != nil {
			return err
		}
		if err := e.bookstoreServer.StartAndWait(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("please specify the correct backend service name")
	}

	if e.enableDynamicRoutingBackend {
		e.dynamicRoutingBackend, err = components.NewEchoHTTPServer(e.ports.DynamicRoutingBackendPort, true, true)
		if err != nil {
			return err
		}
		if err := e.dynamicRoutingBackend.StartAndWait(); err != nil {
			return err
		}
	}
	time.Sleep(setupWaitTime)
	return nil
}

func (e *TestEnv) StopBackendServer() error {
	var retErr error
	// Only one backend is instantiated for test.
	if e.echoBackend != nil {
		if err := e.echoBackend.StopAndWait(); err != nil {
			retErr = err
		}
		e.echoBackend = nil
	}
	if e.bookstoreServer != nil {
		if err := e.bookstoreServer.StopAndWait(); err != nil {
			retErr = err
		}
		e.bookstoreServer = nil
	}
	return retErr
}

// TearDown shutdown the servers.
func (e *TestEnv) TearDown() {
	glog.Infof("start tearing down...")
	if err := e.configMgr.StopAndWait(); err != nil {
		glog.Errorf("error stopping config manager: %v", err)
	}

	if err := e.envoy.StopAndWait(); err != nil {
		glog.Errorf("error stopping envoy: %v", err)
	}

	if e.echoBackend != nil {
		if err := e.echoBackend.StopAndWait(); err != nil {
			glog.Errorf("error stopping Echo Server: %v", err)
		}
	}
	if e.bookstoreServer != nil {
		if err := e.bookstoreServer.StopAndWait(); err != nil {
			glog.Errorf("error stopping Bookstore Server: %v", err)
		}
	}
	if e.dynamicRoutingBackend != nil {
		if err := e.dynamicRoutingBackend.StopAndWait(); err != nil {
			glog.Errorf("error stopping Dynamic Routing Echo Server: %v", err)
		}
	}
	glog.Infof("finish tearing down...")
}
