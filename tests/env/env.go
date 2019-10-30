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
	"strconv"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/api-proxy/tests/env/components"
	"github.com/GoogleCloudPlatform/api-proxy/tests/env/testdata"
	"github.com/golang/glog"

	bookserver "github.com/GoogleCloudPlatform/api-proxy/tests/endpoints/bookstore_grpc/server"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

const (
	// Additional wait time after `TestEnv.Setup`
	setupWaitTime = time.Duration(1 * time.Second)
)

var (
	debugComponents = flag.String("debug_components", "", `display debug logs for components, can be "all", "envoy", "configmanager"`)
)

// A ServiceManagementServer is a HTTP server hosting mock service configs.
type ServiceManagementServer interface {
	Start() (URL string)
}

type TestEnv struct {
	testId                      uint16
	enableDynamicRoutingBackend bool
	mockMetadata                bool
	enableScNetworkFailOpen     bool
	backendService              string
	mockMetadataOverride        map[string]string
	mockIamResps                map[string]string
	bookstoreServer             *bookserver.BookstoreServer
	grpcInteropServer           *components.GrpcInteropGrpcServer
	grpcEchoServer              *components.GrpcEchoGrpcServer
	configMgr                   *components.ConfigManagerServer
	dynamicRoutingBackend       *components.EchoHTTPServer
	echoBackend                 *components.EchoHTTPServer
	envoy                       *components.Envoy
	fakeServiceConfig           *confpb.Service
	MockMetadataServer          *components.MockMetadataServer
	MockIamServer               *components.MockIamServer
	iamServiceAccount           string
	mockServiceManagementServer ServiceManagementServer
	ports                       *components.Ports
	envoyDrainTimeInSec         int
	ServiceControlServer        *components.MockServiceCtrl
	FakeStackdriverServer       *components.FakeTraceServer
	healthRegistry              *components.HealthRegistry
	FakeJwtService              *components.FakeJwtService
}

func NewTestEnv(testId uint16, backendService string) *TestEnv {
	glog.Infof("Running test function #%v", testId)

	fakeServiceConfig := testdata.SetupServiceConfig(backendService)

	return &TestEnv{
		testId:                      testId,
		mockMetadata:                true,
		mockServiceManagementServer: components.NewMockServiceMrg(fakeServiceConfig.GetName(), fakeServiceConfig),
		backendService:              backendService,
		ports:                       components.NewPorts(testId),
		fakeServiceConfig:           fakeServiceConfig,
		ServiceControlServer:        components.NewMockServiceCtrl(fakeServiceConfig.GetName()),
		healthRegistry:              components.NewHealthRegistry(),
		FakeJwtService:              components.NewFakeJwtService(),
	}
}

// SetEnvoyDrainTimeInSec
func (e *TestEnv) SetEnvoyDrainTimeInSec(envoyDrainTimeInSec int) {
	e.envoyDrainTimeInSec = envoyDrainTimeInSec
}

// OverrideMockMetadata overrides mock metadata values given path to response map.
func (e *TestEnv) OverrideMockMetadata(newMetdaData map[string]string) {
	e.mockMetadataOverride = newMetdaData
}

// AppendHttpRules appends Service.Http.Rules.
func (e *TestEnv) SetIamResps(iamResps map[string]string) {
	e.mockIamResps = iamResps
}

func (e *TestEnv) SetIamServiceAccount(serviecAccount string) {
	e.iamServiceAccount = serviecAccount
}

