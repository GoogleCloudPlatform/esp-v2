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

package report_gcp_attributes_test

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/api-proxy/src/go/util"
	"github.com/GoogleCloudPlatform/api-proxy/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/api-proxy/tests/env"
	"github.com/GoogleCloudPlatform/api-proxy/tests/utils"

	comp "github.com/GoogleCloudPlatform/api-proxy/tests/env/components"
)

const (
	platformKey = "servicecontrol.googleapis.com/platform"
	locationKey = "cloud.googleapis.com/location"
)

func TestReportGCPAttributes(t *testing.T) {

	testdata := []struct {
		desc                 string
		mockMetadataOverride map[string]string
		wantPlatform         string
		wantLocation         string
	}{
		{
			desc: "Valid Zone",
			mockMetadataOverride: map[string]string{
				util.ZoneSuffix: "projects/4242424242/zones/us-west-1b",
			},
			wantLocation: "us-west-1b",
			wantPlatform: "GCE",
		},
		{
			desc: "Invalid Zone - without '/'",
			mockMetadataOverride: map[string]string{
				util.ZoneSuffix: "some-invalid-zone",
			},
			wantLocation: "",
			wantPlatform: "GCE",
		},
		{
			desc: "Invalid Zone - ends with '/'",
			mockMetadataOverride: map[string]string{
				util.ZoneSuffix: "project/123123/",
			},
			wantLocation: "",
			wantPlatform: "GCE",
		},
		{
			desc: "Platform - GAE FLEX",
			mockMetadataOverride: map[string]string{
				util.GAEServerSoftwareSuffix: "gae",
			},
			wantLocation: "test-zone",
			wantPlatform: "GAE Flex",
		},
		{
			desc: "Platform - GKE",
			mockMetadataOverride: map[string]string{
				util.KubeEnvSuffix: "kube-env",
			},
			wantLocation: "test-zone",
			wantPlatform: "GKE",
		},
		// If it is neither GAE nor GKE it should be GCE.
		{
			desc:                 "Platform- GCE",
			mockMetadataOverride: map[string]string{},
			wantLocation:         "test-zone",
			wantPlatform:         "GCE",
		},
		{
			desc: "Platform and Zone",
			mockMetadataOverride: map[string]string{
				util.ZoneSuffix:              "projects/4242424242/zones/us-west-1b",
				util.GAEServerSoftwareSuffix: "gae",
			},
			wantLocation: "us-west-1b",
			wantPlatform: "GAE Flex",
		},
	}

	configID := "test-config-id"
	args := []string{"--service_config_id=" + configID,
		"--backend_protocol=http1", "--rollout_strategy=fixed"}

	for _, tc := range testdata {
		func() {
			s := env.NewTestEnv(comp.TestReportGCPAttributes, "echo")
			s.OverrideMockMetadata(tc.mockMetadataOverride)

			defer s.TearDown()
			if err := s.Setup(args); err != nil {
				t.Fatalf("Test(%s): fail to setup test env, %v", tc.desc, err)
			}

			url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo/nokey")
			_, err := client.DoPost(url, "hello")
			if err != nil {
				t.Fatal(err)
			}

			scRequests, err := s.ServiceControlServer.GetRequests(1)
			if err != nil {
				t.Fatalf("Test(%s): GetRequests returns error: %v", tc.desc, err)
			}

			if scRequests[0].ReqType != comp.REPORT_REQUEST {
				t.Fatalf("Test(%s): service control request: should be Report", tc.desc)
			}

			gotRequest, err := utils.UnmarshalReportRequest(scRequests[0].ReqBody)
			if err != nil {
				t.Fatalf("Test(%s): %v", tc.desc, err)
			}

			if len(gotRequest.GetOperations()) != 1 {
				t.Fatalf("Test(%s): service control request: number of operations should be 1", tc.desc)
			}

			labels := gotRequest.GetOperations()[0].GetLabels()

			if gotPlatform := labels[platformKey]; gotPlatform != tc.wantPlatform {
				t.Errorf("Test(%s): Platform does not match got: %v: want: %v", tc.desc, gotPlatform, tc.wantPlatform)
			}

			if gotLocation := labels[locationKey]; gotLocation != tc.wantLocation {
				t.Errorf("Test(%s): Location does not match got: %v: want: %v", tc.desc, gotLocation, tc.wantLocation)
			}
		}()
	}
}
