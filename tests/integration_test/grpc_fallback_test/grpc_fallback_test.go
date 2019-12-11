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

package grpc_fallback_test

import (
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/grpc_echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

func TestGRPCFallback(t *testing.T) {
	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--backend_protocol=grpc", "--rollout_strategy=fixed"}

	s := env.NewTestEnv(comp.TestGRPCFallback, "bookstore")
	s.OverrideBackendService("grpc-echo")
	s.AppendApiMethods([]*apipb.Method{
		{
			Name: "Unspecified",
		},
	})
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: "endpoints.examples.bookstore.Bookstore.Unspecified",
			Pattern: &annotationspb.HttpRule_Custom{
				Custom: &annotationspb.CustomHttpPattern{
					Path: "/**",
					Kind: "*",
				},
			},
		},
	})
	s.AppendUsageRules([]*confpb.UsageRule{
		{
			Selector:               "endpoints.examples.bookstore.Bookstore.Unspecified",
			AllowUnregisteredCalls: true,
		},
	})
	defer s.TearDown()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testPlans := `
plans {
  echo {
    request {
      text: "Hello, world!"
    }
  }
}
`
	result, err := client.RunGRPCEchoTest(testPlans, s.Ports().ListenerPort)
	wantResult := `
results {
  echo {
    text: "Hello, world!"
  }
}`
	if err != nil {
		t.Errorf("TestGRPCErrors: error during tests: %v", err)
	}
	if !strings.Contains(result, wantResult) {
		t.Errorf("TestGRPCErrors: the results are different,\nreceived:\n%s,\nwanted:\n%s", result, wantResult)
	}

	tc := struct {
		desc           string
		wantScRequests []interface{}
	}{
		desc: "succeed GET, no Jwt required, service control sends check request and report request for GET request",
		wantScRequests: []interface{}{
			&utils.ExpectedReport{
				Version:           utils.ESPv2Version(),
				ServiceName:       "bookstore.endpoints.cloudesf-testing.cloud.goog",
				ServiceConfigID:   "test-config-id",
				URL:               "/test.grpc.Test/Echo",
				ApiMethod:         "endpoints.examples.bookstore.Bookstore.Unspecified",
				ProducerProjectID: "producer project",
				FrontendProtocol:  "grpc",
				HttpMethod:        "POST",
				LogMessage:        "endpoints.examples.bookstore.Bookstore.Unspecified is called",
				RequestMsgCounts:  1,
				ResponseMsgCounts: 1,
				StatusCode:        "0",
				ResponseCode:      200,
				Platform:          util.GCE,
				Location:          "test-zone",
			},
		},
	}

	scRequests, err1 := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
	if err1 != nil {
		t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err1)
	}
	utils.CheckScRequest(t, scRequests, tc.wantScRequests, tc.desc)
}
