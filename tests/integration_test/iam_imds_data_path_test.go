package integration_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
	"github.com/golang/glog"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
)

func TestIamImdsDataPath(t *testing.T) {
	t.Parallel()
	testData := []struct {
		desc         string
		useIam       bool
		fakeIamDown  bool
		fakeImdsDown bool
		wantResp     string
		wantErr      string
	}{
		{
			desc:     "Backend auth with IMDS works when everything is up",
			wantResp: `{"Authorization": "Bearer ya29.new", "RequestURI": "/bearertoken/constant?foo=42"}`,
		},
		{
			desc:        "Backend auth with IMDS works, even when IAM is down",
			fakeIamDown: true,
			wantResp:    `{"Authorization": "Bearer ya29.new", "RequestURI": "/bearertoken/constant?foo=42"}`,
		},
		{
			desc:         "Backend auth with IMDS fails (envoy doesn't start) when IMDS is down",
			fakeImdsDown: true,
			wantErr:      `connect: connection refused`,
		},
		{
			desc:     "Backend auth with IAM works when everything is up",
			useIam:   true,
			wantResp: `{"Authorization": "Bearer default-test-id-token", "RequestURI": "/bearertoken/constant?foo=42"}`,
		},
		{
			desc:        "Backend auth with IAM fails (envoy doesn't start) when IAM is down",
			useIam:      true,
			fakeIamDown: true,
			wantErr:     `connect: connection refused`,
		},
		{
			desc:         "Backend auth with IAM fails (envoy doesn't start) when IMDS is down",
			useIam:       true,
			fakeImdsDown: true,
			wantErr:      `connect: connection refused`,
		},
	}

	for _, tc := range testData {

		// Place in closure to allow deferring in loop.
		func() {

			// By default, IMDS will be used for service control and backend auth.
			s := env.NewTestEnv(comp.TestIamImdsDataPath, platform.EchoRemote)

			if tc.useIam {
				// Use IAM for service control and backend auth.
				serviceAccount := "fakeServiceAccount@google.com"
				s.SetBackendAuthIamServiceAccount(serviceAccount)
				s.SetIamResps(map[string]string{}, 1, 0)
			}

			if tc.fakeImdsDown {
				// Fake IMDS will respond with failures.
				s.OverrideMockMetadata(map[string]string{}, 100)
			}

			if tc.fakeIamDown {
				// Fake IAM will respond with failures.
				s.SetIamResps(map[string]string{}, 100, 0)
			}

			if tc.wantErr != "" {
				// Skip health checks since we expect an error.
				s.SkipHealthChecks()
				glog.Infof("Sleeping to ensure Envoy is starting")
				time.Sleep(7 * time.Second)
			}

			defer s.TearDown(t)
			if err := s.Setup(utils.CommonArgs()); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/bearertoken/constant/42")
			resp, err := client.DoWithHeaders(url, "GET", "", nil)

			if tc.wantErr != "" {
				if err == nil {
					t.Errorf("Test Desc(%s): expected err, got none", tc.desc)
					return
				}

				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("Test Desc(%s): want err: %s, got err: %s", tc.desc, tc.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Test Desc(%s): %v", tc.desc, err)
					return
				}

				gotResp := string(resp)
				if err := util.JsonEqual(tc.wantResp, gotResp); err != nil {
					t.Errorf("Test Desc(%s) failed, \n %v", tc.desc, err)
				}
			}
		}()
	}
}
