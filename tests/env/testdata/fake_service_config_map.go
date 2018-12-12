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

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	any "github.com/golang/protobuf/ptypes/any"
	"github.com/google/go-genproto/googleapis/api/servicemanagement/v1"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

var (
	ConfigMap = map[string]*conf.Service{
		"echo": FakeEchoConfig,
	}
)

func init() {
	dat, err := ioutil.ReadFile("../endpoints/bookstore-grpc/proto/api_descriptor.pb")
	if err != nil {
		glog.Errorf("error marshalAny for proto descriptor, %s", err)
	}
	sourceFile := &servicemanagement.ConfigFile{
		FilePath:     "api_descriptor.pb",
		FileContents: dat,
		FileType:     servicemanagement.ConfigFile_FILE_DESCRIPTOR_SET_PROTO,
	}

	content, err := ptypes.MarshalAny(sourceFile)
	if err != nil {
		glog.Errorf("error marshalAny for proto descriptor")
	}
	FakeBookstoreConfig.SourceInfo = &conf.SourceInfo{
		SourceFiles: []*any.Any{content},
	}
	ConfigMap["bookstore"] = FakeBookstoreConfig
}
