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
	"strings"
	"testing"

	"cloudesf.googlesource.com/gcpproxy/src/go/util"
	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/grpc_echo/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	"cloudesf.googlesource.com/gcpproxy/tests/utils"
	"google.golang.org/genproto/googleapis/api/annotations"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/genproto/protobuf/api"

	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
)

func TestGRPCFallback(t *testing.T) {
	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--backend_protocol=grpc", "--rollout_strategy=fixed"}

	s := env.NewTestEnv(comp.TestGRPCFallback, "bookstore")
	s.OverrideBackendService("grpc-echo")
	s.AppendApiMethods([]*api.Method{
		{
			Name: "Unspecified",
		},
	})
	s.AppendHttpRules([]*annotations.HttpRule{
		{
			Selector: "endpoints.examples.bookstore.Bookstore.Unspecified",
			Pattern: &annotations.HttpRule_Custom{
				Custom: &annotations.CustomHttpPattern{
					Path: "/**",
					Kind: "*",
				},
			},
		},
	})
	s.AppendUsageRules([]*conf.UsageRule{
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
				Version:               utils.APIProxyVersion,
				ServiceName:           "bookstore.endpoints.cloudesf-testing.cloud.goog",
				ServiceConfigID:       "test-config-id",
				URL:                   "/test.grpc.Test/Echo",
				ApiMethod:             "endpoints.examples.bookstore.Bookstore.Unspecified",
				ProducerProjectID:     "producer project",
				FrontendProtocol:      "grpc",
				HttpMethod:            "POST",
				LogMessage:            "endpoints.examples.bookstore.Bookstore.Unspecified is called",
				GrpcStreaming:         true,
				ProducerStreamRespCnt: 1,
				StatusCode:            "0",
				RequestSize:           332,
				ResponseSize:          410,
				RequestBytes:          332,
				ResponseBytes:         410,
				ResponseCode:          200,
				Platform:              util.GCE,
				Location:              "test-zone",
			},
		},
	}

	scRequests, err1 := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
	if err1 != nil {
		t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err1)
	}
	utils.CheckScRequest(t, scRequests, tc.wantScRequests, tc.desc)
}
