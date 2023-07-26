// Copyright 2022 Google LLC
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

package compression_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestCompressionTranscoded(t *testing.T) {
	t.Parallel()

	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--rollout_strategy=fixed",
		"--enable_response_compression"}

	s := env.NewTestEnv(platform.TestCompressionTranscoded, platform.GrpcBookstoreSidecar)
	s.OverrideAuthentication(&confpb.Authentication{
		Rules: []*confpb.AuthenticationRule{},
	})

	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	type testType struct {
		desc       string
		httpMethod string
		method     string
		token      string
		headers    map[string][]string
		wantResp   string
		wantDecode string
	}

	tests := []testType{
		{
			desc:       "accept-encode is [gzip]",
			httpMethod: "GET",
			method:     "/v1/shelves/100/books?key=api-key",
			headers:    http.Header{"accept-encoding": []string{"gzip"}},
			wantResp:   `{"books":[{"id":"1001","title":"Alphabet"}]}`,
			wantDecode: "gzip",
		},
		{
			desc:       "accept-encode is [br]",
			httpMethod: "GET",
			method:     "/v1/shelves/100/books?key=api-key",
			headers:    http.Header{"accept-encoding": []string{"br"}},
			wantResp:   `{"books":[{"id":"1001","title":"Alphabet"}]}`,
			wantDecode: "br",
		},
		{
			desc:       "accept-encode is [gzip,br]. choose gzip",
			httpMethod: "GET",
			method:     "/v1/shelves/100/books?key=api-key",
			headers:    http.Header{"accept-encoding": []string{"gzip,br"}},
			wantResp:   `{"books":[{"id":"1001","title":"Alphabet"}]}`,
			wantDecode: "gzip",
		},
		{
			desc:       "accept-encode is [gzip;q=0.1,br;q=0.9]. choose br",
			httpMethod: "GET",
			method:     "/v1/shelves/100/books?key=api-key",
			headers:    http.Header{"accept-encoding": []string{"gzip;q=0.1,br;q=0.9"}},
			wantResp:   `{"books":[{"id":"1001","title":"Alphabet"}]}`,
			wantDecode: "br",
		},
		{
			desc:       "accept-encode is [gzip;q=0.9,br;q=0.1]. choose gzip",
			httpMethod: "GET",
			method:     "/v1/shelves/100/books?key=api-key",
			headers:    http.Header{"accept-encoding": []string{"gzip;q=0.9,br;q=0.1"}},
			wantResp:   `{"books":[{"id":"1001","title":"Alphabet"}]}`,
			wantDecode: "gzip",
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			addr := fmt.Sprintf("%v:%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
			body, decode, err := client.MakeHttpCallWithDecode(addr, tc.httpMethod, tc.method, tc.token, tc.headers)
			if err != nil {
				t.Fatal(err)
			}
			if tc.wantDecode != decode {
				t.Errorf("Failed, got encoding_type: %q, want %q", decode, tc.wantDecode)
			}
			if err := util.JsonEqual(tc.wantResp, body); err != nil {
				t.Errorf("Failed, response body diff, err: %v", err)
			}
		})
	}
}
