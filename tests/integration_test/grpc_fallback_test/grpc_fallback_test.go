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
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
)

func TestGRPCFallback(t *testing.T) {
	t.Parallel()

	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--rollout_strategy=fixed"}

	// Setup the service config for gRPC Bookstore.
	s := env.NewTestEnv(platform.TestGRPCFallback, platform.GrpcBookstoreSidecar)

	// But then spin up the gRPC Echo backend.
	s.OverrideBackendService(platform.GrpcEchoSidecar)
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

	defer s.TearDown(t)
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
		desc: "Success, no api key needed, report made",
		wantScRequests: []interface{}{
			&utils.ExpectedReport{
				Version:           utils.ESPv2Version(),
				ServiceName:       "bookstore.endpoints.cloudesf-testing.cloud.goog",
				ServiceConfigID:   "test-config-id",
				URL:               "/test.grpc.Test/Echo",
				ApiMethod:         "endpoints.examples.bookstore.Bookstore.Unspecified",
				ApiName:           "endpoints.examples.bookstore.Bookstore",
				ApiVersion:        "1.0.0",
				ApiKeyState:       "NOT CHECKED",
				ProducerProjectID: "producer project",
				FrontendProtocol:  "grpc",
				HttpMethod:        "POST",
				LogMessage:        "endpoints.examples.bookstore.Bookstore.Unspecified is called",
				StatusCode:        "0",
				ResponseCode:      200,
				HttpStatusCode:    200,
				GrpcStatusCode:    "OK",
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
