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
)

func TestIamImdsDataPath(t *testing.T) {
	t.Parallel()
	testData := []struct {
		desc         string
		useIam       bool
		fakeIamDown  bool
		fakeImdsDown bool
		confArgs     []string
		wantResp     string
		wantErr      string
	}{
		{
			desc:     "Backend auth with IMDS works when everything is up",
			confArgs: utils.CommonArgs(),
			wantResp: `{"Authorization": "Bearer ya29.new", "RequestURI": "/bearertoken/constant?foo=42"}`,
		},
		{
			desc:        "Backend auth with IMDS works, even when IAM is down",
			fakeIamDown: true,
			confArgs:    utils.CommonArgs(),
			wantResp:    `{"Authorization": "Bearer ya29.new", "RequestURI": "/bearertoken/constant?foo=42"}`,
		},
		{
			desc:         "Backend auth with IMDS fails (envoy doesn't start) when IMDS is down",
			fakeImdsDown: true,
			confArgs:     utils.CommonArgs(),
			wantErr:      `connect: connection refused`,
		},
		{
			desc:         "Backend auth with IMDS fails when IMDS is down, but Envoy starts due to configured error behavior",
			fakeImdsDown: true,
			confArgs: append([]string{
				"--dependency_error_behavior=ALWAYS_INIT",
			}, utils.CommonArgs()...),
			wantErr: `{"code":500,"message":"Token not found for audience: https://localhost/bearertoken/constant"}`,
		},
		{
			desc:     "Backend auth with IAM works when everything is up",
			useIam:   true,
			confArgs: utils.CommonArgs(),
			wantResp: `{"Authorization": "Bearer default-test-id-token", "RequestURI": "/bearertoken/constant?foo=42"}`,
		},
		{
			desc:        "Backend auth with IAM fails (envoy doesn't start) when IAM is down",
			useIam:      true,
			fakeIamDown: true,
			confArgs:    utils.CommonArgs(),
			wantErr:     `connect: connection refused`,
		},
		{
			desc:        "Backend auth with IAM fails when IAM is down, but Envoy starts due to configured error behavior",
			useIam:      true,
			fakeIamDown: true,
			confArgs: append([]string{
				"--dependency_error_behavior=ALWAYS_INIT",
			}, utils.CommonArgs()...),
			wantErr: `{"code":500,"message":"Token not found for audience: https://localhost/bearertoken/constant"}`,
		},
		{
			desc:         "Backend auth with IAM fails (envoy doesn't start) when IMDS is down",
			useIam:       true,
			fakeImdsDown: true,
			confArgs:     utils.CommonArgs(),
			wantErr:      `connect: connection refused`,
		},
		{
			desc:         "Backend auth with IAM fails when IMDS is down, but Envoy starts due to configured error behavior",
			useIam:       true,
			fakeImdsDown: true,
			confArgs: append([]string{
				"--dependency_error_behavior=ALWAYS_INIT",
			}, utils.CommonArgs()...),
			wantErr: `{"code":500,"message":"Token not found for audience: https://localhost/bearertoken/constant"}`,
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			// By default, IMDS will be used for service control and backend auth.
			s := env.NewTestEnv(platform.TestIamImdsDataPath, platform.EchoRemote)

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
				// When we expect a Envoy startup error, we must skip health checks. Otherwise they will prevent the test from running.
				s.SkipHealthChecks()
			}

			defer s.TearDown(t)
			if err := s.Setup(tc.confArgs); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			if tc.wantErr != "" {
				// When health checks are skipped (above), we need to manually sleep some time. Otherwise Envoy will not have time to try starting up.
				glog.Infof("Sleeping to ensure Envoy is starting")
				time.Sleep(10 * time.Second)
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
		})
	}
}
