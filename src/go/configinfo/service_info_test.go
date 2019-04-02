// Copyright 2019 Google Cloud Platform Proxy Authors
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

package configinfo

import (
	"testing"

	"google.golang.org/genproto/protobuf/api"

	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

var (
	testProjectName = "bookstore.endpoints.project123.cloud.goog"
	testApiName     = "endpoints.examples.bookstore.Bookstore"
	testConfigID    = "2019-03-02r0"
)

func TestGetEndpointAllowCorsFlag(t *testing.T) {
	testData := []struct {
		desc                string
		fakeServiceConfig   *conf.Service
		wantedAllowCorsFlag bool
	}{
		{
			desc: "Return true for endpoint name matching service name",
			fakeServiceConfig: &conf.Service{
				Name: testProjectName,
				Apis: []*api.Api{
					{
						Name: testApiName,
					},
				},
				Endpoints: []*conf.Endpoint{
					{
						Name:      testProjectName,
						AllowCors: true,
					},
				},
			},
			wantedAllowCorsFlag: true,
		},
		{
			desc: "Return false for not setting allow_cors",
			fakeServiceConfig: &conf.Service{
				Name: testProjectName,
				Apis: []*api.Api{
					{
						Name: testApiName,
					},
				},
				Endpoints: []*conf.Endpoint{
					{
						Name: testProjectName,
					},
				},
			},
			wantedAllowCorsFlag: false,
		},
		{
			desc: "Return false for endpoint name not matching service name",
			fakeServiceConfig: &conf.Service{
				Name: testProjectName,
				Apis: []*api.Api{
					{
						Name: testApiName,
					},
				},
				Endpoints: []*conf.Endpoint{
					{
						Name:      "echo.endpoints.project123.cloud.goog",
						AllowCors: true,
					},
				},
			},
			wantedAllowCorsFlag: false,
		},
		{
			desc: "Return false for empty endpoint field",
			fakeServiceConfig: &conf.Service{
				Name: testProjectName,
				Apis: []*api.Api{
					{
						Name: testApiName,
					},
				},
			},
			wantedAllowCorsFlag: false,
		},
	}

	for i, tc := range testData {
		serviceInfo, err := NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID)
		if err != nil {
			t.Fatal(err)
		}

		allowCorsFlag := serviceInfo.GetEndpointAllowCorsFlag()
		if allowCorsFlag != tc.wantedAllowCorsFlag {
			t.Errorf("Test Desc(%d): %s, allow CORS flag got: %v, want: %v", i, tc.desc, allowCorsFlag, tc.wantedAllowCorsFlag)
		}
	}
}
