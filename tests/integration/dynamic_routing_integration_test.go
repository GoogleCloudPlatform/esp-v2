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

package integration

import (
	"fmt"
	"testing"
	"time"

	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/echo/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	"cloudesf.googlesource.com/gcpproxy/tests/utils"

	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
)

func TestDynamicRouting(t *testing.T) {
	serviceName := "test-echo"
	configId := "test-config-id"

	// TODO(jcwang): enable service control filter later
	args := []string{"--service=" + serviceName, "--version=" + configId,
		"--backend_protocol=http1", "--rollout_strategy=fixed", "--enable_backend_routing", "--skip_service_control_filter"}

	s := env.TestEnv{
		MockMetadata:          true,
		MockServiceManagement: true,
		MockServiceControl:    true,
		MockJwtProviders:      nil,
		UseHttpsBackend:       true,
	}

	if err := s.Setup(comp.TestDynamicRouting, "echo", args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()
	time.Sleep(time.Duration(3 * time.Second))

	// TODO(jcwang) match ESP backend routing tests:
	// https://github.com/cloudendpoints/esp/blob/master/src/nginx/t/backend_routing_append_path.t
	// https://github.com/cloudendpoints/esp/blob/master/src/nginx/t/backend_routing_constant_address.t
	// TODO(jcwang) add a test that is no re-routing needed
	// TODO(kyuc) test BackendAuth
	testData := []struct {
		desc     string
		path     string
		wantResp string
	}{
		{
			desc:     "Succeed, CONSTANT_ADDRESS path translation is correct",
			path:     "/pet/123/num/987",
			wantResp: `{"Path":"/dynamicrouting/getpetbyid","RawQuery":"pet_id=123&number=987"}`,
		},
		{
			desc:     "Succeed, CONSTANT_ADDRESS path translation is correct",
			path:     "/pet/31/num/565?lang=US&zone=us-west1",
			wantResp: `{"Path":"/dynamicrouting/getpetbyid","RawQuery":"lang=US&zone=us-west1&pet_id=31&number=565"}`,
		},
		{
			desc:     "Succeed, APPEND_PATH_TO_ADDRESS path translation is correct",
			path:     "/searchpet",
			wantResp: `{"Path":"/dynamicrouting/searchpet/searchpet","RawQuery":""}`,
		},
		{
			desc:     "Succeed, APPEND_PATH_TO_ADDRESS path translation is correct",
			path:     "/searchpet?timezone=PST&lang=US",
			wantResp: `{"Path":"/dynamicrouting/searchpet/searchpet","RawQuery":"timezone=PST&lang=US"}`,
		},
		{
			desc:     "Succeed, APPEND_PATH_TO_ADDRESS path translation is correct",
			path:     "/searchdog",
			wantResp: `{"Path":"/dynamicrouting/searchdogs/searchdog","RawQuery":""}`,
		},
		{
			desc:     "Succeed, APPEND_PATH_TO_ADDRESS path translation is correct",
			path:     "/searchdog?timezone=UTC",
			wantResp: `{"Path":"/dynamicrouting/searchdogs/searchdog","RawQuery":"timezone=UTC"}`,
		},
		{
			desc:     "Succeed, APPEND_PATH_TO_ADDRESS path translation is correct",
			path:     "/pets/cat/year/2018",
			wantResp: `{"Path":"/dynamicrouting/listpet/pets/cat/year/2018","RawQuery":""}`,
		},
		{
			desc:     "Succeed, APPEND_PATH_TO_ADDRESS path translation is correct",
			path:     "/pets/dog/year/2019?lang=US&zone=us-west1",
			wantResp: `{"Path":"/dynamicrouting/listpet/pets/dog/year/2019","RawQuery":"lang=US&zone=us-west1"}`,
		},
	}
	for _, tc := range testData {
		url := fmt.Sprintf("http://localhost:%v%v", s.Ports.ListenerPort, tc.path)
		gotResp, err := client.DoGet(url)
		if err != nil {
			t.Fatal(err)
		}
		gotRespStr := utils.NormalizeJson(string(gotResp))

		if gotRespStr != utils.NormalizeJson(tc.wantResp) {
			t.Errorf("response expected: %s, got: %s", tc.wantResp, gotRespStr)
		}
	}
}
