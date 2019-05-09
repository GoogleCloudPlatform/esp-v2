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
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"cloudesf.googlesource.com/gcpproxy/src/go/util"
	"cloudesf.googlesource.com/gcpproxy/tests/endpoints/echo/client"
	"cloudesf.googlesource.com/gcpproxy/tests/env"
	"cloudesf.googlesource.com/gcpproxy/tests/env/testdata"
	"cloudesf.googlesource.com/gcpproxy/tests/utils"
	"google.golang.org/genproto/googleapis/api/annotations"

	bsClient "cloudesf.googlesource.com/gcpproxy/tests/endpoints/bookstore-grpc/client"
	comp "cloudesf.googlesource.com/gcpproxy/tests/env/components"
	sc "github.com/google/go-genproto/googleapis/api/servicecontrol/v1"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestServiceControlBasic(t *testing.T) {
	serviceName := "test-echo"
	configId := "test-config-id"

	args := []string{"--service=" + serviceName, "--version=" + configId,
		"--backend_protocol=http1", "--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(comp.TestServiceControlBasic, "echo", []string{"google_jwt"})
	s.AppendHttpRules([]*annotations.HttpRule{
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simpleget",
			Pattern: &annotations.HttpRule_Get{
				Get: "/simpleget",
			},
		},
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey_override_as_get",
			Pattern: &annotations.HttpRule_Get{
				Get: "/echo/nokey/OverrideAsGet",
			},
		},
	})
	s.AppendUsageRules(
		[]*conf.UsageRule{
			{
				Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey_override_as_get",
				AllowUnregisteredCalls: true,
			},
		})

	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

	testData := []struct {
		desc                  string
		url                   string
		method                string
		requestHeader         map[string]string
		message               string
		wantResp              string
		httpCallError         error
		wantScRequests        []interface{}
		wantGetScRequestError error
	}{
		{
			desc:     "succeed GET, no Jwt required, service control sends check request and report request for GET request",
			url:      fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/simpleget", "?key=api-key"),
			method:   "GET",
			message:  "",
			wantResp: "simple get message",
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.APIProxyVersion,
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simpleget",
					CallerIp:        "127.0.0.1",
				},
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/simpleget?key=api-key",
					ApiKey:            "api-key",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simpleget",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					FrontendProtocol:  "http",
					HttpMethod:        "GET",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simpleget is called",
					StatusCode:        "0",
					RequestSize:       178,
					ResponseSize:      125,
					RequestBytes:      178,
					ResponseBytes:     125,
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:     "succeed, no Jwt required, service control sends check request and report request for POST request",
			url:      fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo", "?key=api-key"),
			method:   "POST",
			message:  "hello",
			wantResp: `{"message":"hello"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedCheck{
					Version:         utils.APIProxyVersion,
					ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID: "test-config-id",
					ConsumerID:      "api_key:api-key",
					OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					CallerIp:        "127.0.0.1",
				},
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/echo?key=api-key",
					ApiKey:            "api-key",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					FrontendProtocol:  "http",
					HttpMethod:        "POST",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo is called",
					StatusCode:        "0",
					RequestSize:       238,
					ResponseSize:      126,
					RequestBytes:      238,
					ResponseBytes:     126,
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:          "succeed, no Jwt required, service control sends report request only for request that does not have API key but is actually required",
			url:           fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo"),
			method:        "POST",
			message:       "hello",
			httpCallError: fmt.Errorf("http response status is not 200 OK: 401 Unauthorized"),
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/echo",
					ErrorType:         "4xx",
					StatusCode:        "16",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					FrontendProtocol:  "http",
					HttpMethod:        "POST",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo is called",
					RequestSize:       206,
					ResponseSize:      281,
					RequestBytes:      206,
					ResponseBytes:     281,
					ResponseCode:      401,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:     "succeed, no Jwt required, allow no api key (unregistered request), service control sends report only",
			url:      fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo/nokey"),
			message:  "hello",
			method:   "POST",
			wantResp: `{"message":"hello"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/echo/nokey",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					HttpMethod:        "POST",
					FrontendProtocol:  "http",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey is called",
					StatusCode:        "0",
					RequestSize:       232,
					ResponseSize:      126,
					RequestBytes:      232,
					ResponseBytes:     126,
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:     "succeed, no Jwt required, allow no api key (there is API key in request though it is not required), service control sends report only",
			url:      fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo/nokey", "?key=api-key"),
			message:  "hello",
			method:   "POST",
			wantResp: `{"message":"hello"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/echo/nokey?key=api-key",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					HttpMethod:        "POST",
					FrontendProtocol:  "http",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey is called",
					StatusCode:        "0",
					RequestSize:       244,
					ResponseSize:      126,
					RequestBytes:      244,
					ResponseBytes:     126,
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:    "succeed with request with referer header, no Jwt required, allow no api key (unregistered request), service control sends report (with referer information) only",
			url:     fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo/nokey"),
			message: "hi",
			method:  "POST",
			requestHeader: map[string]string{
				"Referer": "http://google.com/bookstore/root",
			},
			wantResp: `{"message":"hi"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/echo/nokey",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					HttpMethod:        "POST",
					FrontendProtocol:  "http",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey is called",
					Referer:           "http://google.com/bookstore/root",
					StatusCode:        "0",
					RequestSize:       268,
					ResponseSize:      123,
					RequestBytes:      268,
					ResponseBytes:     123,
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:    "succeed, no Jwt required,no api key required, with X-HTTP-Method-Override as GET, service control sends report only and it has GET as HTTP method",
			url:     fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo/nokey/OverrideAsGet"),
			message: "hello hello",
			method:  "POST",
			requestHeader: map[string]string{
				"X-HTTP-Method-Override": "GET",
			},
			wantResp: `{"message":"hello hello"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/echo/nokey/OverrideAsGet",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey_override_as_get",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					HttpMethod:        "GET",
					FrontendProtocol:  "http",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey_override_as_get is called",
					StatusCode:        "0",
					RequestSize:       277,
					ResponseSize:      132,
					RequestBytes:      277,
					ResponseBytes:     132,
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:     "succeed for unconfigured requests with any path (/**) and POST method, no JWT required, service control sends report request only",
			url:      fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/anypath/x/y/z"),
			method:   "POST",
			message:  "hello",
			wantResp: `{"message":"hello"}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/anypath/x/y/z",
					ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog._post_anypath",
					ProducerProjectID: "producer-project",
					ConsumerProjectID: "123456",
					HttpMethod:        "POST",
					FrontendProtocol:  "http",
					LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog._post_anypath is called",
					StatusCode:        "0",
					RequestSize:       235,
					ResponseSize:      126,
					RequestBytes:      235,
					ResponseBytes:     126,
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:                  "fail for not allowing unconfigured GET method, service control does not send report request since path matcher filter has already reject the request",
			url:                   fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/unconfiguredRequest/get"),
			method:                "GET",
			httpCallError:         fmt.Errorf("http response status is not 200 OK: 404 Not Found"),
			wantScRequests:        []interface{}{},
			wantGetScRequestError: fmt.Errorf("Timeout got 0, expected: 1"),
		},
	}
	for _, tc := range testData {
		var resp []byte
		var err error
		if tc.method == "POST" {
			resp, err = client.DoPostWithHeaders(tc.url, tc.message, tc.requestHeader)
		} else if tc.method == "GET" {
			resp, err = client.DoGet(tc.url)
		} else {
			t.Fatalf("Test (%s): failed, unknown HTTP method to call", tc.desc)
		}
		if tc.httpCallError == nil {
			if err != nil {
				t.Fatalf("Test (%s): failed, %v", tc.desc, err)
			}
			if !strings.Contains(string(resp), tc.wantResp) {
				t.Errorf("Test (%s): failed,  expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
			}
		} else {
			if tc.httpCallError.Error() != err.Error() {
				t.Errorf("Test (%s): failed,  expected Http call error: %v, got: %v", tc.desc, tc.httpCallError, err)
			}
		}

		if tc.wantGetScRequestError != nil {
			scRequests, err1 := s.ServiceControlServer.GetRequests(1)
			if err1 == nil || err1.Error() != tc.wantGetScRequestError.Error() {
				t.Errorf("Test (%s): failed", tc.desc)
				t.Errorf("expected get service control request call error: %v, got: %v", tc.wantGetScRequestError, err1)
				t.Errorf("got service control requests: %v", scRequests)
			}
			continue
		}

		scRequests, err1 := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
		if err1 != nil {
			t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err1)
		}
		utils.CheckScRequest(t, scRequests, tc.wantScRequests, tc.desc)

	}

}

