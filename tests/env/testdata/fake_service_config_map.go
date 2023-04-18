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

package testdata

import (
	"fmt"
	"io/ioutil"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/golang/glog"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	scpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

var (
	configMap = map[platform.Backend]*scpb.Service{
		platform.EchoSidecar:          FakeEchoConfig,
		platform.EchoRemote:           FakeEchoConfigForDynamicRouting,
		platform.GrpcBookstoreSidecar: FakeBookstoreConfig,
		platform.GrpcBookstoreRemote:  FakeBookstoreConfigForDynamicRouting,
		platform.GrpcInteropSidecar:   FakeGrpcInteropConfig,
		platform.GrpcEchoSidecar:      FakeGrpcEchoConfig,
		platform.GrpcEchoRemote:       FakeGrpcEchoConfigForDynamicRouting,
	}
)

func generateSourceInfo(addr string) (*scpb.SourceInfo, error) {
	dat, err := ioutil.ReadFile(addr)
	if err != nil {
		return nil, fmt.Errorf("error marshalAny for proto descriptor, %s", err)
	}
	sourceFile := &smpb.ConfigFile{
		FilePath:     "api_descriptor.pb",
		FileContents: dat,
		FileType:     smpb.ConfigFile_FILE_DESCRIPTOR_SET_PROTO,
	}

	content, err := anypb.New(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("error marshalAny for proto descriptor")
	}
	return &scpb.SourceInfo{
		SourceFiles: []*anypb.Any{content},
	}, nil
}

func SetupServiceConfig(info platform.Backend) *scpb.Service {

	var err error
	foundConfig, ok := configMap[info]
	if !ok {
		glog.Errorf("Could not find service config for backend: %+v", info)
		return nil
	}

	// Clone to prevent modifying the original in-memory service config.
	serviceConfig := proto.Clone(foundConfig).(*scpb.Service)

	// Setup the proto descriptor for gRPC backends.
	switch info {
	case platform.GrpcBookstoreSidecar, platform.GrpcBookstoreRemote:
		serviceConfig.SourceInfo, err = generateSourceInfo(platform.GetFilePath(platform.FakeGrpcBookstoreDescriptor))
		break
	case platform.GrpcInteropSidecar:
		serviceConfig.SourceInfo, err = generateSourceInfo(platform.GetFilePath(platform.FakeGrpcInteropDescriptor))
		break
	case platform.GrpcEchoSidecar, platform.GrpcEchoRemote:
		serviceConfig.SourceInfo, err = generateSourceInfo(platform.GetFilePath(platform.FakeGrpcEchoDescriptor))
		break
	}

	if err != nil {
		glog.Errorf("fail to setup service config for %+v, got err: %v", info, err)
		return nil
	}

	return serviceConfig
}

func SetFakeControlEnvironment(cfg *scpb.Service, url string) {
	cfg.Control = &scpb.Control{
		Environment: url,
	}
}

func AppendLogMetrics(cfg *scpb.Service) error {
	txt, err := ioutil.ReadFile(platform.GetFilePath(platform.LogMetrics))
	if err != nil {
		return fmt.Errorf("error reading logs_metrics.pb.txt, %s", err)
	}

	lm := &scpb.Service{}
	if err = prototext.Unmarshal(txt, lm); err != nil {
		return fmt.Errorf("failed to parse the text from logs_metrics.pb.txt, %s", err)
	}
	proto.Merge(cfg, lm)

	return nil
}
