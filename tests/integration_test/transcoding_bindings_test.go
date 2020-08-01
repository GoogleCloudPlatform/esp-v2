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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestTranscodingBindings(t *testing.T) {
	t.Parallel()

	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--rollout_strategy=fixed"}

	type testType struct {
		desc               string
		clientProtocol     string
		httpMethod         string
		method             string
		noBackend          bool
		token              string
		headers            map[string][]string
		bodyBytes          []byte
		wantResp           string
		wantErr            string
		wantGRPCWebTrailer client.GRPCWebTrailer
	}

	s := env.NewTestEnv(comp.TestTranscodingBindings, platform.GrpcBookstoreSidecar)
	s.OverrideAuthentication(&confpb.Authentication{
		Rules: []*confpb.AuthenticationRule{},
	})

	defer s.TearDown(t)
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	time.Sleep(time.Duration(5 * time.Second))

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
			bodyBytes:      []byte(`{"id":"300","theme":"Horror","any":{"@type":"type.googleapis.com/endpoints.examples.bookstore.Book","id":"123","author":"author","title":"title"}}`),
			wantResp:       `{"id":"300","theme":"Horror","any":{"@type":"type.googleapis.com/endpoints.examples.bookstore.Book","id":"123","author":"author","title":"title"}}`,
		},
	}
	for _, tc := range tests {
		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		resp, err := client.MakeHttpCallWithBody(addr, tc.httpMethod, tc.method, tc.token, tc.bodyBytes)
		if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
			t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantErr, err)
		} else {
			if !strings.Contains(resp, tc.wantResp) {
				t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
			}
		}
	}
}
