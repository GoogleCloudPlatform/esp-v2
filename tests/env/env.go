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
	MockMetadata          bool
	MockServiceManagement bool
	MockServiceControl    bool
	MockJwtProviders      []string
	Ports                 *components.Ports

	envoy                *components.Envoy
	configMgr            *components.ConfigManagerServer
	echoBackend          *components.EchoHTTPServer
	bookstoreServer      *components.BookstoreGrpcServer
	ServiceControlServer *components.MockServiceCtrl
}

// SetUp setups Envoy, ConfigManager, and Backend server for test.
func (e *TestEnv) Setup(name uint16, backendService string, confArgs []string) error {
	e.Ports = components.NewPorts(name)
	if e.MockServiceManagement {
		fakeServiceConfig, ok := testdata.ConfigMap[backendService]
		if !ok {
			return fmt.Errorf("not supported backend")
		}
		if len(e.MockJwtProviders) > 0 {
			testdata.InitMockJwtProviders()
			// Add Mock Jwt Providers to the fake ServiceConfig.
			for _, id := range e.MockJwtProviders {
				provider, ok := testdata.MockJwtProviderMap[id]
				if !ok {
					return fmt.Errorf("not supported jwt provider id")
				}
				auth := fakeServiceConfig.GetAuthentication()
				auth.Providers = append(auth.Providers, provider)
			}
		}

		if e.MockServiceControl {
			e.ServiceControlServer = components.NewMockServiceCtrl(fakeServiceConfig.GetName())
			testdata.SetFakeControlEnvironment(fakeServiceConfig, e.ServiceControlServer.GetURL())
			testdata.AppendLogMetrics(fakeServiceConfig)
		}

		marshaler := &jsonpb.Marshaler{}
		jsonStr, err := marshaler.MarshalToString(fakeServiceConfig)
		if err != nil {
			return fmt.Errorf("fail to unmarshal fakeServiceConfig: %v", err)
		}

		confArgs = append(confArgs, "--service_management_url="+components.NewMockServiceMrg(jsonStr).GetURL())
	}

	if e.MockMetadata {
		confArgs = append(confArgs, "--metadata_url="+components.NewMockMetadata().GetURL())
	}

	confArgs = append(confArgs, fmt.Sprintf("--cluster_port=%v", e.Ports.BackendServerPort))
	confArgs = append(confArgs, fmt.Sprintf("--listener_port=%v", e.Ports.ListenerPort))
	confArgs = append(confArgs, fmt.Sprintf("--discovery_port=%v", e.Ports.DiscoveryPort))
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
	envoyConfPath := "/tmp/apiproxy-testdata-bootstrap.yaml"
	e.envoy, err = components.NewEnvoy((*debugComponents == "all" || *debugComponents == "envoy"), envoyConfPath, e.Ports)
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
		e.echoBackend, err = components.NewEchoHTTPServer(e.Ports.BackendServerPort)
		if err != nil {
			return err
		}
		errCh := e.echoBackend.Start()
		if err = <-errCh; err != nil {
			return err
		}

	case "bookstore":
		e.bookstoreServer, err = components.NewBookstoreGrpcServer(e.Ports.BackendServerPort)
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
	glog.Infof("start tearing down...")
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
	glog.Infof("finish tearing down...")
}
