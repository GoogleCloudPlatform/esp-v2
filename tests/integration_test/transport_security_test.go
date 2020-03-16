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

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	bsclient "github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/bookstore_grpc/client"
	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
)

func TestServiceManagementWithTLS(t *testing.T) {
	t.Parallel()

	testData := []struct {
		desc         string
		certPath     string
		keyPath      string
		port         uint16
		wantResp     string
		wantSetupErr string
	}{
		{
			desc:     "Succeed, ServiceManagement HTTPS server uses same cert as proxy",
			certPath: platform.GetFilePath(platform.ProxyCert),
			keyPath:  platform.GetFilePath(platform.ProxyKey),
			port:     comp.TestServiceManagementWithValidCert,
			wantResp: `{"message":"hello"}`,
		},
		{
			desc:         "Fail, ServiceManagement HTTPS server uses different cert as proxy",
			certPath:     platform.GetFilePath(platform.ServerCert),
			keyPath:      platform.GetFilePath(platform.ServerKey),
			port:         comp.TestServiceManagementWithInvalidCert,
			wantSetupErr: "health check response was not healthy",
		},
	}

	for _, tc := range testData {
		func() {
			s := env.NewTestEnv(tc.port, platform.EchoSidecar)
			defer s.TearDown()
			serverCerts, err := comp.GenerateCert(tc.certPath, tc.keyPath)
			if err != nil {
				t.Fatalf("fial to generate cert: %v", err)
			}

			s.MockServiceManagementServer.SetCert(serverCerts)
			err = s.Setup(utils.CommonArgs())

			if tc.wantSetupErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantSetupErr) {
					t.Errorf("Test (%s): failed, want error: %v, got error: %v", tc.desc, tc.wantSetupErr, err)
				}
			} else {
				url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo?key=api-key")
				resp, err := client.DoPost(url, "hello")
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(resp), tc.wantResp) {
					t.Errorf("expected: %s, got: %s", tc.wantResp, string(resp))
				}
			}
		}()
	}
}

func TestServiceControlWithTLS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc      string
		certPath  string
		keyPath   string
		port      uint16
		token     string
		wantResp  string
		wantError string
	}{
		{
			desc:     "Succeed, ServiceControl HTTPS server uses same cert as proxy",
			token:    testdata.FakeCloudTokenMultiAudiences,
			certPath: platform.GetFilePath(platform.ProxyCert),
			keyPath:  platform.GetFilePath(platform.ProxyKey),
			wantResp: `{"shelves":[{"id":"100","theme":"Kids"},{"id":"200","theme":"Classic"}]}`,
		},
		{
			desc:      "Failed to call ServiceControl HTTPS server, with different Cert as proxy",
			token:     testdata.FakeCloudTokenMultiAudiences,
			port:      comp.TestServiceControlTLSWithValidCert,
			certPath:  platform.GetFilePath(platform.ServerCert),
			keyPath:   platform.GetFilePath(platform.ServerKey),
			wantError: "500 Internal Server Error, INTERNAL:Failed to call service control",
		},
	}

	for _, tc := range tests {
		func() {
			s := env.NewTestEnv(tc.port, platform.GrpcBookstoreSidecar)
			defer s.TearDown()
			serverCerts, err := comp.GenerateCert(tc.certPath, tc.keyPath)
			if err != nil {
				t.Fatalf("fail to create cert, %v", err)
			}
			s.ServiceControlServer.SetCert(serverCerts)

			if err := s.Setup(utils.CommonArgs()); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			s.ServiceControlServer.ResetRequestCount()
			addr := fmt.Sprintf("localhost:%v", s.Ports().ListenerPort)
			resp, err := bsclient.MakeCall("http", addr, "GET", "/v1/shelves?key=api-key", tc.token, nil)
			if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Errorf("Test (%s): failed, expected err: %v, got: %v", tc.desc, tc.wantError, err)
			} else if !strings.Contains(resp, tc.wantResp) {
				t.Errorf("Test (%s): failed, expected: %s, got: %s", tc.desc, tc.wantResp, resp)
			}
		}()
	}
}

