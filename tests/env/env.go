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

package env

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/golang/glog"

	bookserver "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/server"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

const (
	// Additional wait time after `TestEnv.Setup`
	setupWaitTime = 1 * time.Second
	initRolloutId = "test-rollout-id"
)

var (
	debugComponents = flag.String("debug_components", "", `display debug logs for components, can be "all", "envoy", "configmanager"`)
)

type TestEnv struct {
	testId  uint16
	backend platform.Backend
	// used to enable mutual authentication for HTTPS backend
	backendMTLSCertFile             string
	mockMetadata                    bool
	enableScNetworkFailOpen         bool
	enableEchoServerRootPathHandler bool
	mockMetadataOverride            map[string]string
	mockMetadataFailures            int
	mockIamResps                    map[string]string
	mockIamFailures                 int
	bookstoreServer                 *bookserver.BookstoreServer
	grpcInteropServer               *components.GrpcInteropGrpcServer
	grpcEchoServer                  *components.GrpcEchoGrpcServer
	configMgr                       *components.ConfigManagerServer
	echoBackend                     *components.EchoHTTPServer
	envoy                           *components.Envoy
	rolloutId                       string
	fakeServiceConfig               *confpb.Service
	MockMetadataServer              *components.MockMetadataServer
	MockIamServer                   *components.MockIamServer
	backendAuthIamServiceAccount    string
	backendAuthIamDelegates         string
	serviceControlIamServiceAccount string
	serviceControlIamDelegates      string
	MockServiceManagementServer     *components.MockServiceMrg
	ports                           *components.Ports
	envoyDrainTimeInSec             int
	ServiceControlServer            *components.MockServiceCtrl
	FakeStackdriverServer           *components.FakeTraceServer
	healthRegistry                  *components.HealthRegistry
	FakeJwtService                  *components.FakeJwtService
	skipHealthChecks                bool

	// Only implemented for the RemoteEcho backend.
	useWrongBackendCert         bool
	disableHttp2ForHttpsBackend bool
}

func NewTestEnv(testId uint16, backend platform.Backend) *TestEnv {
	glog.Infof("Running test function #%v", testId)

	fakeServiceConfig := testdata.SetupServiceConfig(backend)

	return &TestEnv{
		testId:                      testId,
		backend:                     backend,
		mockMetadata:                true,
		MockServiceManagementServer: components.NewMockServiceMrg(fakeServiceConfig.GetName(), initRolloutId, fakeServiceConfig),
		ports:                       components.NewPorts(testId),
		rolloutId:                   initRolloutId,
		fakeServiceConfig:           fakeServiceConfig,
		ServiceControlServer:        components.NewMockServiceCtrl(fakeServiceConfig.GetName(), initRolloutId),
		healthRegistry:              components.NewHealthRegistry(),
		FakeJwtService:              components.NewFakeJwtService(),
	}
}

// SetEnvoyDrainTimeInSec
func (e *TestEnv) SetEnvoyDrainTimeInSec(envoyDrainTimeInSec int) {
	e.envoyDrainTimeInSec = envoyDrainTimeInSec
}

// OverrideMockMetadata overrides mock metadata values given path to response map.
func (e *TestEnv) OverrideMockMetadata(newImdsData map[string]string, imdsFailures int) {
	e.mockMetadataOverride = newImdsData
	e.mockMetadataFailures = imdsFailures
}

func (e *TestEnv) GetDynamicRoutingBackendPort() uint16 {
	return e.ports.DynamicRoutingBackendPort
}

// Dictates the responses and the number of failures mock IAM will respond with.
func (e *TestEnv) SetIamResps(iamResps map[string]string, iamFailures int) {
	e.mockIamResps = iamResps
	e.mockIamFailures = iamFailures
}

func (e *TestEnv) SetBackendAuthIamServiceAccount(serviecAccount string) {
	e.backendAuthIamServiceAccount = serviecAccount
}

func (e *TestEnv) SetBackendAuthIamDelegates(delegates string) {
	e.backendAuthIamDelegates = delegates
}

func (e *TestEnv) SetServiceControlIamServiceAccount(serviecAccount string) {
	e.serviceControlIamServiceAccount = serviecAccount
}

func (e *TestEnv) SetServiceControlIamDelegates(delegates string) {
	e.serviceControlIamDelegates = delegates
}

// OverrideBackend overrides the mock backend only.
// Warning: This will result in using the service config for the original backend,
// even though the new backend is spun up.
func (e *TestEnv) OverrideBackendService(backend platform.Backend) {
	e.backend = backend
}

// For use when dynamic routing is enabled.
// By default, it uses same cert as Envoy for HTTPS calls. When useWrongBackendCert
// is set to true, purposely fail HTTPS calls for testing.
func (e *TestEnv) UseWrongBackendCertForDR(useWrongBackendCert bool) {
	e.useWrongBackendCert = useWrongBackendCert
}

