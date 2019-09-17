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
	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/grpc-echo/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	"cloudesf.googlesource.com/gcpproxy/tests/utils"

	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
)

func TestGRPCStreaming(t *testing.T) {
	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--backend_protocol=grpc", "--rollout_strategy=fixed"}

	s := env.NewTestEnv(comp.TestGRPCStreaming, "grpc-echo")
	defer s.TearDown()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testPlans := `
plans {
  echo_stream {
    call_config {
      api_key: "this-is-an-api-key"
    }
    request {
      text: "Hello, world!"
    }
    count: 10
  }
}`
	wantResult := `
results {
  echo_stream {
    count: 10
  }
}
`

	result, err := client.RunGRPCEchoTest(testPlans, s.Ports().ListenerPort)
	if err != nil {
		t.Errorf("Error during running test: %v", err)
	}
	if !strings.Contains(result, wantResult) {
		t.Errorf("The results are different,\nreceived:\n%s,\nwanted:\n%s", result, wantResult)
	}

	wantScRequests := []interface{}{
		&utils.ExpectedCheck{
			Version:         utils.APIProxyVersion,
			ServiceName:     "grpc-echo.endpoints.cloudesf-testing.cloud.goog",
			ServiceConfigID: "test-config-id",
			ConsumerID:      "api_key:this-is-an-api-key",
			OperationName:   "test.grpc.Test.EchoStream",
			CallerIp:        "127.0.0.1",
		},
		&utils.ExpectedReport{
			Version:           utils.APIProxyVersion,
			ServiceName:       "grpc-echo.endpoints.cloudesf-testing.cloud.goog",
			ServiceConfigID:   "test-config-id",
			URL:               "/test.grpc.Test/EchoStream",
			ApiKey:            "this-is-an-api-key",
			ApiMethod:         "test.grpc.Test.EchoStream",
			ProducerProjectID: "producer-project",
			ConsumerProjectID: "123456",
			FrontendProtocol:  "grpc",
			HttpMethod:        "POST",
			LogMessage:        "test.grpc.Test.EchoStream is called",
			StatusCode:        "0",
			RequestSize:       545,
			ResponseSize:      473,
			RequestBytes:      545,
			ResponseBytes:     473,
			ResponseCode:      200,
			Platform:          util.GCE,
			Location:          "test-zone",
		},
	}

	scRequests, err := s.ServiceControlServer.GetRequests(len(wantScRequests))
	if err != nil {
		t.Fatalf("GetRequests returns error: %v", err)
	}
	utils.CheckScRequest(t, scRequests, wantScRequests, "")
}