func TestHttpsClients(t *testing.T) {
	t.Parallel()
	args := utils.CommonArgs()
	args = append(args, "--ssl_server_cert_path=../env/testdata/")

	s := env.NewTestEnv(comp.TestHttpsClients, platform.EchoSidecar)
	defer s.TearDown()
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simpleget",
			Pattern: &annotationspb.HttpRule_Get{
				Get: "/simpleget",
			},
		},
	})

	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc         string
		httpsVersion int
		certPath     string
		port         uint16
		wantResp     string
		wantError    error
	}{
		{
			desc:         "Succcess for HTTP1 client with TLS",
			httpsVersion: 1,
			certPath:     platform.GetFilePath(platform.ServerCert),
			wantResp:     `simple get message`,
		},
		{
			desc:         "Succcess for HTTP2 client with TLS",
			httpsVersion: 2,
			certPath:     platform.GetFilePath(platform.ServerCert),
			wantResp:     `simple get message`,
		},
		{
			desc:         "Fail for HTTP1 client, with incorrect key and cert",
			httpsVersion: 1,
			certPath:     platform.GetFilePath(platform.ProxyCert),
			wantError:    fmt.Errorf("x509: certificate signed by unknown authority"),
		},
		{
			desc:         "Fail for HTTP2 client, with incorrect key and cert",
			httpsVersion: 2,
			certPath:     platform.GetFilePath(platform.ProxyCert),
			wantError:    fmt.Errorf("x509: certificate signed by unknown authority"),
		},
	}

	for _, tc := range testData {
		var resp []byte
		var err error

		url := fmt.Sprintf("https://localhost:%v/simpleget?key=api-key", s.Ports().ListenerPort)
		_, resp, err = client.DoHttpsGet(url, tc.httpsVersion, tc.certPath)
		if tc.wantError == nil {
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(resp), tc.wantResp) {
				t.Errorf("Test desc (%v) expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
			}
		} else if !strings.Contains(err.Error(), tc.wantError.Error()) {
			t.Errorf("Test (%s): failed\nexpected: %v\ngot: %v", tc.desc, tc.wantError, err)
		}
	}
}

func TestHSTS(t *testing.T) {
	t.Parallel()
	args := utils.CommonArgs()
	args = append(args, "--ssl_server_cert_path=../env/testdata/")
	args = append(args, "--enable_strict_transport_security")

	s := env.NewTestEnv(comp.TestHttpsClients, platform.EchoSidecar)
	defer s.TearDown()
	s.AppendHttpRules([]*annotationspb.HttpRule{
		{
			Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simpleget",
			Pattern: &annotationspb.HttpRule_Get{
				Get: "/simpleget",
			},
		},
	})
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc           string
		httpsVersion   int
		certPath       string
		wantHSTSHeader string
		wantResp       string
	}{
		{
			desc:           "Succcess for HTTP1 client with HSTS",
			httpsVersion:   1,
			certPath:       platform.GetFilePath(platform.ServerCert),
			wantHSTSHeader: "max-age=31536000; includeSubdomains",
			wantResp:       `simple get message`,
		},
		{
			desc:           "Succcess for HTTP2 client with HSTS",
			httpsVersion:   2,
			certPath:       platform.GetFilePath(platform.ServerCert),
			wantHSTSHeader: "max-age=31536000; includeSubdomains",
			wantResp:       `simple get message`,
		},
	}

	for _, tc := range testData {
		url := fmt.Sprintf("https://localhost:%v/simpleget?key=api-key", s.Ports().ListenerPort)
		respHeader, respBody, err := client.DoHttpsGet(url, tc.httpsVersion, tc.certPath)

		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(string(respBody), tc.wantResp) {
			t.Errorf("Test desc (%v) expected: %s, got: %s", tc.desc, tc.wantResp, string(respBody))
		}

		if gotHeader := respHeader.Get("Strict-Transport-Security"); gotHeader != tc.wantHSTSHeader {
			t.Errorf("Test desc (%v) expected: %s, got: %s", tc.desc, tc.wantHSTSHeader, gotHeader)
		}
	}
}
