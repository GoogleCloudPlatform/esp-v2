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

package integration

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestTranscodingPrintOptions(t *testing.T) {
	t.Parallel()

	configID := "test-config-id"
	type testType struct {
		desc                                  string
		clientProtocol                        string
		httpMethod                            string
		method                                string
		bodyBytes                             []byte
		wantResp                              string
		transcodingAlwaysPrintPrimitiveFields bool
		transcodingAlwaysPrintEnumsAsInts     bool
		transcodingPreserveProtoFieldNames    bool
	}
	tests := []testType{
		{
			desc:           "Success. Default setting used to be compared with other test cases.",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves/100/books?key=api-key",
			bodyBytes:      []byte(`{"id": 4, "type": 1, "author":"Mark", "price_in_usd": 100}`),
			wantResp:       `{"id":"4","author":"Mark","type":"COMIC","price_in_usd":100}`,
		},
		{
			desc:                                  "Success. Set transcoding_always_print_primitive_fields to true",
			clientProtocol:                        "http",
			httpMethod:                            "POST",
			method:                                "/v1/shelves/100/books?key=api-key",
			bodyBytes:                             []byte(`{"id": 4}`),
			transcodingAlwaysPrintPrimitiveFields: true,
			wantResp:                              `{"id":"4","author":"","title":"","type":"CLASSIC","price_in_usd":0}`,
		},
		{
			desc:                              "Success. Set transcoding_always_print_enums_as_ints to true",
			clientProtocol:                    "http",
			httpMethod:                        "POST",
			method:                            "/v1/shelves/100/books?key=api-key",
			bodyBytes:                         []byte(`{"id": 4, "type":1}`),
			transcodingAlwaysPrintEnumsAsInts: true,
			wantResp:                          `{"id":"4","type":1}`,
		},
		{
			desc:                               "Success. Set transcoding_preserve_proto_field_names to true",
			clientProtocol:                     "http",
			httpMethod:                         "POST",
			method:                             "/v1/shelves/100/books?key=api-key",
			bodyBytes:                          []byte(`{"id": 4, "price_in_usd": 100}`),
			transcodingPreserveProtoFieldNames: true,
			wantResp:                           `{"id":"4","priceInUsd":100}`,
		},
	}
	for _, tc := range tests {
		func() {
			args := []string{"--service_config_id=" + configID,
				"--rollout_strategy=fixed"}

			if tc.transcodingAlwaysPrintPrimitiveFields {
				args = append(args, "--transcoding_always_print_primitive_fields=true")
			}

			if tc.transcodingAlwaysPrintEnumsAsInts {
				args = append(args, "--transcoding_always_print_enums_as_ints=true")
			}

			if tc.transcodingPreserveProtoFieldNames {
				args = append(args, "--transcoding_preserve_proto_field_names=true")
			}

			s := env.NewTestEnv(comp.TestTranscodingPrintOptions, platform.GrpcBookstoreSidecar)
			s.OverrideAuthentication(&confpb.Authentication{
				Rules: []*confpb.AuthenticationRule{},
			})

			defer s.TearDown(t)
			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
			resp, err := client.MakeHttpCallWithBody(addr, tc.httpMethod, tc.method, "", tc.bodyBytes)
			if err != nil {
				t.Errorf("Test (%s): failed with  err %v", tc.desc, err)
			} else {
				if !strings.Contains(resp, tc.wantResp) {
					t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
				}
			}
		}()
	}
}

func TestTranscodingIgnoreParameters(t *testing.T) {
	t.Parallel()

	configID := "test-config-id"
	type testType struct {
		desc                                    string
		clientProtocol                          string
		httpMethod                              string
		method                                  string
		bodyBytes                               []byte
		wantResp                                string
		wantError                               string
		transcodingIgnoreUnknownQueryParameters bool
		transcodingIgnoreQueryParameters        string
	}

	tests := []testType{
		{
			desc:           "Success. Default setting used to be compared with other test cases.",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves/100/books?key=api-key&unknown_parameter=val",
			bodyBytes:      []byte(`{"id": 4, "type": 1, "author":"Mark", "price_in_usd": 100}`),
			wantError:      "503 Service Unavailable",
		},
		{
			desc:                                    "Success. Set transcodingIgnoreUnknownQueryParameters to true.",
			clientProtocol:                          "http",
			httpMethod:                              "POST",
			method:                                  "/v1/shelves/100/books?key=api-key&unknown_parameter_foo=val&unknown_parameter_bar=val",
			bodyBytes:                               []byte(`{"id": 4, "type": 1, "author":"Mark", "price_in_usd": 100}`),
			transcodingIgnoreUnknownQueryParameters: true,
			wantResp:                                `{"id":"4","author":"Mark","type":"COMIC","price_in_usd":100}`,
		},
		{
			desc:                             "Fail. Set transcodingIgnoreQueryParameters with insufficient ignore parameters.",
			clientProtocol:                   "http",
			httpMethod:                       "POST",
			method:                           "/v1/shelves/100/books?key=api-key&unknown_parameter_foo=val&unknown_parameter_bar=val",
			bodyBytes:                        []byte(`{"id": 4, "type": 1, "author":"Mark", "price_in_usd": 100}`),
			transcodingIgnoreQueryParameters: "unknown_parameter_foo",
			wantError:                        "503 Service Unavailable",
		},
		{
			desc:                             "Success. Set right transcodingIgnoreQueryParameters.",
			clientProtocol:                   "http",
			httpMethod:                       "POST",
			method:                           "/v1/shelves/100/books?key=api-key&unknown_parameter_foo=val&unknown_parameter_bar=val",
			bodyBytes:                        []byte(`{"id": 4, "type": 1, "author":"Mark", "price_in_usd": 100}`),
			transcodingIgnoreQueryParameters: "unknown_parameter_foo,unknown_parameter_bar",
			wantResp:                         `{"id":"4","author":"Mark","type":"COMIC","price_in_usd":100}`,
		},
	}
	for _, tc := range tests {
		func() {
			args := []string{"--service_config_id=" + configID,
				"--rollout_strategy=fixed"}
			if tc.transcodingIgnoreUnknownQueryParameters {
				args = append(args, "--transcoding_ignore_unknown_query_parameters=true")
			}
			if tc.transcodingIgnoreQueryParameters != "" {
				args = append(args, "--transcoding_ignore_query_parameters="+tc.transcodingIgnoreQueryParameters)
			}

			s := env.NewTestEnv(comp.TestTranscodingIgnoreQueryParameters, platform.GrpcBookstoreSidecar)
			s.OverrideAuthentication(&confpb.Authentication{
				Rules: []*confpb.AuthenticationRule{},
			})

			defer s.TearDown(t)
			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
			resp, err := client.MakeHttpCallWithBody(addr, tc.httpMethod, tc.method, "", tc.bodyBytes)
			if err == nil {
				if !strings.Contains(resp, tc.wantResp) {
					t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
				}
				return
			}

			if tc.wantError == "" {
				t.Errorf("Test (%s): failed with  err %v", tc.desc, err)
				return
			}

			if !strings.Contains(err.Error(), tc.wantError) {
				t.Errorf("Test (%s): failed with unexpected error, want: %v, get: %s", tc.desc, err, tc.wantError)
			}
		}()
	}
}
