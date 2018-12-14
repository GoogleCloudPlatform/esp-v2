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
	"os/exec"

	"cloudesf.googlesource.com/gcpproxy/tests/env/components"
	"cloudesf.googlesource.com/gcpproxy/tests/env/testdata"
	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
)

var (
	debugComponents = flag.String("debug_components", "", `display debug logs for components, can be "all", "envoy", "configmanager"`)
)

func init() {
	flag.Parse()
}

type TestEnv struct {
	serviceName string
	configId    string

	envoy           *components.Envoy
	configMgr       *components.ConfigManagerServer
	echoBackend     *components.EchoHTTPServer
	bookstoreServer *components.BookstoreGrpcServer

	cmd                   *exec.Cmd
	mockMetadata          bool
	mockServiceManagement bool
	mockServiceControl    bool
	mockJwtProviders      []string
}

func NewTestEnv(mockMetadata, mockServiceManagement, mockServiceControl bool, mockJwtProviders []string) *TestEnv {
	return &TestEnv{
		mockMetadata:          mockMetadata,
		mockServiceManagement: mockServiceManagement,
		mockServiceControl:    mockServiceControl,
		mockJwtProviders:      mockJwtProviders,
	}
}

// SetUp setups Envoy, ConfigManager, and Backend server for test.
func (e *TestEnv) Setup(backendService string, confArgs []string) error {
	if e.mockServiceManagement {
		fakeServiceConfig, ok := testdata.ConfigMap[backendService]
		if !ok {
			return fmt.Errorf("not supported backend")
		}
		if len(e.mockJwtProviders) > 0 {
			testdata.InitMockJwtProviders()
			// Add Mock Jwt Providers to the fake ServiceConfig.
			for _, id := range e.mockJwtProviders {
				provider, ok := testdata.MockJwtProviderMap[id]
				if !ok {
					return fmt.Errorf("not supported jwt provider id")
				}
				auth := fakeServiceConfig.GetAuthentication()
				auth.Providers = append(auth.Providers, provider)
			}
		}

		marshaler := &jsonpb.Marshaler{}
		jsonStr, err := marshaler.MarshalToString(fakeServiceConfig)
		if err != nil {
			return fmt.Errorf("fail to unmarshal fakeServiceConfig: %v", err)
		}

		confArgs = append(confArgs, "--service_management_url="+components.NewMockServiceMrg(jsonStr).GetURL())
	}

	if e.mockMetadata {
		confArgs = append(confArgs, "--metadata_url="+components.NewMockMetadata().GetURL())
	}

	var err error
	// Starts XDS.
	e.configMgr, err = components.NewConfigManagerServer((*debugComponents == "all" || *debugComponents == "configmanager"), confArgs)
	if err != nil {
		return err
	}
	err = e.configMgr.Start()
	if err != nil {
		return err
	}

	// Starts envoy.
	envoyConfPath := "../../tests/env/testdata/bootstrap.yaml"
	e.envoy, err = components.NewEnvoy((*debugComponents == "all" || *debugComponents == "envoy"), envoyConfPath)
	if err != nil {
		glog.Errorf("unable to create Envoy %v", err)
		return err
	}

	err = e.envoy.Start()
	if err != nil {
		return err
	}

	switch backendService {
	case "echo":
		// Starts Echo HTTP1 Server
		e.echoBackend, err = components.NewEchoHTTPServer()
		if err != nil {
			return err
		}
		errCh := e.echoBackend.Start()
		if err = <-errCh; err != nil {
			return err
		}

	case "bookstore":
		e.bookstoreServer, err = components.NewBookstoreGrpcServer()
		if err != nil {
			return err
		}
		errCh := e.bookstoreServer.Start()
		if err = <-errCh; err != nil {
			return err
		}
	default:
		return fmt.Errorf("please specific the correct backend service name")
	}
	return nil
}

// TearDown shutdown the servers.
func (e *TestEnv) TearDown() {
	if err := e.configMgr.Stop(); err != nil {
		glog.Errorf("error stopping config manager: %v", err)
	}

	if err := e.envoy.Stop(); err != nil {
		glog.Errorf("error stopping envoy: %v", err)
	}

	if e.echoBackend != nil {
		if err := e.echoBackend.Stop(); err != nil {
			glog.Errorf("error stopping Echo Server: %v", err)
		}
	}
	if e.bookstoreServer != nil {
		if err := e.bookstoreServer.Stop(); err != nil {
			glog.Errorf("error stopping Bookstore Server: %v", err)
		}
	}
}
