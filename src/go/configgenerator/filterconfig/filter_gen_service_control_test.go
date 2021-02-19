// Copyright 2019 Google LLC
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

package filterconfig

import (
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/jsonpb"

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

func TestServiceControl(t *testing.T) {
	fakeServiceConfig := &confpb.Service{
		Name: testProjectName,
		Apis: []*apipb.Api{
			{
				Name: testApiName,
				Methods: []*apipb.Method{
					{
						Name: "ListShelves",
					},
				},
			},
		},
		Control: &confpb.Control{
			Environment: util.StatPrefix,
		},
	}
	testData := []struct {
		desc                            string
		serviceControlCredentials       *options.IAMCredentialsOptions
		serviceAccountKey               string
		wantPartialServiceControlFilter string
	}{
		{
			desc: "get access token from imds",
			wantPartialServiceControlFilter: `
    "imdsToken": {
      "cluster": "metadata-cluster",
      "timeout": "30s",
      "uri": "http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
    },`,
		},
		{
			desc: "get access token from iam",
			serviceControlCredentials: &options.IAMCredentialsOptions{
				ServiceAccountEmail: "ServiceControl@iam.com",
				Delegates:           []string{"delegate_foo", "delegate_bar"},
			},
			wantPartialServiceControlFilter: `
    "iamToken": {
      "accessToken": {
        "remoteToken": {
          "cluster": "metadata-cluster",
          "timeout": "30s",
          "uri": "http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
        }
      },
      "delegates": [
        "delegate_foo",
        "delegate_bar"
      ],
      "iamUri": {
        "cluster": "iam-cluster",
        "timeout": "30s",
        "uri": "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/ServiceControl@iam.com:generateAccessToken"
      },
      "serviceAccountEmail": "ServiceControl@iam.com"
    },`,
		},
		{
			desc:              "get access token from the token agent server",
			serviceAccountKey: "this-is-sa-cred",
			wantPartialServiceControlFilter: `
    "imdsToken": {
      "cluster": "token-agent-cluster",
      "timeout": "30s",
      "uri": "http://127.0.0.1:8791/local/access_token"
    },`,
		},
	}
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {

			opts := options.DefaultConfigGeneratorOptions()
			opts.ServiceControlCredentials = tc.serviceControlCredentials
			opts.ServiceAccountKey = tc.serviceAccountKey

			fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(fakeServiceConfig, testConfigID, opts)
			if err != nil {
				t.Error(err)
			}

			marshaler := &jsonpb.Marshaler{}
			filter, _, err := scFilterGenFunc(fakeServiceInfo)
			if err != nil {
				t.Fatal(err)
			}

			gotFilter, err := marshaler.MarshalToString(filter)
			if err != nil {
				t.Fatal(err)
			}

			if err := util.JsonContains(gotFilter, tc.wantPartialServiceControlFilter); err != nil {
				t.Errorf("makeServiceControlFilter failed,\n%v", err)
			}
		})
	}
}
