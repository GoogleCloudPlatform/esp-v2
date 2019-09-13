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

package testdata

import (
	"io/ioutil"

	"cloudesf.googlesource.com/gcpproxy/tests/env/platform"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	anypb "github.com/golang/protobuf/ptypes/any"
	scpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

var (
	ConfigMap = map[string]*scpb.Service{
		"echo":                  FakeEchoConfig,
		"echoForDynamicRouting": FakeEchoConfigForDynamicRouting,
		"bookstore":             FakeBookstoreConfig,
		"grpc-interop":          FakeGRPCInteropConfig,
		"grpc-echo":             FakeGRPCEchoConfig,
	}
)

func generateSourceInfo(addr string) *scpb.SourceInfo {
	dat, err := ioutil.ReadFile(addr)
	if err != nil {
		glog.Errorf("error marshalAny for proto descriptor, %s", err)
		return nil
	}
	sourceFile := &smpb.ConfigFile{
		FilePath:     "api_descriptor.pb",
		FileContents: dat,
		FileType:     smpb.ConfigFile_FILE_DESCRIPTOR_SET_PROTO,
	}

	content, err := ptypes.MarshalAny(sourceFile)
	if err != nil {
		glog.Errorf("error marshalAny for proto descriptor")
		return nil
	}
	return &scpb.SourceInfo{
		SourceFiles: []*anypb.Any{content},
	}
}

func SetupSourceInfo() {
	FakeGRPCEchoConfig.SourceInfo = generateSourceInfo(platform.GetFilePath(platform.FakeGRPCEchoConfig))
	FakeGRPCInteropConfig.SourceInfo = generateSourceInfo(platform.GetFilePath(platform.FakeGRPCInteropConfig))
	FakeBookstoreConfig.SourceInfo = generateSourceInfo(platform.GetFilePath(platform.FakeBookstoreConfig))
}

func SetFakeControlEnvironment(cfg *scpb.Service, url string) {
	cfg.Control = &scpb.Control{
		Environment: url,
	}
}

func AppendLogMetrics(cfg *scpb.Service) {
	txt, err := ioutil.ReadFile("../env/testdata/logs_metrics.pb.txt")
	if err != nil {
		glog.Errorf("error reading logs_metrics.pb.txt, %s", err)
	}

	lm := &scpb.Service{}
	if err = proto.UnmarshalText(string(txt), lm); err != nil {
		glog.Errorf("failed to parse the text from logs_metrics.pb.txt, %s", err)
	}
	proto.Merge(cfg, lm)
}