// SetBackendMTLSCert sets the backend cert file to enable mutual authentication.
func (e *TestEnv) SetBackendMTLSCert(fileName string) {
	e.backendMTLSCertFile = fileName
}

// Ports returns test environment ports.
func (e *TestEnv) Ports() *components.Ports {
	return e.ports
}

// OverrideAuthentication overrides Service.Authentication.
func (e *TestEnv) OverrideAuthentication(authentication *confpb.Authentication) {
	e.fakeServiceConfig.Authentication = authentication
}

// OverrideAuthentication overrides Service.Authentication.
func (e *TestEnv) OverrideRolloutIdAndConfigId(newRolloutId, newConfigId string) {
	e.fakeServiceConfig.Id = newConfigId
	e.rolloutId = newRolloutId
	e.MockServiceManagementServer.SetRolloutId(newRolloutId)
	e.ServiceControlServer.SetRolloutIdConfigIdInReport(newRolloutId)
}

func (e *TestEnv) ServiceConfigId() string {
	if e.fakeServiceConfig == nil {
		return ""
	}
	return e.fakeServiceConfig.Id
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

// RemoveAllBackendRules removes all Service.Backend.Rules.
// This is useful for testing
func (e *TestEnv) RemoveAllBackendRules() {
	e.fakeServiceConfig.Backend = &confpb.Backend{}
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

func (e *TestEnv) EnableEchoServerRootPathHandler() {
	e.enableEchoServerRootPathHandler = true
}

// Limit usage of this, as it causes flakes in CI.
// Only intended to be used to test if Envoy starts up correctly.
// Ideally, the test using this should have it's own retry loop.
func (e *TestEnv) SkipHealthChecks() {
	e.skipHealthChecks = true
}

// In the service config for each backend, the backend port is represented with a "-1".
// Example: Address: "https://localhost:-1/"
// During env setup, replace the -1 with the actual dynamic routing port for the test.
func addDynamicRoutingBackendPort(serviceConfig *confpb.Service, port uint16) error {
	for _, rule := range serviceConfig.Backend.GetRules() {
		if !strings.Contains(rule.Address, "-1") {
			return fmt.Errorf("backend rule address (%v) is not properly formatted", rule.Address)
		}

		rule.Address = strings.ReplaceAll(rule.Address, "-1", strconv.Itoa(int(port)))
	}
	return nil
}

func (e *TestEnv) SetupFakeTraceServer() {
	// Start fake stackdriver server
	e.FakeStackdriverServer = components.NewFakeStackdriver()
}

func (e *TestEnv) DisableHttp2ForHttpsBackend() {
	e.disableHttp2ForHttpsBackend = true
}

// Setup setups Envoy, Config Manager, and Backend server for test.
func (e *TestEnv) Setup(confArgs []string) error {
	var envoyArgs []string
	var bootstrapperArgs []string
	mockJwtProviders := make(map[string]bool)
	if e.MockServiceManagementServer != nil {
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

		for providerId := range mockJwtProviders {
			provider, ok := e.FakeJwtService.ProviderMap[providerId]
			if !ok {
				return fmt.Errorf("not supported jwt provider id: %v", providerId)
			}
			auth := e.fakeServiceConfig.GetAuthentication()
			auth.Providers = append(auth.Providers, provider.AuthProvider)
		}

		e.ServiceControlServer.Setup()
		testdata.SetFakeControlEnvironment(e.fakeServiceConfig, e.ServiceControlServer.GetURL())
		confArgs = append(confArgs, "--service_control_url="+e.ServiceControlServer.GetURL())
		if err := testdata.AppendLogMetrics(e.fakeServiceConfig); err != nil {
			return err
		}

		confArgs = append(confArgs, "--service_management_url="+e.MockServiceManagementServer.Start())
	}

	if !e.enableScNetworkFailOpen {
		confArgs = append(confArgs, "--service_control_network_fail_open=false")
	}

	if e.mockMetadata {
		e.MockMetadataServer = components.NewMockMetadata(e.mockMetadataOverride, e.mockMetadataFailures)
		confArgs = append(confArgs, "--metadata_url="+e.MockMetadataServer.GetURL())
		bootstrapperArgs = append(bootstrapperArgs, "--metadata_url="+e.MockMetadataServer.GetURL())
	}

	if e.mockIamResps != nil || e.mockIamFailures != 0 {
		e.MockIamServer = components.NewIamMetadata(e.mockIamResps, e.mockIamFailures)
		confArgs = append(confArgs, "--iam_url="+e.MockIamServer.GetURL())
	}

	if e.backendAuthIamServiceAccount != "" {
		confArgs = append(confArgs, "--backend_auth_iam_service_account="+e.backendAuthIamServiceAccount)
	}

	if e.backendAuthIamDelegates != "" {
		confArgs = append(confArgs, "--backend_auth_iam_delegates="+e.backendAuthIamDelegates)
	}

	if e.serviceControlIamServiceAccount != "" {
		confArgs = append(confArgs, "--service_control_iam_service_account="+e.serviceControlIamServiceAccount)
	}

	if e.serviceControlIamDelegates != "" {
		confArgs = append(confArgs, "--service_control_iam_delegates="+e.serviceControlIamDelegates)
	}

	if e.FakeStackdriverServer != nil {
		e.FakeStackdriverServer.StartStackdriverServer(e.ports.FakeStackdriverPort)
	}

	confArgs = append(confArgs, fmt.Sprintf("--listener_port=%v", e.ports.ListenerPort))
	confArgs = append(confArgs, fmt.Sprintf("--discovery_port=%v", e.ports.DiscoveryPort))
	confArgs = append(confArgs, fmt.Sprintf("--service=%v", e.fakeServiceConfig.Name))

	// Enable tracing if the stackdriver server was setup for this test
	shouldEnableTrace := e.FakeStackdriverServer != nil
	if !shouldEnableTrace {
		confArgs = append(confArgs, "--disable_tracing")
	}

	// Starts XDS.
	var err error
	debugConfigMgr := *debugComponents == "all" || *debugComponents == "configmanager"
	e.configMgr, err = components.NewConfigManagerServer(debugConfigMgr, e.ports, e.backend, confArgs)
	if err != nil {
		return err
	}
	if err = e.configMgr.StartAndWait(); err != nil {
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

	e.envoy, err = components.NewEnvoy(envoyArgs, bootstrapperArgs, envoyConfPath, shouldEnableTrace, e.ports, e.testId)
	if err != nil {
		glog.Errorf("unable to create Envoy %v", err)
		return err
	}
	e.healthRegistry.RegisterHealthChecker(e.envoy)

	if err = e.envoy.StartAndWait(); err != nil {
		return err
	}

	switch e.backend {
	case platform.EchoSidecar:
		e.echoBackend, err = components.NewEchoHTTPServer(e.ports.BackendServerPort /*enableHttps=*/, false /*enableRootPathHandler=*/, e.enableEchoServerRootPathHandler /*useAuthorizedBackendCert*/, false, e.backendMTLSCertFile, e.disableHttp2ForHttpsBackend)
		if err != nil {
			return err
		}
		if err := e.echoBackend.StartAndWait(); err != nil {
			return err
		}
	case platform.EchoRemote:
		e.echoBackend, err = components.NewEchoHTTPServer(e.ports.DynamicRoutingBackendPort /*enableHttps=*/, true /*enableRootPathHandler=*/, true, e.useWrongBackendCert, e.backendMTLSCertFile, e.disableHttp2ForHttpsBackend)
		if err != nil {
			return err
		}
		if err := e.echoBackend.StartAndWait(); err != nil {
			return err
		}
	case platform.GrpcBookstoreSidecar:
		e.bookstoreServer, err = bookserver.NewBookstoreServer(e.ports.BackendServerPort /*enableTLS=*/, false /*useAuthorizedBackendCert*/, false /*backendMTLSCertFile=*/, "")
		if err != nil {
			return err
		}
		e.bookstoreServer.StartServer()
	case platform.GrpcBookstoreRemote:
		e.bookstoreServer, err = bookserver.NewBookstoreServer(e.ports.DynamicRoutingBackendPort /*enableTLS=*/, true, e.useWrongBackendCert, e.backendMTLSCertFile)
		if err != nil {
			return err
		}
		e.bookstoreServer.StartServer()
	case platform.GrpcInteropSidecar:
		e.grpcInteropServer, err = components.NewGrpcInteropGrpcServer(e.ports.BackendServerPort)
		if err != nil {
			return err
		}
		if err := e.grpcInteropServer.StartAndWait(); err != nil {
			return err
		}
	case platform.GrpcEchoSidecar:
		e.grpcEchoServer, err = components.NewGrpcEchoGrpcServer(e.ports.BackendServerPort)
		if err != nil {
			return err
		}
		if err := e.grpcEchoServer.StartAndWait(); err != nil {
			return err
		}
	case platform.GrpcEchoRemote:
		e.grpcEchoServer, err = components.NewGrpcEchoGrpcServer(e.ports.DynamicRoutingBackendPort)
		if err != nil {
			return err
		}
		if err := e.grpcEchoServer.StartAndWait(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("backend (%v) is not supported", e.backend)
	}

	time.Sleep(setupWaitTime)

	// Run health checks
	if !e.skipHealthChecks {
		if err := e.healthRegistry.RunAllHealthChecks(); err != nil {
			return err
		}
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
	if !e.skipHealthChecks {
		if err := e.healthRegistry.RunAllHealthChecks(); err != nil {
			glog.Errorf("health check failure during teardown: %v", err)
		}
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

	// Only need to stop the stackdriver server if it was ever enabled
	if e.FakeStackdriverServer != nil {
		e.FakeStackdriverServer.StopAndWait()
	}

	glog.Infof("finish tearing down...")
}
