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
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
)

const (
	platformKey = "servicecontrol.googleapis.com/platform"
	locationKey = "cloud.googleapis.com/location"
)

func validateReportGcpAttributes(s *env.TestEnv, wantPlatform string, wantLocation string) error {
	scRequests, err := s.ServiceControlServer.GetRequests(1)
	if err != nil {
		return fmt.Errorf("GetRequests returns error: %v", err)
	}

	if scRequests[0].ReqType != utils.ReportRequest {
		return fmt.Errorf("service control request: should be Report")
	}

	gotRequest, err := utils.UnmarshalReportRequest(scRequests[0].ReqBody)
	if err != nil {
		return err
	}

	if len(gotRequest.GetOperations()) != 1 {
		return fmt.Errorf("service control request: number of operations should be 1")
	}

	labels := gotRequest.GetOperations()[0].GetLabels()

	if gotPlatform := labels[platformKey]; gotPlatform != wantPlatform {
		return fmt.Errorf("Platform does not match got: %v: want: %v", gotPlatform, wantPlatform)
	}

	if gotLocation := labels[locationKey]; gotLocation != wantLocation {
		return fmt.Errorf("Location does not match got: %v: want: %v", gotLocation, wantLocation)
	}

	return nil
}

func TestReportGCPAttributes(t *testing.T) {
	t.Parallel()

	testdata := []struct {
		desc                 string
		mockMetadataOverride map[string]string
		platformOverride     string
		wantPlatform         string
		wantLocation         string
	}{
		{
			desc: "Valid Zone",
			mockMetadataOverride: map[string]string{
				util.ZonePath: "projects/4242424242/zones/us-west-1b",
			},
			wantLocation: "us-west-1b",
			wantPlatform: "GCE(ESPv2)",
		},
		{
			desc: "Invalid Zone - without '/'",
			mockMetadataOverride: map[string]string{
				util.ZonePath: "some-invalid-zone",
			},
			wantLocation: "global",
			wantPlatform: "GCE(ESPv2)",
		},
		{
			desc: "Invalid Zone - ends with '/'",
			mockMetadataOverride: map[string]string{
				util.ZonePath: "project/123123/",
			},
			wantLocation: "global",
			wantPlatform: "GCE(ESPv2)",
		},
		{
			desc: "Platform - GAE FLEX",
			mockMetadataOverride: map[string]string{
				util.GAEServerSoftwarePath: "gae",
			},
			wantLocation: "test-zone",
			wantPlatform: "GAE_FLEX(ESPv2)",
		},
		{
			desc: "Platform - GKE",
			mockMetadataOverride: map[string]string{
				util.KubeEnvPath: "kube-env",
			},
			wantLocation: "test-zone",
			wantPlatform: "GKE(ESPv2)",
		},
		// If it is neither GAE nor GKE it should be GCE.
		{
			desc:                 "Platform- GCE",
			mockMetadataOverride: map[string]string{},
			wantLocation:         "test-zone",
			wantPlatform:         "GCE(ESPv2)",
		},
		{
			desc: "Platform and Zone",
			mockMetadataOverride: map[string]string{
				util.ZonePath:              "projects/4242424242/zones/us-west-1b",
				util.GAEServerSoftwarePath: "gae",
			},
			wantLocation: "us-west-1b",
			wantPlatform: "GAE_FLEX(ESPv2)",
		},
		{
			desc:                 "Override Platform",
			mockMetadataOverride: map[string]string{},
			platformOverride:     "Cloud Run",
			wantLocation:         "test-zone",
			wantPlatform:         "Cloud Run",
		},
		{
			desc: "Cloud Run uses regional location, zone is ignored",
			mockMetadataOverride: map[string]string{
				util.RegionPath: "projects/123/regions/test-region",
			},
			platformOverride: "Cloud Run(ESPv2)",
			wantLocation:     "test-region",
			wantPlatform:     "Cloud Run(ESPv2)",
		},
	}

	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--rollout_strategy=fixed"}

	for _, tc := range testdata {
		if tc.platformOverride != "" {
			args = append(args, fmt.Sprintf("--compute_platform_override=%v", tc.platformOverride))
		}
		func() {
			s := env.NewTestEnv(comp.TestReportGCPAttributes, platform.EchoSidecar)
			s.OverrideMockMetadata(tc.mockMetadataOverride, 0)

			defer s.TearDown(t)
			if err := s.Setup(args); err != nil {
				t.Fatalf("Test(%s): fail to setup test env, %v", tc.desc, err)
			}

			url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo/nokey")
			_, err := client.DoPost(url, "hello")
			if err != nil {
				t.Fatal(err)
			}

			err = validateReportGcpAttributes(s, tc.wantPlatform, tc.wantLocation)
			if err != nil {
				t.Errorf("Test (%v) failed: %v", tc.desc, err)
			}
		}()
	}
}

// Regression test for https://github.com/GoogleCloudPlatform/esp-v2/issues/304.
func TestReportGCPAttributesPerPlatform(t *testing.T) {
	t.Parallel()

	customSa, err := utils.NewServiceAccountForTest()
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	defer customSa.MockTokenServer.Close()

	testdata := []struct {
		desc         string
		confArgs     []string
		wantPlatform string
		wantLocation string
	}{
		{
			desc:         "Uses IMDS for GCP deployment",
			confArgs:     utils.CommonArgs(),
			wantLocation: "test-zone",
			wantPlatform: "GCE(ESPv2)",
		},
		{
			desc: "Uses IMDS for GCP deployment with custom service account",
			confArgs: append([]string{
				"--service_account_key=" + customSa.FileName,
			}, utils.CommonArgs()...),
			wantLocation: "test-zone",
			wantPlatform: "GCE(ESPv2)",
		},
		{
			desc: "Uses built-in defaults for non-GCP deployment",
			confArgs: append([]string{
				"--non_gcp",
				"--service_account_key=" + customSa.FileName,
			}, utils.CommonArgs()...),
			wantLocation: "global",
			wantPlatform: "UNKNOWN(ESPv2)",
		},
	}

	for _, tc := range testdata {
		func() {
			s := env.NewTestEnv(comp.TestReportGCPAttributesPerPlatform, platform.EchoSidecar)

			defer s.TearDown(t)
			if err := s.Setup(tc.confArgs); err != nil {
				t.Fatalf("Test(%s): fail to setup test env, %v", tc.desc, err)
			}

			url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo/nokey")
			_, err := client.DoPost(url, "hello")
			if err != nil {
				t.Fatal(err)
			}

			err = validateReportGcpAttributes(s, tc.wantPlatform, tc.wantLocation)
			if err != nil {
				t.Errorf("Test (%v) failed: %v", tc.desc, err)
			}
		}()
	}
}
