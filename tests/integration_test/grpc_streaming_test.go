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

package integration_test

import (
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/grpc_echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	scpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
)

func TestGRPCStreaming(t *testing.T) {
	t.Parallel()

	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--rollout_strategy=fixed"}

	s := env.NewTestEnv(platform.TestGRPCStreaming, platform.GrpcEchoSidecar)
	defer s.TearDown(t)
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
			Version:         utils.ESPv2Version(),
			ServiceName:     "grpc-echo.endpoints.cloudesf-testing.cloud.goog",
			ServiceConfigID: "test-config-id",
			ConsumerID:      "api_key:this-is-an-api-key",
			OperationName:   "test.grpc.Test.EchoStream",
			CallerIp:        platform.GetLoopbackAddress(),
		},
		&utils.ExpectedReport{
			Version:                      utils.ESPv2Version(),
			ServiceName:                  "grpc-echo.endpoints.cloudesf-testing.cloud.goog",
			ServiceConfigID:              "test-config-id",
			URL:                          "/test.grpc.Test/EchoStream",
			ApiKeyInOperationAndLogEntry: "this-is-an-api-key",
			ApiKeyState:                  "VERIFIED",
			ApiMethod:                    "test.grpc.Test.EchoStream",
			ApiName:                      "test.grpc.Test",
			ApiVersion:                   "v1",
			ProducerProjectID:            "producer-project",
			ConsumerProjectID:            "123456",
			FrontendProtocol:             "grpc",
			HttpMethod:                   "POST",
			LogMessage:                   "test.grpc.Test.EchoStream is called",
			StatusCode:                   "0",
			ResponseCode:                 200,
			Platform:                     util.GCE,
			Location:                     "test-zone",
		},
	}

	scRequests, err := s.ServiceControlServer.GetRequests(len(wantScRequests))
	if err != nil {
		t.Fatalf("GetRequests returns error: %v", err)
	}
	utils.CheckScRequest(t, scRequests, wantScRequests, "")
}

func findInMetricSlice(t *testing.T, metrics []*scpb.MetricValueSet, wantMetricName string, expectExist bool) *scpb.MetricValueSet {
	for _, metric := range metrics {
		if metric.MetricName == wantMetricName {
			if !expectExist {
				t.Fatalf("Final report shouldn't have metric %v", wantMetricName)
			}
			return metric
		}
	}
	if expectExist {
		t.Fatalf("Final report should have metric %v", wantMetricName)
	}
	return nil
}

func checkLabels(t *testing.T, op *scpb.Operation, wantLabelName, wantLabelValue string) {
	if getLabelValue := op.Labels[wantLabelName]; getLabelValue != wantLabelValue {
		t.Errorf("Wrong %s, expect: %s, get %s", wantLabelName, wantLabelValue, getLabelValue)
	}
}
