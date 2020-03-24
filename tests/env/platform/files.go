// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package platform

// Keys for all files
type RuntimeFile int

const (
	// Javascript files
	GrpcBookstore RuntimeFile = iota

	// Binaries
	Bootstrapper
	ConfigManager
	Echo
	Envoy
	GrpcEchoClient
	GrpcEchoServer
	GrpcInteropClient
	GrpcInteropServer
	GrpcInteropStressClient

	// Proto descriptors
	FakeGrpcEchoDescriptor
	FakeGrpcInteropDescriptor
	FakeGrpcBookstoreDescriptor

	// Other files
	ServerCert
	ServerKey
	ProxyCert
	ProxyKey
	LogMetrics
	Version
	FakeServiceAccount

	// Configurations
	AuthServiceConfig
	AuthEnvoyConfig
	ScServiceConfig
	ScEnvoyConfig
	DrServiceConfig
	DrEnvoyConfig
	PmServiceConfig
	PmEnvoyConfig
	GrpcEchoServiceConfig
	GrpcEchoEnvoyConfig
)

var fileMap = map[RuntimeFile]string{
	GrpcBookstore:               "../endpoints/bookstore_grpc/grpc_server.js",
	Bootstrapper:                "../../bin/bootstrap",
	ConfigManager:               "../../bin/configmanager",
	Echo:                        "../../bin/echo/server",
	Envoy:                       "../../bin/envoy",
	GrpcEchoClient:              "../../bin/grpc_echo_client",
	GrpcEchoServer:              "../../bin/grpc_echo_server",
	GrpcInteropClient:           "../../bin/interop_client",
	GrpcInteropServer:           "../../bin/interop_server",
	GrpcInteropStressClient:     "../../bin/stress_test",
	FakeGrpcEchoDescriptor:      "../endpoints/grpc_echo/proto/api_descriptor.pb",
	FakeGrpcInteropDescriptor:   "../endpoints/grpc_interop/proto/api_descriptor.pb",
	FakeGrpcBookstoreDescriptor: "../endpoints/bookstore_grpc/proto/api_descriptor.pb",
	ServerCert:                  "../env/testdata/server.crt",
	ServerKey:                   "../env/testdata/server.key",
	ProxyCert:                   "../env/testdata/proxy.crt",
	ProxyKey:                    "../env/testdata/proxy.key",
	LogMetrics:                  "../env/testdata/logs_metrics.pb.txt",
	Version:                     "../../VERSION",
	AuthServiceConfig:           "../../../../examples/auth/service_config_generated.json",
	AuthEnvoyConfig:             "../../../../examples/auth/envoy_config.json",
	ScServiceConfig:             "../../../../examples/service_control/service_config_generated.json",
	ScEnvoyConfig:               "../../../../examples/service_control/envoy_config.json",
	DrServiceConfig:             "../../../../examples/dynamic_routing/service_config_generated.json",
	DrEnvoyConfig:               "../../../../examples/dynamic_routing/envoy_config.json",
	PmServiceConfig:             "../../../../src/go/bootstrap/static/testdata/path_matcher/service_config_generated.json",
	PmEnvoyConfig:               "../../../../src/go/bootstrap/static/testdata/path_matcher/envoy_config.json",
	GrpcEchoServiceConfig:       "../../../../examples/grpc_dynamic_routing/service_config_generated.json",
	GrpcEchoEnvoyConfig:         "../../../../examples/grpc_dynamic_routing/envoy_config.json",
	FakeServiceAccount:             "./testdata/fake_service_account.json",
}

// Get the runtime file path for the specified file.
func GetFilePath(file RuntimeFile) string {
	return fileMap[file]
}