func TestServiceControlCache(t *testing.T) {
	serviceName := "test-echo"
	configId := "test-config-id"

	args := []string{"--service=" + serviceName, "--version=" + configId,
		"--backend_protocol=http1", "--rollout_strategy=fixed", "--suppress_envoy_headers"}

	s := env.NewTestEnv(comp.TestServiceControlCache, "echo", []string{"google_jwt"})
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

	url := fmt.Sprintf("http://localhost:%v%v%v", s.Ports().ListenerPort, "/echo", "?key=api-key")
	message := "hello"
	num := 10
	wantResp := `{"message":"hello"}`
	for i := 0; i < num; i++ {
		resp, err := client.DoPost(url, message)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(resp), wantResp) {
			t.Errorf("expected: %s, got: %s", wantResp, string(resp))
		}
	}

	wantScRequests := []interface{}{
		&utils.ExpectedCheck{
			Version:         utils.APIProxyVersion,
			ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
			ServiceConfigID: "test-config-id",
			ConsumerID:      "api_key:api-key",
			OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
			CallerIp:        "127.0.0.1",
		},
		&utils.ExpectedReport{
			Aggregate:         int64(num),
			Version:           utils.APIProxyVersion,
			ServiceName:       "echo-api.endpoints.cloudesf-testing.cloud.goog",
			ServiceConfigID:   "test-config-id",
			URL:               "/echo?key=api-key",
			ApiKey:            "api-key",
			ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
			ProducerProjectID: "producer-project",
			ConsumerProjectID: "123456",
			FrontendProtocol:  "http",
			HttpMethod:        "POST",
			LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo is called",
			StatusCode:        "0",
			RequestSize:       238,
			ResponseSize:      126,
			RequestBytes:      238,
			ResponseBytes:     126,
			ResponseCode:      200,
			Platform:          util.GCE,
			Location:          "test-zone",
		},
	}

	scRequests, err := s.ServiceControlServer.GetRequests(len(wantScRequests))
	if err != nil {
		t.Fatalf("GetRequests returns error: %v", err)
	}

	utils.CheckScRequest(t, scRequests, wantScRequests, "TestServiceControlCache")
}

