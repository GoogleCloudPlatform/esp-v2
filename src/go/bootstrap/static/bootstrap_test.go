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

package static

import (
	"bytes"
	"flag"
	"io/ioutil"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configmanager/flags"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"

	bootstrappb "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

var (
	FakeConfigID = "2019-12-16r0"
)

func TestServiceToBootstrapConfig(t *testing.T) {
	testData := []struct {
		desc              string
		flags             map[string]string
		serviceConfigPath string
		envoyConfigPath   string
		want              *bootstrappb.Admin
	}{
		{
			desc: "envoy config with service control, no tracing",
			flags: map[string]string{
				"backend_protocol": "http1",
				"disable_tracing":  "true",
			},
			serviceConfigPath: platform.GetFilePath(platform.ScServiceConfig),
			envoyConfigPath:   platform.GetFilePath(platform.ScEnvoyConfig),
		},
		{
			desc: "envoy config with dynamic routing",
			flags: map[string]string{
				"backend_protocol":       "http2",
				"disable_tracing":        "true",
				"enable_backend_routing": "true",
			},
			serviceConfigPath: platform.GetFilePath(platform.DrServiceConfig),
			envoyConfigPath:   platform.GetFilePath(platform.DrEnvoyConfig),
		},
	}

	for _, tc := range testData {
		for key, value := range tc.flags {
			flag.Set(key, value)
		}

		configBytes, err := ioutil.ReadFile(tc.serviceConfigPath)
		if err != nil {
			t.Fatalf("ReadFile failed, got %v", err)
		}
		unmarshaler := &jsonpb.Unmarshaler{
			AnyResolver:        util.Resolver,
			AllowUnknownFields: true,
		}

		var s confpb.Service
		if err := unmarshaler.Unmarshal(bytes.NewBuffer(configBytes), &s); err != nil {
			t.Fatalf("Unmarshal() returned error %v, want nil", err)
		}

		opts := flags.EnvoyConfigOptionsFromFlags()

		// Function under test
		gotBootstrap, err := ServiceToBootstrapConfig(&s, FakeConfigID, opts)
		if err != nil {
			t.Fatal(err)
		}

		envoyConfig, err := ioutil.ReadFile(tc.envoyConfigPath)
		if err != nil {
			t.Fatalf("ReadFile failed, got %v", err)
		}

		var expectedBootstrap bootstrappb.Bootstrap
		if err := unmarshaler.Unmarshal(bytes.NewBuffer(envoyConfig), &expectedBootstrap); err != nil {
			t.Fatalf("Unmarshal() returned error %v, want nil", err)
		}

		if !proto.Equal(gotBootstrap, &expectedBootstrap) {
			gotString, err := bootstrapToJson(gotBootstrap)
			if err != nil {
				t.Fatal(err)
			}
			wantString, err := bootstrapToJson(&expectedBootstrap)
			if err != nil {
				t.Fatal(err)
			}
			t.Errorf("\ngot : %v, \nwant: %v", gotString, wantString)
		}
	}
}

func bootstrapToJson(protoMsg *bootstrappb.Bootstrap) (string, error) {
	// Marshal both protos back to json-strings to pretty print them
	marshaler := &jsonpb.Marshaler{
		AnyResolver: util.Resolver,
	}
	gotString, err := marshaler.MarshalToString(protoMsg)
	if err != nil {
		return "", err
	}
	return gotString, nil
}
