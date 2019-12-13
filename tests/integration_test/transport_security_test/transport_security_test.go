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

package transport_security_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
)

func TestServiceManagementWithTLS(t *testing.T) {
	args := []string{
		"--service_config_id=test-config-id",
		"--backend_protocol=http1",
		"--rollout_strategy=fixed",
		"--suppress_envoy_headers",
	}

	testData := []struct {
		desc         string
		certPath     string
		wantResp     string
		wantSetupErr string
	}{
		{
			desc:     "call ServiceManagement HTTPS server succeed with same cert",
			certPath: platform.GetFilePath(platform.ServerCert),
			wantResp: `{"message":"hello"}`,
		},
		{
			desc:         "call ServiceManagement HTTPS server failed with different cert",
			certPath:     platform.GetFilePath(platform.ProxyCert),
			wantSetupErr: "connection refused",
		},
	}

	for _, tc := range testData {
		s := env.NewTestEnv(comp.TestServiceManagementWithTLS, "echo")
		defer s.TearDown()
		serverCerts, err := comp.GenerateCert()
		if err != nil {
			t.Fatalf("fial to generate cert: %v", err)
		}

		s.MockServiceManagementServer.SetCert(serverCerts)
		defer s.TearDown()

		args = append(args, fmt.Sprintf("--root_certs_path=%s", tc.certPath))
		err = s.Setup(args)

		if tc.wantSetupErr != "" {
			if err == nil || !strings.Contains(err.Error(), tc.wantSetupErr) {
				t.Errorf("Test (%s): failed, want error: %v, got error: %v", tc.desc, tc.wantSetupErr, err)
			}
			continue
		}

		url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, "/echo?key=api-key")
		resp, err := client.DoPost(url, "hello")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(resp), tc.wantResp) {
			t.Errorf("expected: %s, got: %s", tc.wantResp, string(resp))
		}
	}
}
