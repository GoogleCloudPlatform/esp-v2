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

package utils

import (
	"testing"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"

	scpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
)

const expectedCheck = `
service_name: "SERVICE_NAME"
service_config_id: "SERVICE_CONFIG_ID"
operation: <
  operation_name: "ListShelves"
  consumer_id: "project:endpoints-app"
  labels: <
    key: "servicecontrol.googleapis.com/android_cert_fingerprint"
    value: "ABCDESF"
  >
  labels: <
    key: "servicecontrol.googleapis.com/android_package_name"
    value: "com.google.cloud"
  >
  labels: <
    key: "servicecontrol.googleapis.com/ios_bundle_id"
    value: "5b40ad6af9a806305a0a56d7cb91b82a27c26909"
  >
  labels: <
    key: "servicecontrol.googleapis.com/referer"
    value: "referer"
  >
  labels: <
    key: "servicecontrol.googleapis.com/caller_ip"
    value: "127.0.0.1"
  >
  labels: <
    key: "servicecontrol.googleapis.com/service_agent"
    value: "ESPv2/0.3.4"
  >
  labels: <
    key: "servicecontrol.googleapis.com/user_agent"
    value: "ESPv2"
  >
 >
`

func TestCreateCheck(t *testing.T) {
	er := CreateCheck(&ExpectedCheck{
		Version:                "0.3.4",
		ServiceName:            "SERVICE_NAME",
		ServiceConfigID:        "SERVICE_CONFIG_ID",
		ConsumerID:             "project:endpoints-app",
		OperationName:          "ListShelves",
		CallerIp:               "127.0.0.1",
		AndroidCertFingerprint: "ABCDESF",
		AndroidPackageName:     "com.google.cloud",
		IosBundleID:            "5b40ad6af9a806305a0a56d7cb91b82a27c26909",
		Referer:                "referer",
	})

	expected := scpb.CheckRequest{}
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
          consumer_id: "api_key:api-key"
          labels: <
            key: "/credential_id"
            value: "apikey:api-key"
          >
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
            value: "ESPv2/"
          >
          labels: <
            key: "servicecontrol.googleapis.com/user_agent"
            value: "ESPv2"
          >
          labels: <
            key: "serviceruntime.googleapis.com/api_method"
            value: "ListShelves"
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/consumer/backend_latencies"
            metric_values: <
              distribution_value: <
                count: 1
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 1
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
                >
              >
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/consumer/error_count"
            metric_values: <
              int64_value: 1
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/consumer/request_bytes"
            metric_values: <
              int64_value: 200
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/consumer/request_count"
            metric_values: <
              int64_value: 1
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/consumer/request_overhead_latencies"
            metric_values: <
              distribution_value: <
                count: 1
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 1
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
                >
              >
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/consumer/request_sizes"
            metric_values: <
              distribution_value: <
                count: 1
                mean: 1000
                minimum: 1000
                maximum: 1000
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
            metric_name: "serviceruntime.googleapis.com/api/consumer/response_bytes"
            metric_values: <
              int64_value: 200
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/consumer/response_sizes"
            metric_values: <
              distribution_value: <
                count: 1
                mean: 1000
                minimum: 1000
                maximum: 1000
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
            metric_name: "serviceruntime.googleapis.com/api/consumer/streaming_durations"
            metric_values: <
              distribution_value: <
                count: 1
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 1
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
                >
              >
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/consumer/total_latencies"
            metric_values: <
              distribution_value: <
                count: 1
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 1
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
                >
              >
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/backend_latencies"
            metric_values: <
              distribution_value: <
                count: 1
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 1
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
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
            metric_name: "serviceruntime.googleapis.com/api/producer/request_bytes"
            metric_values: <
              int64_value: 200
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/request_count"
            metric_values: <
              int64_value: 1
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/request_overhead_latencies"
            metric_values: <
              distribution_value: <
                count: 1
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 1
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
                >
              >
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/request_sizes"
            metric_values: <
              distribution_value: <
                count: 1
                mean: 1000
                minimum: 1000
                maximum: 1000
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
            metric_name: "serviceruntime.googleapis.com/api/producer/response_bytes"
            metric_values: <
              int64_value: 200
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/response_sizes"
            metric_values: <
              distribution_value: <
                count: 1
                mean: 1000
                minimum: 1000
                maximum: 1000
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
            metric_name: "serviceruntime.googleapis.com/api/producer/streaming_durations"
            metric_values: <
              distribution_value: <
                count: 1
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 1
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
                >
              >
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/total_latencies"
            metric_values: <
              distribution_value: <
                count: 1
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 1
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
                >
              >
            >
          >
          log_entries: <
            name: "endpoints_log"
            severity: ERROR
            struct_payload: <
              fields: <
                key: "api_key"
                value: <
                  string_value: "api-key"
                >
              >
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
                key: "url"
                value: <
                  string_value: "/shelves"
                >
              >
              fields: <
                key: "client_ip"
                value: <
                  string_value: "127.0.0.1"
                >
              >
            >
          >
        >
        operations: <
          operation_name: "ListShelves"
          consumer_id: "api_key:api-key"
          labels: <
            key: "/credential_id"
            value: "apikey:api-key"
          >
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
            value: "ESPv2/"
          >
          labels: <
            key: "servicecontrol.googleapis.com/user_agent"
            value: "ESPv2"
          >
          labels: <
            key: "serviceruntime.googleapis.com/api_method"
            value: "ListShelves"
          >
          labels: <
            key: "serviceruntime.googleapis.com/consumer_project"
            value: "123456"
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/by_consumer/backend_latencies"
            metric_values: <
              distribution_value: <
                count: 1
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 1
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
                >
              >
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/by_consumer/error_count"
            metric_values: <
              int64_value: 1
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/by_consumer/request_count"
            metric_values: <
              int64_value: 1
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/by_consumer/request_overhead_latencies"
            metric_values: <
              distribution_value: <
                count: 1
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 1
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
                >
              >
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/by_consumer/request_sizes"
            metric_values: <
              distribution_value: <
                count: 1
                mean: 1000
                minimum: 1000
                maximum: 1000
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
            metric_name: "serviceruntime.googleapis.com/api/producer/by_consumer/response_sizes"
            metric_values: <
              distribution_value: <
                count: 1
                mean: 1000
                minimum: 1000
                maximum: 1000
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
            metric_name: "serviceruntime.googleapis.com/api/producer/by_consumer/total_latencies"
            metric_values: <
              distribution_value: <
                count: 1
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 1
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
                >
              >
            >
          >
        >
        service_config_id: "SERVICE_CONFIG_ID"
	`

func TestCreateReport(t *testing.T) {
	got := CreateReport(&ExpectedReport{
		ServiceName:       "SERVICE_NAME",
		ServiceConfigID:   "SERVICE_CONFIG_ID",
		URL:               "/shelves",
		ApiMethod:         "ListShelves",
		ApiKey:            "api-key",
		ProducerProjectID: "endpoints-test",
		ConsumerProjectID: "123456",
		Location:          "us-central1",
		HttpMethod:        "GET",
		LogMessage:        "Method: ListShelves",
		ResponseCode:      503,
		StatusCode:        "14",
		ErrorType:         "5xx",
	})

	want := scpb.ReportRequest{}
	if err := proto.UnmarshalText(expectedReport, &want); err != nil {
		t.Fatalf("proto.UnmarshalText: %v", err)
	}
	if diff := ProtoDiff(&want, &got); diff != "" {
		glog.Infof("---Want---\n%v", proto.MarshalTextString(&want))
		glog.Infof("---Got---\n%v", proto.MarshalTextString(&got))
		t.Errorf("Report diff (-want, +got):\n%s", diff)
	}
}

const expectedReportAgg3 = `
        service_name: "SERVICE_NAME"
        operations: <
          operation_name: "ListShelves"
          consumer_id: "api_key:api-key"
          labels: <
            key: "/credential_id"
            value: "apikey:api-key"
          >
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
            value: "ESPv2/"
          >
          labels: <
            key: "servicecontrol.googleapis.com/user_agent"
            value: "ESPv2"
          >
          labels: <
            key: "serviceruntime.googleapis.com/api_method"
            value: "ListShelves"
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/consumer/backend_latencies"
            metric_values: <
              distribution_value: <
                count: 3
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 3
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
                >
              >
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/consumer/error_count"
            metric_values: <
              int64_value: 3
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/consumer/request_bytes"
            metric_values: <
              int64_value: 600
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/consumer/request_count"
            metric_values: <
              int64_value: 3
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/consumer/request_overhead_latencies"
            metric_values: <
              distribution_value: <
                count: 3
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 3
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
                >
              >
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/consumer/request_sizes"
            metric_values: <
              distribution_value: <
                count: 3
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 3
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
            metric_name: "serviceruntime.googleapis.com/api/consumer/response_bytes"
            metric_values: <
              int64_value: 600
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/consumer/response_sizes"
            metric_values: <
              distribution_value: <
                count: 3
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 3
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
            metric_name: "serviceruntime.googleapis.com/api/consumer/streaming_durations"
            metric_values: <
              distribution_value: <
                count: 3
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 3
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
                >
              >
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/consumer/total_latencies"
            metric_values: <
              distribution_value: <
                count: 3
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 3
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
                >
              >
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/backend_latencies"
            metric_values: <
              distribution_value: <
                count: 3
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 3
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
                >
              >
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/error_count"
            metric_values: <
              int64_value: 3
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/request_bytes"
            metric_values: <
              int64_value: 600
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/request_count"
            metric_values: <
              int64_value: 3
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/request_overhead_latencies"
            metric_values: <
              distribution_value: <
                count: 3
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 3
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
                >
              >
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/request_sizes"
            metric_values: <
              distribution_value: <
                count: 3
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 3
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
            metric_name: "serviceruntime.googleapis.com/api/producer/response_bytes"
            metric_values: <
              int64_value: 600
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/response_sizes"
            metric_values: <
              distribution_value: <
                count: 3
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 3
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
            metric_name: "serviceruntime.googleapis.com/api/producer/streaming_durations"
            metric_values: <
              distribution_value: <
                count: 3
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 3
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
                >
              >
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/total_latencies"
            metric_values: <
              distribution_value: <
                count: 3
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 3
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
                >
              >
            >
          >
          log_entries: <
            name: "endpoints_log"
            severity: ERROR
            struct_payload: <
              fields: <
                key: "api_key"
                value: <
                  string_value: "api-key"
                >
              >
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
                key: "url"
                value: <
                  string_value: "/shelves"
                >
              >
              fields: <
                key: "client_ip"
                value: <
                  string_value: "127.0.0.1"
                >
              >
            >
          >
          log_entries: <
            name: "endpoints_log"
            severity: ERROR
            struct_payload: <
              fields: <
                key: "api_key"
                value: <
                  string_value: "api-key"
                >
              >
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
                key: "url"
                value: <
                  string_value: "/shelves"
                >
              >
              fields: <
                key: "client_ip"
                value: <
                  string_value: "127.0.0.1"
                >
              >
            >
          >
          log_entries: <
            name: "endpoints_log"
            severity: ERROR
            struct_payload: <
              fields: <
                key: "api_key"
                value: <
                  string_value: "api-key"
                >
              >
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
                key: "url"
                value: <
                  string_value: "/shelves"
                >
              >
              fields: <
                key: "client_ip"
                value: <
                  string_value: "127.0.0.1"
                >
              >
            >
          >
        >
        operations: <
          operation_name: "ListShelves"
          consumer_id: "api_key:api-key"
          labels: <
            key: "/credential_id"
            value: "apikey:api-key"
          >
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
            value: "ESPv2/"
          >
          labels: <
            key: "servicecontrol.googleapis.com/user_agent"
            value: "ESPv2"
          >
          labels: <
            key: "serviceruntime.googleapis.com/api_method"
            value: "ListShelves"
          >
          labels: <
            key: "serviceruntime.googleapis.com/consumer_project"
            value: "123456"
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/by_consumer/backend_latencies"
            metric_values: <
              distribution_value: <
                count: 3
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 3
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
                >
              >
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/by_consumer/error_count"
            metric_values: <
              int64_value: 3
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/by_consumer/request_count"
            metric_values: <
              int64_value: 3
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/by_consumer/request_overhead_latencies"
            metric_values: <
              distribution_value: <
                count: 3
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 3
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
                >
              >
            >
          >
          metric_value_sets: <
            metric_name: "serviceruntime.googleapis.com/api/producer/by_consumer/request_sizes"
            metric_values: <
              distribution_value: <
                count: 3
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 3
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
            metric_name: "serviceruntime.googleapis.com/api/producer/by_consumer/response_sizes"
            metric_values: <
              distribution_value: <
                count: 3
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 3
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
            metric_name: "serviceruntime.googleapis.com/api/producer/by_consumer/total_latencies"
            metric_values: <
              distribution_value: <
                count: 3
                mean: 1000
                minimum: 1000
                maximum: 1000
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 0
                bucket_counts: 3
                exponential_buckets: <
                  num_finite_buckets: 29
                  growth_factor: 2
                  scale: 1e-06
                >
              >
            >
          >
        >
        service_config_id: "SERVICE_CONFIG_ID"
	`

func TestCreateAggregateReport(t *testing.T) {
	got := CreateReport(&ExpectedReport{
		ServiceName:       "SERVICE_NAME",
		ServiceConfigID:   "SERVICE_CONFIG_ID",
		URL:               "/shelves",
		ApiMethod:         "ListShelves",
		ApiKey:            "api-key",
		ProducerProjectID: "endpoints-test",
		ConsumerProjectID: "123456",
		Location:          "us-central1",
		HttpMethod:        "GET",
		LogMessage:        "Method: ListShelves",
		ResponseCode:      503,
		StatusCode:        "14",
		ErrorType:         "5xx",
	})

	AggregateReport(&got, 3)
	want := scpb.ReportRequest{}
	if err := proto.UnmarshalText(expectedReportAgg3, &want); err != nil {
		t.Fatalf("proto.UnmarshalText3: %v", err)
	}
	if diff := ProtoDiff(&want, &got); diff != "" {
		glog.Infof("---Want---\n%v", proto.MarshalTextString(&want))
		glog.Infof("---Got---\n%v", proto.MarshalTextString(&got))
		t.Errorf("Aggregated report diff (-want, +got):\n%s", diff)
	}
}
