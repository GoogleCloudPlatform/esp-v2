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

package utils

import (
	"bytes"
	"testing"

	"github.com/golang/protobuf/proto"
	sc "github.com/google/go-genproto/googleapis/api/servicecontrol/v1"
)

const expectedCheck = `
service_name: "SERVICE_NAME"
service_config_id: "SERVICE_CONFIG_ID"
operation: <
  operation_name: "ListShelves"
  consumer_id: "project:endpoints-app"
  labels: <
    key: "servicecontrol.googleapis.com/caller_ip"
    value: "127.0.0.1"
  >
  labels: <
    key: "servicecontrol.googleapis.com/service_agent"
    value: "ESP/0.3.4"
  >
  labels: <
    key: "servicecontrol.googleapis.com/user_agent"
    value: "ESP"
  >
 >
`

func TestCreateCheck(t *testing.T) {
	er := CreateCheck(&ExpectedCheck{
		Version:         "0.3.4",
		ServiceName:     "SERVICE_NAME",
		ServiceConfigID: "SERVICE_CONFIG_ID",
		ConsumerID:      "project:endpoints-app",
		OperationName:   "ListShelves",
		CallerIp:        "127.0.0.1",
	})

	expected := sc.CheckRequest{}
	if err := proto.UnmarshalText(expectedCheck, &expected); err != nil {
		t.Fatalf("proto.UnmarshalText: %v", err)
	}
	if !proto.Equal(&er, &expected) {
		t.Errorf("Got:\n===\n%v===\nExpected:\n===\n%v===\n", er.String(), expected.String())
	}
}

const expectedReport = `
        service_name: "SERVICE_NAME"
        operations: <
          operation_name: "ListShelves"
          labels: <
            key: "/error_type"
            value: "5xx"
          >
          labels: <
            key: "/protocol"
            value: "unknown"
          >
          labels: <
            key: "/response_code"
            value: "503"
          >
          labels: <
            key: "/response_code_class"
            value: "5xx"
          >
          labels: <
            key: "/status_code"
            value: "14"
          >
          labels: <
            key: "cloud.googleapis.com/location"
            value: "us-central1"
          >
          labels: <
            key: "servicecontrol.googleapis.com/platform"
            value: "unknown"
          >
          labels: <
            key: "servicecontrol.googleapis.com/service_agent"
            value: "ESP/"
          >
          labels: <
            key: "servicecontrol.googleapis.com/user_agent"
            value: "ESP"
          >
          labels: <
            key: "serviceruntime.googleapis.com/api_method"
            value: "ListShelves"
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/consumer/error_count"
            metric_values: <
              int64_value: 1
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/consumer/request_count"
            metric_values: <
              int64_value: 1
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/consumer/request_sizes"
            metric_values: <
              distribution_value: <
                count: 1
                mean: 39
                minimum: 39
                maximum: 39
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 1
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                exponential_buckets: <
                  num_finite_buckets: 8
                  growth_factor: 10
                  scale: 1
                >
              >
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/consumer/response_sizes"
            metric_values: <
              distribution_value: <
                count: 1
                mean: 208
                minimum: 208
                maximum: 208
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 1
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                exponential_buckets: <
                  num_finite_buckets: 8
                  growth_factor: 10
                  scale: 1
                >
              >
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/error_count"
            metric_values: <
              int64_value: 1
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/request_count"
            metric_values: <
              int64_value: 1
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/request_sizes"
            metric_values: <
              distribution_value: <
                count: 1
                mean: 39
                minimum: 39
                maximum: 39
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 1
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                exponential_buckets: <
                  num_finite_buckets: 8
                  growth_factor: 10
                  scale: 1
                >
              >
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/response_sizes"
            metric_values: <
              distribution_value: <
                count: 1
                mean: 208
                minimum: 208
                maximum: 208
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 1
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                exponential_buckets: <
                  num_finite_buckets: 8
                  growth_factor: 10
                  scale: 1
                >
              >
            >
          >
          log_entries: <
            name: "endpoints_log"
            severity: ERROR
            struct_payload: <
              fields: <
                key: "api_method"
                value: <
                  string_value: "ListShelves"
                >
              >
              fields: <
                key: "http_method"
                value: <
                  string_value: "GET"
                >
              >
              fields: <
                key: "http_response_code"
                value: <
                  number_value: 503
                >
              >
              fields: <
                key: "location"
                value: <
                  string_value: "us-central1"
                >
              >
              fields: <
                key: "log_message"
                value: <
                  string_value: "Method: ListShelves"
                >
              >
              fields: <
                key: "producer_project_id"
                value: <
                  string_value: "endpoints-test"
                >
              >
              fields: <
                key: "request_size_in_bytes"
                value: <
                  number_value: 39
                >
              >
              fields: <
                key: "response_size_in_bytes"
                value: <
                  number_value: 208
                >
              >
              fields: <
                key: "url"
                value: <
                  string_value: "/shelves"
                >
              >
            >
          >
        >
        service_config_id: "SERVICE_CONFIG_ID"
`

func TestCreateReport(t *testing.T) {
	er := CreateReport(&ExpectedReport{
		ServiceName:       "SERVICE_NAME",
		ServiceConfigID:   "SERVICE_CONFIG_ID",
		URL:               "/shelves",
		ApiMethod:         "ListShelves",
		ProducerProjectID: "endpoints-test",
		Location:          "us-central1",
		HttpMethod:        "GET",
		LogMessage:        "Method: ListShelves",
		RequestSize:       39,
		ResponseSize:      208,
		ResponseCode:      503,
		ErrorType:         "5xx",
	})

	expected := sc.ReportRequest{}
	if err := proto.UnmarshalText(expectedReport, &expected); err != nil {
		t.Fatalf("proto.UnmarshalText: %v", err)
	}
	if !proto.Equal(&er, &expected) {
		var buf bytes.Buffer
		if err := proto.MarshalText(&buf, &er); err != nil {
			t.Errorf("proto.MarsalText: %v", err)
		}
		t.Errorf("Got:\n===\n%v===\nExpected:\n===\n%v===\n", buf.String(), expectedReport)
	}
}