func TestServiceControlCredentialId(t *testing.T) {
	serviceName := "test-bookstore"
	configId := "test-config-id"

	args := []string{"--service=" + serviceName, "--version=" + configId,
		"--backend_protocol=grpc", "--rollout_strategy=fixed", "--suppress_envoy_headers",
	}
	s := env.NewTestEnv(comp.TestServiceControlLogJwtPayloads, "bookstore", []string{"google_jwt"})

	s.OverrideAuthentication(&conf.Authentication{Rules: []*conf.AuthenticationRule{
		{
			Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
			Requirements: []*conf.AuthRequirement{
				{
					ProviderId: "google_jwt",
				},
			},
		},
	},
	})

	s.AppendUsageRules([]*conf.UsageRule{
		{
			Selector:               "endpoints.examples.bookstore.Bookstore.ListShelves",
			AllowUnregisteredCalls: true,
		},
	})

	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()

	testData := []struct {
		desc                  string
		clientProtocol        string
		method                string
		httpMethod            string
		token                 string
		requestHeader         map[string]string
		message               string
		usageRules            []*conf.UsageRule
		authenticationRules   []*conf.AuthenticationRule
		wantResp              string
		httpCallError         error
		wantScRequests        []interface{}
		wantGetScRequestError error
	}{
		{
			desc:           "success; When api_key is unavaliable, the label credential_id is iss and the check request is skipped",
			clientProtocol: "http",
			method:         "/v1/shelves",
			httpMethod:     "GET",
			token:          testdata.FakeCloudToken,
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/v1/shelves",
					JwtAuth:           "issuer=YXBpLXByb3h5LXRlc3RpbmdAY2xvdWQuZ29vZw",
					ApiMethod:         "endpoints.examples.bookstore.Bookstore.ListShelves",
					ProducerProjectID: "producer project",
					ConsumerProjectID: "123456",
					FrontendProtocol:  "http",
					BackendProtocol:   "grpc",
					HttpMethod:        "GET",
					LogMessage:        "endpoints.examples.bookstore.Bookstore.ListShelves is called",
					StatusCode:        "0",
					RequestSize:       167,
					ResponseSize:      291,
					RequestBytes:      167,
					ResponseBytes:     291,
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
		{
			desc:           "success; When api_key is unavaliable, the label credential_id is iss plus aud and the check request is skipped",
			clientProtocol: "http",
			method:         "/v1/shelves",
			httpMethod:     "GET",
			token:          testdata.FakeCloudTokenSingleAudience1,
			wantResp:       `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
			wantScRequests: []interface{}{
				&utils.ExpectedReport{
					Version:           utils.APIProxyVersion,
					ServiceName:       "bookstore.endpoints.cloudesf-testing.cloud.goog",
					ServiceConfigID:   "test-config-id",
					URL:               "/v1/shelves",
					JwtAuth:           "issuer=YXBpLXByb3h5LXRlc3RpbmdAY2xvdWQuZ29vZw&audience=Ym9va3N0b3JlX3Rlc3RfY2xpZW50LmNsb3VkLmdvb2c",
					ApiMethod:         "endpoints.examples.bookstore.Bookstore.ListShelves",
					ProducerProjectID: "producer project",
					ConsumerProjectID: "123456",
					FrontendProtocol:  "http",
					BackendProtocol:   "grpc",
					HttpMethod:        "GET",
					LogMessage:        "endpoints.examples.bookstore.Bookstore.ListShelves is called",
					StatusCode:        "0",
					RequestSize:       167,
					ResponseSize:      291,
					RequestBytes:      167,
					ResponseBytes:     291,
					ResponseCode:      200,
					Platform:          util.GCE,
					Location:          "test-zone",
				},
			},
		},
	}

	for _, tc := range testData {
		addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
		resp, err := bsClient.MakeCall(tc.clientProtocol, addr, tc.httpMethod, tc.method, tc.token, http.Header{})

		if tc.httpCallError == nil {
			if err != nil {
				t.Fatalf("Test (%s): failed, %v", tc.desc, err)
			}
			if !strings.Contains(string(resp), tc.wantResp) {
				t.Errorf("Test (%s): failed,  expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
			}
		} else {
			if tc.httpCallError.Error() != err.Error() {
				t.Errorf("Test (%s): failed,  expected Http call error: %v, got: %v", tc.desc, tc.httpCallError, err)
			}
		}

		if tc.wantGetScRequestError != nil {
			scRequests, err1 := s.ServiceControlServer.GetRequests(1)
			if err1.Error() != tc.wantGetScRequestError.Error() {
				t.Errorf("Test (%s): failed", tc.desc)
				t.Errorf("expected get service control request call error: %v, got: %v", tc.wantGetScRequestError, err1)
				t.Errorf("got service control requests: %v", scRequests)
			}
			continue
		}

		scRequests, err1 := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
		if err1 != nil {
			t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err1)
		}
		utils.CheckScRequest(t, scRequests, tc.wantScRequests, tc.desc)
	}
}

func TestAuthOKCheckFail(t *testing.T) {
	serviceName := "echo-service"
	configID := "test-config-id"
	args := []string{"--service=" + serviceName, "--version=" + configID,
		"--backend_protocol=http1", "--rollout_strategy=fixed"}

	s := env.NewTestEnv(comp.TestAuthOKCheckFail, "echo", []string{"google_jwt"})

	comp.ResetReqCnt()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}
	defer s.TearDown()
	s.ServiceControlServer.SetCheckResponse(
		&sc.CheckResponse{
			CheckInfo: &sc.CheckResponse_CheckInfo{
				ConsumerInfo: &sc.CheckResponse_ConsumerInfo{
					ProjectNumber: 123456,
				},
			},
			CheckErrors: []*sc.CheckError{
				&sc.CheckError{
					Code: sc.CheckError_PROJECT_INVALID,
				},
			},
		},
	)
	type reqCnt struct {
		key string
		cnt int
	}
	time.Sleep(time.Duration(5 * time.Second))
	tc := struct {
		desc           string
		httpMethod     string
		httpPath       string
		apiKey         string
		token          string
		wantMsReq      reqCnt
		wantPdReq      reqCnt
		wantError      string
		wantScRequests []interface{}
	}{
		desc:       "Auth passed but check failed",
		httpMethod: "GET",
		httpPath:   "/auth/info/googlejwt",
		apiKey:     "api-key",
		token:      testdata.FakeCloudToken,
		wantMsReq:  reqCnt{"/v1/instance/service-accounts/default/token", 1},
		wantPdReq:  reqCnt{"google_jwt", 1},
		wantError:  "400 Bad Request, INVALID_ARGUMENT:Client project not valid. Please pass a valid project",
		wantScRequests: []interface{}{
			&utils.ExpectedCheck{
				Version:         utils.APIProxyVersion,
				ServiceName:     "echo-api.endpoints.cloudesf-testing.cloud.goog",
				ServiceConfigID: "test-config-id",
				ConsumerID:      "api_key:api-key",
				OperationName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth_info_google_jwt",
				CallerIp:        "127.0.0.1",
			},
			&utils.ExpectedReport{
				Version:     utils.APIProxyVersion,
				ServiceName: "echo-api.endpoints.cloudesf-testing.cloud.goog", ServiceConfigID: "test-config-id",
				URL:               "/auth/info/googlejwt?key=api-key",
				ApiKey:            "api-key",
				ApiMethod:         "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth_info_google_jwt",
				ProducerProjectID: "producer-project",
				ConsumerProjectID: "123456",
				FrontendProtocol:  "http",
				HttpMethod:        "GET",
				LogMessage:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth_info_google_jwt is called",
				ErrorType:         "4xx",
				StatusCode:        "3",
				RequestSize:       188,
				ResponseSize:      163,
				RequestBytes:      188,
				ResponseBytes:     163,
				ResponseCode:      400,
				Platform:          util.GCE,
				Location:          "test-zone",
			},
		},
	}

	host := fmt.Sprintf("http://localhost:%v", s.Ports().ListenerPort)
	_, err := client.DoJWT(host, tc.httpMethod, tc.httpPath, tc.apiKey, "", tc.token)

	if realCnt := s.MockMetadataServer.GetReqCnt(tc.wantMsReq.key); realCnt != tc.wantMsReq.cnt {
		t.Errorf("Test (%s): failed, %s on MetadataServer should be requested by %v times not %v times.", tc.desc, tc.wantPdReq.key, tc.wantMsReq.cnt, realCnt)
	}

	provider, ok := comp.JwtProviders[tc.wantPdReq.key]
	if !ok {
		t.Errorf("Test (%s): failed, the provider is not inited.", tc.desc)
	} else if realCnt := provider.GetReqCnt(); realCnt != tc.wantPdReq.cnt {
		t.Errorf("Test (%s): failed, pubkey of %s shoud be fetched by %v times instead of %v times.", tc.desc, tc.wantPdReq.key, tc.wantPdReq.cnt, realCnt)
	}

	if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
		t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
	}

	scRequests, err1 := s.ServiceControlServer.GetRequests(len(tc.wantScRequests))
	if err1 != nil {
		t.Fatalf("Test (%s): failed, GetRequests returns error: %v", tc.desc, err1)
	}
	utils.CheckScRequest(t, scRequests, tc.wantScRequests, tc.desc)
}
