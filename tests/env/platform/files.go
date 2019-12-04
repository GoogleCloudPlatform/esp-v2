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
	FakeGRPCEchoConfig
	FakeGRPCInteropConfig
	FakeBookstoreConfig

	// Other files
	HttpsCert
	HttpsKey
	LogMetrics
	Version
)

var fileMap = map[RuntimeFile]string{
	GrpcBookstore:           "../../endpoints/bookstore_grpc/grpc_server.js",
	Bootstrapper:            "../../../bin/bootstrap",
	ConfigManager:           "../../../bin/configmanager",
	Echo:                    "../../../bin/echo/server",
	Envoy:                   "../../../bin/envoy",
	GrpcEchoClient:          "../../../bin/grpc_echo_client",
	GrpcEchoServer:          "../../../bin/grpc_echo_server",
	GrpcInteropClient:       "../../../bin/interop_client",
	GrpcInteropServer:       "../../../bin/interop_server",
	GrpcInteropStressClient: "../../../bin/stress_test",
	FakeGRPCEchoConfig:      "../../endpoints/grpc_echo/proto/api_descriptor.pb",
	FakeGRPCInteropConfig:   "../../endpoints/grpc_interop/proto/api_descriptor.pb",
	FakeBookstoreConfig:     "../../endpoints/bookstore_grpc/proto/api_descriptor.pb",
	HttpsCert:               "../../env/testdata/localhost.crt",
	HttpsKey:                "../../env/testdata/localhost.key",
	LogMetrics:              "../../env/testdata/logs_metrics.pb.txt",
	Version: "../../../VERSION",
}

// Get the runtime file path for the specified file.
func GetFilePath(file RuntimeFile) string {
	return fileMap[file]
}
