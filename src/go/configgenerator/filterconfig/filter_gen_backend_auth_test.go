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
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/jsonpb"

	commonpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v10/http/common"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

func TestBackendAuthFilter(t *testing.T) {
	testdata := []struct {
		desc                  string
		iamServiceAccount     string
		fakeServiceConfig     *confpb.Service
		delegates             []string
		depErrorBehavior      string
		wantBackendAuthFilter string
		wantError             string
	}{
		{
			desc: "Success, generate backend auth filter in general",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "testapipb",
						Methods: []*apipb.Method{
							{
								Name: "foo",
							},
							{
								Name: "bar",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Selector:        "testapipb.foo",
							Address:         "https://testapipb.com/foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "foo.com",
							},
						},
						{
							Selector:        "testapipb.bar",
							Address:         "https://testapipb.com/foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "bar.com",
							},
						},
					},
				},
			},
			depErrorBehavior: commonpb.DependencyErrorBehavior_BLOCK_INIT_ON_ANY_ERROR.String(),
			wantBackendAuthFilter: `
{
   "name":"com.google.espv2.filters.http.backend_auth",
   "typedConfig":{
      "@type":"type.googleapis.com/espv2.api.envoy.v10.http.backend_auth.FilterConfig",
      "depErrorBehavior":"BLOCK_INIT_ON_ANY_ERROR",
      "imdsToken":{
          "cluster":"metadata-cluster",
          "timeout":"30s",
          "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/identity"
      },
      "jwtAudienceList":["bar.com","foo.com"]
   }
}
`,
		},
		{
			desc: "Success, generate backend auth filter with allow Cors",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Endpoints: []*confpb.Endpoint{
					{
						Name:      testProjectName,
						AllowCors: true,
					},
				},
				Apis: []*apipb.Api{
					{
						Name: "get_testapi",
						Methods: []*apipb.Method{
							{
								Name: "foo",
							},
							{
								Name: "bar",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "get_testapi.foo",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/foo",
							},
						},
						{
							Selector: "get_testapi.bar",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/bar",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Selector:        "get_testapi.foo",
							Address:         "https://testapipb.com/foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "foo.com",
							},
						},
						{
							Selector:        "get_testapi.bar",
							Address:         "https://testapipb.com/bar",
							PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "bar.com",
							},
						},
					},
				},
			},
			depErrorBehavior: commonpb.DependencyErrorBehavior_BLOCK_INIT_ON_ANY_ERROR.String(),
			wantBackendAuthFilter: `{
        "name":"com.google.espv2.filters.http.backend_auth",
        "typedConfig":{
          "@type":"type.googleapis.com/espv2.api.envoy.v10.http.backend_auth.FilterConfig",
          "depErrorBehavior":"BLOCK_INIT_ON_ANY_ERROR",
          "imdsToken":{
            "cluster":"metadata-cluster",
            "timeout":"30s",
            "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/identity"
          },
          "jwtAudienceList":["bar.com","foo.com"]
        }
      }`,
		},
		{
			desc:              "Success, set iamIdToken when iam service account is set",
			iamServiceAccount: "service-account@google.com",
			delegates:         []string{"delegate_foo", "delegate_bar", "delegate_baz"},
			depErrorBehavior:  commonpb.DependencyErrorBehavior_ALWAYS_INIT.String(),
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "testapipb",
						Methods: []*apipb.Method{
							{
								Name: "bar",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Selector:        "testapipb.bar",
							Address:         "https://testapipb.com/foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "bar.com",
							},
						},
					},
				},
			},
			wantBackendAuthFilter: `
{
   "name":"com.google.espv2.filters.http.backend_auth",
   "typedConfig":{
      "@type":"type.googleapis.com/espv2.api.envoy.v10.http.backend_auth.FilterConfig",
			"depErrorBehavior":"ALWAYS_INIT",
      "iamToken":{
         "accessToken":{
            "remoteToken":{
               "cluster":"metadata-cluster",
               "timeout":"30s",
               "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"
            }
         },
         "iamUri":{
            "cluster":"iam-cluster",
            "timeout":"30s",
            "uri":"https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/service-account@google.com:generateIdToken"
         },
         "delegates":["delegate_foo","delegate_bar","delegate_baz"],
         "serviceAccountEmail":"service-account@google.com"
      },
      "jwtAudienceList":["bar.com"]
   }
}
`,
		},
		{
			desc:             "Fail when invalid dependency error behavior is provided",
			depErrorBehavior: "UNKNOWN_ERROR_BEHAVIOR",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "testapipb",
						Methods: []*apipb.Method{
							{
								Name: "bar",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Selector:        "testapipb.bar",
							Address:         "https://testapipb.com/foo",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "bar.com",
							},
						},
					},
				},
			},
			wantError: `unknown value for DependencyErrorBehavior (UNKNOWN_ERROR_BEHAVIOR), accepted values are: ["ALWAYS_INIT" "BLOCK_INIT_ON_ANY_ERROR" "UNSPECIFIED"]`,
		},
	}

	for _, tc := range testdata {
		t.Run(tc.desc, func(t *testing.T) {
			opts := options.DefaultConfigGeneratorOptions()
			opts.BackendAddress = "grpc://127.0.0.1:80"
			opts.DependencyErrorBehavior = tc.depErrorBehavior
			if tc.iamServiceAccount != "" {
				opts.BackendAuthCredentials = &options.IAMCredentialsOptions{
					ServiceAccountEmail: tc.iamServiceAccount,
					TokenKind:           options.IDToken,
					Delegates:           tc.delegates,
				}
			}

			fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
			if err != nil {
				t.Fatal(err)
			}

			filterConfig, _, err := baFilterGenFunc(fakeServiceInfo)
			if err != nil {
				if tc.wantError == "" || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("exepected err (%v), got err (%v)", tc.wantError, err)
				}
				return
			}

			marshaler := &jsonpb.Marshaler{}
			gotFilter, err := marshaler.MarshalToString(filterConfig)
			if err != nil {
				t.Fatal(err)
			}
			if err := util.JsonEqual(tc.wantBackendAuthFilter, gotFilter); err != nil {
				t.Errorf("makeBackendAuthFilter failed,\n %v", err)
			}
		})
	}
}
