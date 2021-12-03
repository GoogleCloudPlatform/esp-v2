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

package transcoding_bindings_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestTranscodingBindings(t *testing.T) {
	t.Parallel()

	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--rollout_strategy=fixed"}

	s := env.NewTestEnv(platform.TestTranscodingBindings, platform.GrpcBookstoreSidecar)
	s.OverrideAuthentication(&confpb.Authentication{
		Rules: []*confpb.AuthenticationRule{},
	})

	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	type testType struct {
		desc               string
		clientProtocol     string
		httpMethod         string
		method             string
		token              string
		headers            map[string][]string
		bodyBytes          []byte
		wantResp           string
		wantErr            string
		wantGRPCWebTrailer client.GRPCWebTrailer
	}

	tests := []testType{
		// Binding shelf=100 in ListBooksRequest
		// HTTP template:
		// GET /shelves/{shelf}/books
		{
			desc:           "Succeeded, made a ListBookRequest",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves/100?key=api-key",
			wantResp:       `{"id":"100","theme":"Kids"}`,
		},
		// Binding shelf=100 and book=<post body> in CreateBookRequest
		// HTTP template:
		// POST /shelves/{shelf}/books
		// body: book
		{
			desc:           "Succeeded, made CreateBookRequest1",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves/100/books?key=api-key",
			bodyBytes:      []byte(`{"id": 4, "author" : "Leo Tolstoy", "title" : "War and Peace"}`),
			wantResp:       `{"id":"4","author":"Leo Tolstoy","title":"War and Peace"}`,
		},
		// Binding shelf=100, book.id=5, book.author="Leo Tolstoy" and book.title=<post body>
		// in CreateBookRequest.
		// HTTP template:
		// POST /shelves/{shelf}/books/{book.id}/{book.author}
		// body: book
		{
			desc:           "Succeeded, made CreateBookRequest2",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves/100/books/5/Mark?key=api-key",
			bodyBytes:      []byte(`{"title" : "The Adventures of Huckleberry Finn"}`),
			wantResp:       `{"id":"5","author":"Mark","title":"The Adventures of Huckleberry Finn"}`,
		},
		// Binding shelf=100, book.id=6, book.author="Foo/Bar/Baz" and book.title=<post body>
		// in CreateBookRequest.
		// HTTP template:
		// POST /shelves/{shelf}/books/{book.id}/{book.author}
		{
			desc:           "Succeeded, specifically test escaped slash  in the URL path",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves/100/books/6/Foo%2FBar%2FBaz?key=api-key",
			bodyBytes:      []byte(`{"title" : "The Adventures of Huckleberry Finn"}`),
			wantResp:       `{"id":"6","author":"Foo/Bar/Baz","title":"The Adventures of Huckleberry Finn"}`,
		},
		// Binding shelf=100 and book=100 in DeleteBookRequest
		// HTTP template:
		// DELETE /shelves/{shelf}/books/{book}
		{
			desc:           "Succeeded, made DeleteBookRequest",
			clientProtocol: "http",
			httpMethod:     "DELETE",
			method:         "/v1/shelves/100/books/100?key=api-key",
			wantResp:       `{}`,
		},
		// Binding shelf=100 in ListBooksRequest
		// HTTP template:
		// GET /shelves/{shelf}/books
		{
			desc:           "Succeeded, check the result after all the requests",
			clientProtocol: "http",
			httpMethod:     "GET",
			method:         "/v1/shelves/100/books?key=api-key",
			wantResp:       `{"books":[{"id":"1001","title":"Alphabet"},{"id":"4","author":"Leo Tolstoy","title":"War and Peace"},{"id":"5","author":"Mark","title":"The Adventures of Huckleberry Finn"},{"id":"6","author":"Foo/Bar/Baz","title":"The Adventures of Huckleberry Finn"}]}`,
		},
		{
			desc:           "Succeed, test basic Any type transcoding by echoing book in Any",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves?key=api-key",
			bodyBytes:      []byte(`{"id":"300","theme":"Horror","any":{"@type":"type.googleapis.com/endpoints.examples.bookstore.ObjectOnlyForAny","id":"123","name":"name"}}`),
			wantResp:       `{"id":"300","theme":"Horror","any":{"@type":"type.googleapis.com/endpoints.examples.bookstore.ObjectOnlyForAny","id":123,"name":"name"}}`,
		},
		// Binding shelf=100 and some fields in query parameters in CreateBookRequest
		// query parameter + has been unescaped into space.
		// HTTP template:
		// POST /shelves/{shelf}/books
		// body: book
		{
			desc:           "Succeeded, call CreateBookRequest with query parameters",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves/100/books?key=api-key&book.author=Leo%20Tolstoy&book.title=War+and+Peace",
			bodyBytes:      []byte(`{"id": 14}`),
			wantResp:       `{"id":"14","author":"Leo Tolstoy","title":"War and Peace"}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			addr := fmt.Sprintf("%v:%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
			resp, err := client.MakeHttpCallWithBody(addr, tc.httpMethod, tc.method, tc.token, tc.bodyBytes)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantErr, err)
				}
			} else {
				if err := util.JsonEqual(tc.wantResp, resp); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func TestTranscodingBindingsForCustomVerb(t *testing.T) {
	type testType struct {
		desc                                 string
		clientProtocol                       string
		httpMethod                           string
		method                               string
		noBackend                            bool
		token                                string
		includeColonInUrlPathWildcardSegment bool
		headers                              map[string][]string
		bodyBytes                            []byte
		wantResp                             string
		wantErr                              string
		wantGRPCWebTrailer                   client.GRPCWebTrailer
	}

	tests := []testType{
		// The test case for backwards compatibility.
		{
			desc:                                 "Failed, registered custom verb is matched with trailing single wildcard in route but not in transcoder filter",
			clientProtocol:                       "http",
			httpMethod:                           "POST",
			includeColonInUrlPathWildcardSegment: true,
			method:                               "/v1/shelves/100/single/random:registeredCustomVerb?key=api-key",
			bodyBytes:                            []byte(`{"id": 5, "author" : "Leo Tolstoy", "title" : "War and Peace"}`),
			wantErr:                              `503 Service Unavailable, {"code":503,"message":"upstream connect error or disconnect/reset before headers. reset reason: remote reset"}`,
		},
		// The test case for backwards compatibility.
		{
			desc:                                 "Failed, registered custom verb is matched with trailing double wildcard in route but not in transcoder filter",
			clientProtocol:                       "http",
			httpMethod:                           "POST",
			includeColonInUrlPathWildcardSegment: true,
			method:                               "/v1/shelves/100/double/random:registeredCustomVerb?key=api-key",
			bodyBytes:                            []byte(`{"id": 5, "author" : "Leo Tolstoy", "title" : "War and Peace"}`),
			wantErr:                              `503 Service Unavailable, {"code":503,"message":"upstream connect error or disconnect/reset before headers. reset reason: remote reset"}`,
		},
		{
			desc:           "Failed, registered custom verb is not matched with trailing single wildcard by either route regex or transcoder filter",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves/100/single/random:registeredCustomVerb?key=api-key",
			bodyBytes:      []byte(`{"id": 5, "author" : "Leo Tolstoy", "title" : "War and Peace"}`),
			wantErr:        `404 Not Found, {"code":404,"message":"The current request is not defined by this API."}`,
		},
		{
			desc:           "Failed, registered custom verb is not matched with trailing single wildcard by either route regex or transcoder filter",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves/100/double/random:registeredCustomVerb?key=api-key",
			bodyBytes:      []byte(`{"id": 5, "author" : "Leo Tolstoy", "title" : "War and Peace"}`),
			wantErr:        `404 Not Found, {"code":404,"message":"The current request is not defined by this API."}`,
		},
		{
			// TODO(b/208716168): it should returns 404 for this request. Right now, the url
			// `/v1/shelves/100/books/random:unregisteredCustomVerb` is matched with `/v1/shelves/{shelf}/books/*`
			// from API method  CreateBookWithTrailingSingleWildcard.
			desc:           "Succeed, unregistered custom verb is matched in transcoder filter with trailing wildcard",
			clientProtocol: "http",
			httpMethod:     "POST",
			method:         "/v1/shelves/100/single/random:unregisteredCustomVerb?key=api-key",
			bodyBytes:      []byte(`{"id": 5, "author" : "Leo Tolstoy", "title" : "War and Peace"}`),
			wantResp:       `{"id":"5","author":"Leo Tolstoy","title":"War and Peace"}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			s := env.NewTestEnv(platform.TestTranscodingBindings, platform.GrpcBookstoreSidecar)
			s.OverrideAuthentication(&confpb.Authentication{
				Rules: []*confpb.AuthenticationRule{},
			})

			defer s.TearDown(t)
			args := utils.CommonArgs()
			if tc.includeColonInUrlPathWildcardSegment {
				args = append(args, "--include_colon_in_wildcard_path_segment")
			}
			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			addr := fmt.Sprintf("%v:%v", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
			resp, err := client.MakeHttpCallWithBody(addr, tc.httpMethod, tc.method, tc.token, tc.bodyBytes)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantErr, err)
				}
			} else {
				if err := util.JsonEqual(tc.wantResp, resp); err != nil {
					t.Error(err)
				}
			}
		})
	}
}
