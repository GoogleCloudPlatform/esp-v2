// Copyright 2019 Google Cloud Platform Proxy Authors
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

package bootstrap

import (
	"encoding/json"
	"flag"
	"testing"

	"cloudesf.googlesource.com/gcpproxy/src/go/bootstrap/testdata"
	"github.com/gogo/protobuf/jsonpb"

	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
)

func TestServiceToBoostrapConfig(t *testing.T) {
	flag.Set("backend_protocol", "HTTP1")
	gotBootstrap, err := ServiceToBoostrapConfig(testdata.FakeBookstoreConfig, testdata.FakeConfigID)
	if err != nil {
		t.Fatal(err)
	}

	marshaler := &jsonpb.Marshaler{
		AnyResolver: ut.Resolver,
	}
	gotEnvoyString, err := marshaler.MarshalToString(gotBootstrap)
	if err != nil {
		t.Fatal(err)
	}
	if gotEnvoyString = normalizeJson(gotEnvoyString); gotEnvoyString != normalizeJson(testdata.ExpectedBookstoreEnvoyConfig) {
		t.Errorf("ToEnvoyConfig got: %v,\nwanted: %v", gotEnvoyString, testdata.ExpectedBookstoreEnvoyConfig)
	}
}

func normalizeJson(input string) string {
	var jsonObject map[string]interface{}
	json.Unmarshal([]byte(input), &jsonObject)
	outputString, _ := json.Marshal(jsonObject)
	return string(outputString)
}
