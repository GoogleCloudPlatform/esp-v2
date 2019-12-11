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

package grpc_streaming_test

import (
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/grpc_echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	scpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
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
			Version:         utils.ESPv2Version(),
			ServiceName:     "grpc-echo.endpoints.cloudesf-testing.cloud.goog",
			ServiceConfigID: "test-config-id",
			ConsumerID:      "api_key:this-is-an-api-key",
			OperationName:   "test.grpc.Test.EchoStream",
			CallerIp:        platform.GetLoopbackAddress(),
		},
		&utils.ExpectedReport{
			Version:               utils.ESPv2Version(),
			ServiceName:           "grpc-echo.endpoints.cloudesf-testing.cloud.goog",
			ServiceConfigID:       "test-config-id",
			URL:                   "/test.grpc.Test/EchoStream",
			ApiKey:                "this-is-an-api-key",
			ApiMethod:             "test.grpc.Test.EchoStream",
			ProducerProjectID:     "producer-project",
			ConsumerProjectID:     "123456",
			FrontendProtocol:      "grpc",
			HttpMethod:            "POST",
			LogMessage:            "test.grpc.Test.EchoStream is called",
			StatusCode:            "0",
			ConsumerStreamReqCnt:  10,
			ConsumerStreamRespCnt: 10,
			ProducerStreamReqCnt:  10,
			ProducerStreamRespCnt: 10,
			ResponseCode:          200,
			Platform:              util.GCE,
			Location:              "test-zone",
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

func TestGRPCLongStreaming(t *testing.T) {
	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--backend_protocol=grpc", "--rollout_strategy=fixed", "--min_stream_report_interval_ms=500"}
	streamingBytesMetrics := []string{
		"serviceruntime.googleapis.com/api/producer/request_bytes",
		"serviceruntime.googleapis.com/api/consumer/request_bytes",
		"serviceruntime.googleapis.com/api/consumer/response_bytes",
		"serviceruntime.googleapis.com/api/producer/response_bytes",
	}
	finalMetrics := []string{
		"serviceruntime.googleapis.com/api/consumer/streaming_durations",
		"serviceruntime.googleapis.com/api/producer/streaming_durations",
		"serviceruntime.googleapis.com/api/producer/streaming_request_message_counts",
		"serviceruntime.googleapis.com/api/consumer/streaming_request_message_counts",
		"serviceruntime.googleapis.com/api/producer/streaming_response_message_counts",
		"serviceruntime.googleapis.com/api/consumer/streaming_response_message_counts",
	}
	s := env.NewTestEnv(comp.TestGRPCLongStreaming, "grpc-echo")
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

	testPlans := `
plans {
  echo_stream {
    call_config {
      api_key: "this-is-an-api-key"
    }
    request {
      text: "Hello, world!"
    }
    duration_in_sec: 3
  }
}`

	_, err := client.RunGRPCEchoTest(testPlans, s.Ports().ListenerPort)
	if err != nil {
		t.Errorf("Error during running test: %v", err)
	}

	scRequests := s.ServiceControlServer.GetAllRequests()
	// Should get 1 check + n reports and n should be at least 2.
	if len(scRequests) < 3 {
		t.Errorf("The number of ScRequest should be larger than 2")
	}

	//The first service control call should be check.
	if scRequests[0].ReqType != comp.CHECK_REQUEST {
		t.Errorf("First ScRequest should be check")
	}

	// All the rest service control call should be report.
	for i := 1; i < len(scRequests); i++ {
		if scRequests[i].ReqType != comp.REPORT_REQUEST {
			t.Errorf("Except the first ScRequest, all the rest should be report")
		}
	}
	{
		firstReport, err := utils.UnmarshalReportRequest(scRequests[1].ReqBody)
		if err != nil {
			t.Errorf("Failed in unmarshal report reqeust: %v", err)
		}

		firstOperation := firstReport.Operations[0]

		// Check Operation Name
		if firstOperation.OperationName != "test.grpc.Test.EchoStream" {
			t.Errorf("Wrong operationName, expect: \"test.grpc.Test.EchoStream\", get %v", firstOperation.OperationName)
		}

		// Check labels.
		checkLabels(t, firstOperation, "/credential_id", "apikey:this-is-an-api-key")
		checkLabels(t, firstOperation, "/protocol", "grpc")

		// The requestCount should be 1 and only exist in the first report.
		wantMetricName := "serviceruntime.googleapis.com/api/producer/request_count"
		metric := findInMetricSlice(t, firstOperation.MetricValueSets, wantMetricName, true)
		if metric.MetricValues[0].GetInt64Value() != 1 {
			t.Errorf("First reporst's metric %s should be 1", wantMetricName)
		}

		// The requestCount should be 1 and only exist in the first report.
		wantMetricName = "serviceruntime.googleapis.com/api/consumer/request_count"
		metric = findInMetricSlice(t, firstOperation.MetricValueSets, wantMetricName, true)
		if metric.MetricValues[0].GetInt64Value() != 1 {
			t.Errorf("First reporst's metric %s should be 1", wantMetricName)
		}

		// Check request_bytes/response_bytes > 0.
		for _, wantMetricName := range streamingBytesMetrics {
			metric := findInMetricSlice(t, firstOperation.MetricValueSets, wantMetricName, true)
			if !(metric.MetricValues[0].GetInt64Value() > 0) {
				t.Errorf("First reporst's metric %v should be larger than 1, get %v", wantMetricName, metric.MetricValues[0].GetInt64Value())
			}
		}

		// Check no other final-report metrics.
		for _, notWantMetricName := range finalMetrics {
			findInMetricSlice(t, firstOperation.MetricValueSets, notWantMetricName, false)
		}
	}
	{
		finalReport, err := utils.UnmarshalReportRequest(scRequests[len(scRequests)-1].
			ReqBody)
		if err != nil {
			t.Errorf("Failed in unmarshal report reqeust: %v", err)
		}

		// In case intermediate reports are batched, we get the last second operation.
		if len(finalReport.Operations) < 2 {
			t.Fatalf("Should have at least 2 operations but now only have %v operations", len(finalReport.Operations))
		}
		finalOperation := finalReport.Operations[len(finalReport.Operations)-2]

		// Check Operation Name
		if finalOperation.OperationName != "test.grpc.Test.EchoStream" {
			t.Errorf("Wrong operationName, expect: \"test.grpc.Test.EchoStream\", get %v", finalOperation.OperationName)
		}

		// Check labels.
		checkLabels(t, finalOperation, "/credential_id", "apikey:this-is-an-api-key")
		checkLabels(t, finalOperation, "/protocol", "grpc")

		// The last report should have response_code and response_code_class.
		checkLabels(t, finalOperation, "/response_code", "200")
		checkLabels(t, finalOperation, "/response_code_class", "2xx")

		// The requestCount should not exist in the final report.
		wantMetricName := "serviceruntime.googleapis.com/api/producer/request_count"
		findInMetricSlice(t, finalOperation.MetricValueSets, wantMetricName, false)
		wantMetricName = "serviceruntime.googleapis.com/api/consumer/request_count"
		findInMetricSlice(t, finalOperation.MetricValueSets, wantMetricName, false)

		// Check request_bytes/response_bytes > 0
		for _, wantMetricName := range streamingBytesMetrics {
			metric := findInMetricSlice(t, finalOperation.MetricValueSets, wantMetricName, true)
			if !(metric.MetricValues[0].GetInt64Value() > 0) {
				t.Errorf("Final reporst's metric %v should be larger than 1, get %v", wantMetricName, metric.MetricValues[0].GetInt64Value())
			}
		}

		// Check all other final-report metrics.
		for _, notWantMetricName := range finalMetrics {
			findInMetricSlice(t, finalOperation.MetricValueSets, notWantMetricName, true)
		}
	}
}
