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
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"

	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
)

func TestBackendAuthDisableAuth(t *testing.T) {
	t.Parallel()

	s := env.NewTestEnv(comp.TestBackendAuthDisableAuth, platform.EchoRemote)
	s.OverrideMockMetadata(
		map[string]string{
			util.IdentityTokenSuffix + "?format=standard&audience=https://localhost/bearertoken/constant": "ya29.JwtAudienceSet",
			util.IdentityTokenSuffix + "?format=standard&audience=https://localhost":                      "ya29.DefaultAuth",
		}, 0)

	defer s.TearDown()
	if err := s.Setup(utils.CommonArgs()); err != nil {
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
			desc:     "Authentication is set with JwtAudience",
			method:   "GET",
			path:     "/bearertoken/constant/0",
			wantResp: `{"Authorization": "Bearer ya29.JwtAudienceSet", "RequestURI": "/bearertoken/constant?foo=0"}`,
		},
		{
			desc:     "Authentication is set with Disable as True so backend auth is disabled",
			method:   "GET",
			path:     "/disableauthsettotrue/constant/disableauthsettotrue",
			wantResp: `{"Authorization": "", "RequestURI": "/bearertoken/constant?foo=disableauthsettotrue"}`,
		},
		{
			desc:     "Authentication is set with Disable as False so JwtAudience is set with the backend address",
			method:   "GET",
			path:     "/disableauthsettofalse/constant/disableauthsettofalse",
			wantResp: `{"Authorization": "Bearer ya29.DefaultAuth", "RequestURI": "/bearertoken/constant?foo=disableauthsettofalse"}`,
		},
		{
			desc:     "Authentication is not set so JwtAudience is set with the backend address",
			method:   "GET",
			path:     "/authenticationnotset/constant/authenticationnotset",
			wantResp: `{"Authorization": "Bearer ya29.DefaultAuth", "RequestURI": "/bearertoken/constant?foo=authenticationnotset"}`,
		},
	}

	for _, tc := range testData {
		url := fmt.Sprintf("http://localhost:%v%v", s.Ports().ListenerPort, tc.path)
		resp, err := client.DoWithHeaders(url, tc.method, tc.message, nil)

		if err != nil {
			t.Fatalf("Test Desc(%s): %v", tc.desc, err)
		}

		gotResp := string(resp)
		if err := util.JsonEqual(tc.wantResp, gotResp); err != nil {
			t.Errorf("Test Desc(%s) failed, \n %v", tc.desc, err)
		}
	}
}
