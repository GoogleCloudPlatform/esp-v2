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

package filtergen_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	commonpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/common"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestNewBackendAuthFilterGensFromOPConfig_GenConfig(t *testing.T) {
	testdata := []SuccessOPTestCase{
		{
			Desc: "Generate with defaults",
			ServiceConfigIn: &confpb.Service{
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
			WantFilterConfigs: []string{
				`
{
   "name":"com.google.espv2.filters.http.backend_auth",
   "typedConfig":{
      "@type":"type.googleapis.com/espv2.api.envoy.v12.http.backend_auth.FilterConfig",
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
		},
		{
			Desc: "Generate with customized fail open error",
			ServiceConfigIn: &confpb.Service{
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
			OptsIn: options.ConfigGeneratorOptions{
				DependencyErrorBehavior: commonpb.DependencyErrorBehavior_ALWAYS_INIT.String(),
			},
			WantFilterConfigs: []string{
				`
{
   "name":"com.google.espv2.filters.http.backend_auth",
   "typedConfig":{
      "@type":"type.googleapis.com/espv2.api.envoy.v12.http.backend_auth.FilterConfig",
      "depErrorBehavior":"ALWAYS_INIT",
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
		},
		{
			Desc: "Set iamIdToken when iam service account is set",
			ServiceConfigIn: &confpb.Service{
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
			OptsIn: options.ConfigGeneratorOptions{
				CommonOptions: options.CommonOptions{
					BackendAuthCredentials: &options.IAMCredentialsOptions{
						ServiceAccountEmail: "service-account@google.com",
						TokenKind:           options.IDToken,
						Delegates:           []string{"delegate_foo", "delegate_bar", "delegate_baz"},
					},
				},
			},
			WantFilterConfigs: []string{
				`
{
   "name":"com.google.espv2.filters.http.backend_auth",
   "typedConfig":{
      "@type":"type.googleapis.com/espv2.api.envoy.v12.http.backend_auth.FilterConfig",
			"depErrorBehavior":"BLOCK_INIT_ON_ANY_ERROR",
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
		},
		{
			Desc: "DisableAuth is set to true, no filter config generated",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:        "grpc://abc.com/api",
							Selector:       "abc.com.api",
							Deadline:       10.5,
							Authentication: &confpb.BackendRule_DisableAuth{DisableAuth: true},
						},
					},
				},
			},
			WantFilterConfigs: nil,
		},
		{
			Desc: "DisableAuth is set to false, jwt aud is generated (with transformation)",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:        "grpc://abc.com/api",
							Selector:       "abc.com.api",
							Deadline:       10.5,
							Authentication: &confpb.BackendRule_DisableAuth{DisableAuth: false},
						},
					},
				},
			},
			WantFilterConfigs: []string{
				`
{
   "name":"com.google.espv2.filters.http.backend_auth",
   "typedConfig":{
      "@type":"type.googleapis.com/espv2.api.envoy.v12.http.backend_auth.FilterConfig",
      "depErrorBehavior":"BLOCK_INIT_ON_ANY_ERROR",
      "imdsToken":{
          "cluster":"metadata-cluster",
          "timeout":"30s",
          "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/identity"
      },
      "jwtAudienceList":["http://abc.com"]
   }
}
`,
			},
		},
		{
			Desc: "DisableAuth is empty, jwt aud is generated (with transformation)",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://abc.com/api",
							Selector: "abc.com.api",
							Deadline: 10.5,
						},
					},
				},
			},
			WantFilterConfigs: []string{
				`
{
   "name":"com.google.espv2.filters.http.backend_auth",
   "typedConfig":{
      "@type":"type.googleapis.com/espv2.api.envoy.v12.http.backend_auth.FilterConfig",
      "depErrorBehavior":"BLOCK_INIT_ON_ANY_ERROR",
      "imdsToken":{
          "cluster":"metadata-cluster",
          "timeout":"30s",
          "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/identity"
      },
      "jwtAudienceList":["http://abc.com"]
   }
}
`,
			},
		},
		{
			Desc: "DisableAuth is empty, jwt aud is generated (with HTTPS transformation)",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpcs://abc.com/api",
							Selector: "abc.com.api",
							Deadline: 10.5,
						},
					},
				},
			},
			WantFilterConfigs: []string{
				`
{
   "name":"com.google.espv2.filters.http.backend_auth",
   "typedConfig":{
      "@type":"type.googleapis.com/espv2.api.envoy.v12.http.backend_auth.FilterConfig",
      "depErrorBehavior":"BLOCK_INIT_ON_ANY_ERROR",
      "imdsToken":{
          "cluster":"metadata-cluster",
          "timeout":"30s",
          "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/identity"
      },
      "jwtAudienceList":["https://abc.com"]
   }
}
`,
			},
		},
		{
			Desc: "JwtAudience is used",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:        "grpc://abc.com/api",
							Selector:       "abc.com.api",
							Deadline:       10.5,
							Authentication: &confpb.BackendRule_JwtAudience{JwtAudience: "audience-foo"},
						},
					},
				},
			},
			WantFilterConfigs: []string{
				`
{
   "name":"com.google.espv2.filters.http.backend_auth",
   "typedConfig":{
      "@type":"type.googleapis.com/espv2.api.envoy.v12.http.backend_auth.FilterConfig",
      "depErrorBehavior":"BLOCK_INIT_ON_ANY_ERROR",
      "imdsToken":{
          "cluster":"metadata-cluster",
          "timeout":"30s",
          "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/identity"
      },
      "jwtAudienceList":["audience-foo"]
   }
}
`,
			},
		},
		{
			Desc: "JwtAudience is set, but non-GCP runtime disables backend auth",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:        "grpc://abc.com/api",
							Selector:       "abc.com.api",
							Deadline:       10.5,
							Authentication: &confpb.BackendRule_JwtAudience{JwtAudience: "audience-foo"},
						},
					},
				},
			},
			OptsIn: options.ConfigGeneratorOptions{
				CommonOptions: options.CommonOptions{
					NonGCP: true,
				},
			},
			WantFilterConfigs: nil,
		},
		{
			Desc: "Mix all Authentication cases",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:        "grpc://abc.com/api",
							Selector:       "abc.com.api",
							Deadline:       10.5,
							Authentication: &confpb.BackendRule_JwtAudience{JwtAudience: "audience-foo"},
						},
						{
							Address:        "grpc://def.com/api",
							Selector:       "def.com.api",
							Deadline:       10.5,
							Authentication: &confpb.BackendRule_JwtAudience{JwtAudience: "audience-bar"},
						},
						{
							Address:        "grpc://ghi.com/api",
							Selector:       "ghi.com.api",
							Deadline:       10.5,
							Authentication: &confpb.BackendRule_DisableAuth{DisableAuth: false},
						},
						{
							Address:        "grpc://jkl.com/api",
							Selector:       "jkl.com.api",
							Deadline:       10.5,
							Authentication: &confpb.BackendRule_DisableAuth{DisableAuth: true},
						},
						{
							Address:  "grpcs://mno.com/api",
							Selector: "mno.com.api",
							Deadline: 10.5,
						},
					},
				},
			},
			WantFilterConfigs: []string{
				`
{
   "name":"com.google.espv2.filters.http.backend_auth",
   "typedConfig":{
      "@type":"type.googleapis.com/espv2.api.envoy.v12.http.backend_auth.FilterConfig",
      "depErrorBehavior":"BLOCK_INIT_ON_ANY_ERROR",
      "imdsToken":{
          "cluster":"metadata-cluster",
          "timeout":"30s",
          "uri":"http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/identity"
      },
      "jwtAudienceList":["audience-bar", "audience-foo", "http://ghi.com", "https://mno.com"]
   }
}
`,
			},
		},
		{
			Desc: "Skip for discovery apis",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:        "grpc://abc.com/api",
							Selector:       "google.discovery",
							Deadline:       10.5,
							Authentication: &confpb.BackendRule_JwtAudience{JwtAudience: "audience-foo"},
						},
					},
				},
			},
			WantFilterConfigs: nil,
		},
	}

	for _, tc := range testdata {
		tc.RunTest(t, filtergen.NewBackendAuthFilterGensFromOPConfig)
	}
}

func TestNewBackendAuthFilterGensFromOPConfig_BadInputFactory(t *testing.T) {
	testdata := []FactoryErrorOPTestCase{
		{
			// Should never happen in theory, as API compiler ensures valid address.
			Desc: "Fail when backend address is invalid",
			ServiceConfigIn: &confpb.Service{
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Selector:        "testapipb.bar",
							Address:         "https://test^^port:80:googleapis.com/#&^#",
							PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
						},
					},
				},
			},
			WantFactoryError: `fail to parse JWT audience for backend rule`,
		},
	}

	for _, tc := range testdata {
		tc.RunTest(t, filtergen.NewBackendAuthFilterGensFromOPConfig)
	}
}

func TestNewBackendAuthFilterGensFromOPConfig_BadInputFilterGen(t *testing.T) {
	testdata := []GenConfigErrorOPTestCase{
		{
			Desc: "Fail when invalid dependency error behavior is provided",
			ServiceConfigIn: &confpb.Service{
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
			OptsIn: options.ConfigGeneratorOptions{
				DependencyErrorBehavior: "UNKNOWN_ERROR_BEHAVIOR",
			},
			WantGenErrors: []string{`unknown value for DependencyErrorBehavior (UNKNOWN_ERROR_BEHAVIOR), accepted values are: ["ALWAYS_INIT" "BLOCK_INIT_ON_ANY_ERROR" "UNSPECIFIED"]`},
		},
	}

	for _, tc := range testdata {
		tc.RunTest(t, filtergen.NewBackendAuthFilterGensFromOPConfig)
	}
}