// OverrideBackend overrides mock backend.
func (e *TestEnv) OverrideBackendService(newBackendService string) {
	e.backendService = newBackendService
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
func (e *TestEnv) OverrideAuthentication(authentication *confpb.Authentication) {
	e.fakeServiceConfig.Authentication = authentication
}

// OverrideSystemParameters overrides Service.SystemParameters.
func (e *TestEnv) OverrideSystemParameters(systemParameters *confpb.SystemParameters) {
	e.fakeServiceConfig.SystemParameters = systemParameters
}

// OverrideQuota overrides Service.Quota.
func (e *TestEnv) OverrideQuota(quota *confpb.Quota) {
	e.fakeServiceConfig.Quota = quota
}

// AppendApiMethods appends methods to the service config.
func (e *TestEnv) AppendApiMethods(methods []*apipb.Method) {
	e.fakeServiceConfig.Apis[0].Methods = append(e.fakeServiceConfig.Apis[0].Methods, methods...)
}

// AppendHttpRules appends Service.Http.Rules.
func (e *TestEnv) AppendHttpRules(rules []*annotationspb.HttpRule) {
	e.fakeServiceConfig.Http.Rules = append(e.fakeServiceConfig.Http.Rules, rules...)
}

// AppendBackendRules appends Service.Backend.Rules.
func (e *TestEnv) AppendBackendRules(rules []*confpb.BackendRule) {
	if e.fakeServiceConfig.Backend == nil {
		e.fakeServiceConfig.Backend = &confpb.Backend{}
	}
	e.fakeServiceConfig.Backend.Rules = append(e.fakeServiceConfig.Backend.Rules, rules...)
}

// EnableScNetworkFailOpen sets enableScNetworkFailOpen to be true.
func (e *TestEnv) EnableScNetworkFailOpen() {
	e.enableScNetworkFailOpen = true
}

// AppendUsageRules appends Service.Usage.Rules.
func (e *TestEnv) AppendUsageRules(rules []*confpb.UsageRule) {
	e.fakeServiceConfig.Usage.Rules = append(e.fakeServiceConfig.Usage.Rules, rules...)
}

// SetAllowCors Sets AllowCors in API endpoint to true.
func (e *TestEnv) SetAllowCors() {
	e.fakeServiceConfig.Endpoints[0].AllowCors = true
}

func addDynamicRoutingBackendPort(serviceConfig *confpb.Service, port uint16) error {
	for _, v := range serviceConfig.Backend.GetRules() {
		if v.PathTranslation != confpb.BackendRule_PATH_TRANSLATION_UNSPECIFIED {
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

func (e *TestEnv) SetupFakeTraceServer() {
	// Start fake stackdriver server
	e.FakeStackdriverServer = components.NewFakeStackdriver()
}

// Setup setups Envoy, ConfigManager, and Backend server for test.
func (e *TestEnv) Setup(confArgs []string) error {
	var envoyArgs []string
	var bootstrapperArgs []string
	mockJwtProviders := make(map[string]bool)

	if e.mockServiceManagementServer != nil {
		if err := addDynamicRoutingBackendPort(e.fakeServiceConfig, e.ports.DynamicRoutingBackendPort); err != nil {
			return err
		}

		for _, rule := range e.fakeServiceConfig.GetAuthentication().GetRules() {
			for _, req := range rule.GetRequirements() {
				if providerId := req.GetProviderId(); providerId != "" {
					mockJwtProviders[providerId] = true
				}
			}
		}

		glog.Infof("Requested JWT providers for this test: %v", mockJwtProviders)
		if err := e.FakeJwtService.SetupJwt(mockJwtProviders, e.ports); err != nil {
			return err
		}

		for providerId, _ := range mockJwtProviders {
			provider, ok := e.FakeJwtService.ProviderMap[providerId]
			if !ok {
				return fmt.Errorf("not supported jwt provider id: %v", providerId)
			}
			auth := e.fakeServiceConfig.GetAuthentication()
			auth.Providers = append(auth.Providers, provider.AuthProvider)
		}

		e.ServiceControlServer.Setup()
		testdata.SetFakeControlEnvironment(e.fakeServiceConfig, e.ServiceControlServer.GetURL())
		if err := testdata.AppendLogMetrics(e.fakeServiceConfig); err != nil {
			return err
		}

		confArgs = append(confArgs, "--service_management_url="+e.mockServiceManagementServer.Start())
	}

	if !e.enableScNetworkFailOpen {
		confArgs = append(confArgs, "--service_control_network_fail_open=false")
	}

	if e.mockMetadata {
		e.MockMetadataServer = components.NewMockMetadata(e.mockMetadataOverride)
		confArgs = append(confArgs, "--metadata_url="+e.MockMetadataServer.GetURL())
		bootstrapperArgs = append(bootstrapperArgs, "--metadata_url="+e.MockMetadataServer.GetURL())
	}

	if e.mockIamResps != nil {
		e.MockIamServer = components.NewIamMetadata(e.mockIamResps)
		confArgs = append(confArgs, "--iam_url="+e.MockIamServer.GetURL())
	}

	if e.iamServiceAccount != "" {
		confArgs = append(confArgs, "--iam_service_account="+e.iamServiceAccount)
	}

	if e.FakeStackdriverServer != nil {
		e.FakeStackdriverServer.StartStackdriverServer(e.ports.FakeStackdriverPort)
	}

	confArgs = append(confArgs, fmt.Sprintf("--cluster_port=%v", e.ports.BackendServerPort))
	confArgs = append(confArgs, fmt.Sprintf("--listener_port=%v", e.ports.ListenerPort))
	confArgs = append(confArgs, fmt.Sprintf("--discovery_port=%v", e.ports.DiscoveryPort))
	confArgs = append(confArgs, fmt.Sprintf("--service=%v", e.fakeServiceConfig.Name))

	// Starts XDS.
	var err error
	debugConfigMgr := *debugComponents == "all" || *debugComponents == "configmanager"
	e.configMgr, err = components.NewConfigManagerServer(debugConfigMgr, e.ports, confArgs)
	if err != nil {
		return err
	}
	if err = e.configMgr.Start(); err != nil {
		return err
	}
	e.healthRegistry.RegisterHealthChecker(e.configMgr)

	// Starts envoy.
	envoyConfPath := fmt.Sprintf("/tmp/apiproxy-testdata-bootstrap-%v.yaml", e.testId)
	if *debugComponents == "all" || *debugComponents == "envoy" {
		envoyArgs = append(envoyArgs, "--log-level", "debug")
		if e.envoyDrainTimeInSec == 0 {
			envoyArgs = append(envoyArgs, "--drain-time-s", "1")
		}
	}
	if e.envoyDrainTimeInSec != 0 {
		envoyArgs = append(envoyArgs, "--drain-time-s", strconv.Itoa(e.envoyDrainTimeInSec))
	}

	// Enable tracing if the stackdriver server was setup for this test
	shouldEnableTrace := e.FakeStackdriverServer != nil

	e.envoy, err = components.NewEnvoy(envoyArgs, bootstrapperArgs, envoyConfPath, shouldEnableTrace, e.ports, e.testId)
	if err != nil {
		glog.Errorf("unable to create Envoy %v", err)
		return err
	}
	e.healthRegistry.RegisterHealthChecker(e.envoy)

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
		e.bookstoreServer, err = bookserver.NewBookstoreServer(e.ports.BackendServerPort)
		if err != nil {
			return err
		}
		e.bookstoreServer.StartServer()
	case "grpc-interop":
		e.grpcInteropServer, err = components.NewGrpcInteropGrpcServer(e.ports.BackendServerPort)
		if err != nil {
			return err
		}
		if err := e.grpcInteropServer.StartAndWait(); err != nil {
			return err
		}
	case "grpc-echo":
		e.grpcEchoServer, err = components.NewGrpcEchoGrpcServer(e.ports.BackendServerPort)
		if err != nil {
			return err
		}
		if err := e.grpcEchoServer.StartAndWait(); err != nil {
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

	// Run health checks
	if err := e.healthRegistry.RunAllHealthChecks(); err != nil {
		return err
	}

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
		e.bookstoreServer.StopServer()
		e.bookstoreServer = nil
	}
	return retErr
}

// TearDown shutdown the servers.
func (e *TestEnv) TearDown() {
	glog.Infof("start tearing down...")

	// Run all health checks. If they fail, our test causes a server to become unhealthy...
	if err := e.healthRegistry.RunAllHealthChecks(); err != nil {
		glog.Errorf("health check failure during teardown: %v", err)
	}

	if e.FakeJwtService != nil {
		e.FakeJwtService.TearDown()
	}

	if e.configMgr != nil {
		if err := e.configMgr.StopAndWait(); err != nil {
			glog.Errorf("error stopping config manager: %v", err)
		}
	}

	if e.envoy != nil {
		if err := e.envoy.StopAndWait(); err != nil {
			glog.Errorf("error stopping envoy: %v", err)
		}
	}

	if e.echoBackend != nil {
		if err := e.echoBackend.StopAndWait(); err != nil {
			glog.Errorf("error stopping Echo Server: %v", err)
		}
	}
	if e.bookstoreServer != nil {
		e.bookstoreServer.StopServer()
		e.bookstoreServer = nil
	}
	if e.grpcInteropServer != nil {
		if err := e.grpcInteropServer.StopAndWait(); err != nil {
			glog.Errorf("error stopping GrpcInterop Server: %v", err)
		}
	}
	if e.grpcEchoServer != nil {
		if err := e.grpcEchoServer.StopAndWait(); err != nil {
			glog.Errorf("error stopping GrpcEcho Server: %v", err)
		}
	}
	if e.dynamicRoutingBackend != nil {
		if err := e.dynamicRoutingBackend.StopAndWait(); err != nil {
			glog.Errorf("error stopping Dynamic Routing Echo Server: %v", err)
		}
	}

	// Only need to stop the stackdriver server if it was ever enabled
	if e.FakeStackdriverServer != nil {
		e.FakeStackdriverServer.StopAndWait()
	}

	glog.Infof("finish tearing down...")
}
