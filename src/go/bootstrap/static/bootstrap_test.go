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
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configmanager/flags"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/golang/protobuf/jsonpb"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"

	bootstrappb "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
)

var (
	FakeConfigID = "2019-12-16r0"
)

func TestServiceToBootstrapConfig(t *testing.T) {
	testData := []struct {
		desc              string
		opt_mod           func(opt *options.ConfigGeneratorOptions)
		serviceConfigPath string
		envoyConfigPath   string
		want              *bootstrappb.Admin
	}{
		{
			desc: "envoy config with service control, no tracing",
			opt_mod: func(opt *options.ConfigGeneratorOptions) {
				opt.AdminPort = 0
				opt.BackendAddress = "http://127.0.0.1:8082"
				opt.DisableTracing = true
			},
			serviceConfigPath: platform.GetFilePath(platform.ScServiceConfig),
			envoyConfigPath:   platform.GetFilePath(platform.ScEnvoyConfig),
		},
		{
			desc: "envoy config for auth",
			opt_mod: func(opt *options.ConfigGeneratorOptions) {
				opt.AdminPort = 0
				opt.BackendAddress = "http://127.0.0.1:8082"
				opt.DisableTracing = true
				opt.SkipServiceControlFilter = true
			},
			serviceConfigPath: platform.GetFilePath(platform.AuthServiceConfig),
			envoyConfigPath:   platform.GetFilePath(platform.AuthEnvoyConfig),
		},
		{
			desc: "envoy config with dynamic routing",
			opt_mod: func(opt *options.ConfigGeneratorOptions) {
				opt.AdminPort = 0
				opt.BackendAddress = "http://127.0.0.1:8082"
				opt.DisableTracing = true
				opt.SkipServiceControlFilter = true
			},
			serviceConfigPath: platform.GetFilePath(platform.DrServiceConfig),
			envoyConfigPath:   platform.GetFilePath(platform.DrEnvoyConfig),
		},
		{
			desc: "envoy config for route match",
			opt_mod: func(opt *options.ConfigGeneratorOptions) {
				opt.AdminPort = 0
				opt.BackendAddress = "http://127.0.0.1:8082"
				opt.DisableTracing = true
				opt.SkipServiceControlFilter = true
			},
			serviceConfigPath: platform.GetFilePath(platform.RmServiceConfig),
			envoyConfigPath:   platform.GetFilePath(platform.RmEnvoyConfig),
		},
		{
			desc: "grpc dynamic routing",
			opt_mod: func(opt *options.ConfigGeneratorOptions) {
				opt.AdminPort = 0
				opt.BackendAddress = "grpc://127.0.0.1:8082"
				opt.DisableTracing = true
			},
			serviceConfigPath: platform.GetFilePath(platform.GrpcEchoServiceConfig),
			envoyConfigPath:   platform.GetFilePath(platform.GrpcEchoEnvoyConfig),
		},
		{
			desc: "envoy config for sidecar with x-google-backend",
			opt_mod: func(opt *options.ConfigGeneratorOptions) {
				opt.AdminPort = 0
				opt.BackendAddress = "http://127.0.0.1:8082"
				opt.DisableTracing = true
				opt.SkipServiceControlFilter = true
			},
			serviceConfigPath: platform.GetFilePath(platform.SbServiceConfig),
			envoyConfigPath:   platform.GetFilePath(platform.SbEnvoyConfig),
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			configBytes, err := ioutil.ReadFile(tc.serviceConfigPath)
			if err != nil {
				t.Fatalf("ReadFile failed, got %v", err)
			}

			unmarshaler := &jsonpb.Unmarshaler{
				AnyResolver:        util.Resolver,
				AllowUnknownFields: false,
			}

			var s confpb.Service
			if err := unmarshaler.Unmarshal(bytes.NewBuffer(configBytes), &s); err != nil {
				t.Fatalf("Unmarshal() returned error %v, want nil", err)
			}

			opts := flags.EnvoyConfigOptionsFromFlags()
			tc.opt_mod(&opts)

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

			gotString, err := bootstrapToJson(gotBootstrap)
			if err != nil {
				t.Fatal(err)
			}
			wantString, err := bootstrapToJson(&expectedBootstrap)
			if err != nil {
				t.Fatal(err)
			}
			if err := util.JsonEqual(wantString, gotString); err != nil {
				t.Fatalf("got err: %v", err)
			}
		})
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
	var jsonObject map[string]interface{}
	err = json.Unmarshal([]byte(gotString), &jsonObject)
	if err != nil {
		return "", err
	}
	outputString, err := json.Marshal(jsonObject)
	return string(outputString), err
}
