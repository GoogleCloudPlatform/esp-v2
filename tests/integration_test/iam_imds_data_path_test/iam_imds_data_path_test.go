package iam_imds_data_path_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
)

var testDataPathArgs = []string{
	"--service_config_id=test-config-id",
	"--rollout_strategy=fixed",
	"--backend_dns_lookup_family=v4only",
	"--suppress_envoy_headers",
}

func TestDataPathImdsSuccessWhenIamDown(t *testing.T) {
	s := env.NewTestEnv(comp.TestDataPathImdsSuccessWhenIamDown, platform.EchoRemote)

	// Simulate Iam failures.
	s.SetIamResps(map[string]string{}, 100)

	defer s.TearDown()
	if err := s.Setup(testDataPathArgs); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc     string
		method   string
		path     string
		message  string
		wantResp string
	}{
		{
			desc:     "Backend auth with IMDS works, even when IAM is down",
			method:   "GET",
			path:     "/bearertoken/constant/42",
			wantResp: `{"Authorization": "Bearer ya29.new", "RequestURI": "/bearertoken/constant?foo=42"}`,
		},
	}

	for _, tc := range testData {
		url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, tc.path)
		resp, err := client.DoWithHeaders(url, tc.method, tc.message, nil)

		if err != nil {
			t.Fatalf("Test Desc(%s): %v", tc.desc, err)
		}

		gotResp := string(resp)
		if !utils.JsonEqual(gotResp, tc.wantResp) {
			t.Errorf("Test Desc(%s): want: %s, got: %s", tc.desc, tc.wantResp, gotResp)
		}
	}
}

func TestDataPathIamFailWhenImdsDown(t *testing.T) {
	s := env.NewTestEnv(comp.TestDataPathIamFailWhenImdsDown, platform.EchoRemote)

	// Use IAM instead of IMDS.
	serviceAccount := "fakeServiceAccount@google.com"
	s.SetBackendAuthIamServiceAccount(serviceAccount)

	// Simulate IMDS failures.
	s.OverrideMockMetadata(map[string]string{}, 100)

	// Skip health checks, we expect these to fail.
	s.SkipHealthChecks()

	defer s.TearDown()
	if err := s.Setup(testDataPathArgs); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	testData := []struct {
		desc    string
		method  string
		path    string
		message string
		wantErr string
	}{
		{
			desc:    "If IMDS is down, then Envoy will fail to start, even when using IAM",
			method:  "GET",
			path:    "/bearertoken/constant/42",
			wantErr: `connect: connection refused`,
		},
	}

	for _, tc := range testData {
		url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, tc.path)
		_, err := client.DoWithHeaders(url, tc.method, tc.message, nil)

		if err == nil {
			t.Errorf("Test Desc(%s): expected err, got none", tc.desc)
			continue
		}

		if !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("Test Desc(%s): want err: %s, got err: %s", tc.desc, tc.wantErr, err)
		}
	}
}
