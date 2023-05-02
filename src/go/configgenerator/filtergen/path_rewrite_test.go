// Copyright 2023 Google LLC
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
	"github.com/google/go-cmp/cmp"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestGetTranslationInfoFromOPConfig(t *testing.T) {
	testdata := []struct {
		desc            string
		serviceConfigIn *servicepb.Service
		optsIn          options.ConfigGeneratorOptions
		wantInfo        map[string]filtergen.TranslationInfo
	}{
		{
			desc: "Normal case where backend addresses are captured with minimal processing",
			serviceConfigIn: &servicepb.Service{
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Selector:        "selector_1",
							Address:         "https://my-backend.com:8080/api/v1",
							PathTranslation: servicepb.BackendRule_APPEND_PATH_TO_ADDRESS,
						},
						{
							Selector:        "selector_2",
							Address:         "https://my-backend.com:8080/api/v2/const-path",
							PathTranslation: servicepb.BackendRule_CONSTANT_ADDRESS,
						},
					},
				},
			},
			wantInfo: map[string]filtergen.TranslationInfo{
				"selector_1": {
					TranslationType: servicepb.BackendRule_APPEND_PATH_TO_ADDRESS,
					Path:            "/api/v1",
				},
				"selector_2": {
					TranslationType: servicepb.BackendRule_CONSTANT_ADDRESS,
					Path:            "/api/v2/const-path",
				},
			},
		},
		{
			desc: "Empty path is only captured for CONST address",
			serviceConfigIn: &servicepb.Service{
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Selector:        "selector_1",
							Address:         "https://my-backend.com:8080",
							PathTranslation: servicepb.BackendRule_APPEND_PATH_TO_ADDRESS,
						},
						{
							Selector:        "selector_2",
							Address:         "https://my-backend.com:8080",
							PathTranslation: servicepb.BackendRule_CONSTANT_ADDRESS,
						},
					},
				},
			},
			wantInfo: map[string]filtergen.TranslationInfo{
				"selector_2": {
					TranslationType: servicepb.BackendRule_CONSTANT_ADDRESS,
					Path:            "/",
				},
			},
		},
		{
			desc: "Discovery API not captured",
			serviceConfigIn: &servicepb.Service{
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Selector:        "google.discovery.GetDiscoveryRest",
							Address:         "https://my-backend.com:8080/api/v1",
							PathTranslation: servicepb.BackendRule_APPEND_PATH_TO_ADDRESS,
						},
						{
							Selector:        "selector_2",
							Address:         "https://my-backend.com:8080/api/v2/const-path",
							PathTranslation: servicepb.BackendRule_CONSTANT_ADDRESS,
						},
					},
				},
			},
			wantInfo: map[string]filtergen.TranslationInfo{
				"selector_2": {
					TranslationType: servicepb.BackendRule_CONSTANT_ADDRESS,
					Path:            "/api/v2/const-path",
				},
			},
		},
		{
			desc: "Non-OpenAPI HTTP backend is NOT captured",
			serviceConfigIn: &servicepb.Service{
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Selector:        "selector_1",
							Address:         "https://my-backend.com:8080/api/v1",
							PathTranslation: servicepb.BackendRule_APPEND_PATH_TO_ADDRESS,
							OverridesByRequestProtocol: map[string]*servicepb.BackendRule{
								"http": {
									Selector:        "selector_1",
									Address:         "https://my-backend.com:8080/api/v3/http-backend",
									PathTranslation: servicepb.BackendRule_APPEND_PATH_TO_ADDRESS,
								},
							},
						},
						{
							Selector:        "selector_2",
							Address:         "https://my-backend.com:8080/api/v2/const-path",
							PathTranslation: servicepb.BackendRule_CONSTANT_ADDRESS,
						},
					},
				},
			},
			wantInfo: map[string]filtergen.TranslationInfo{
				"selector_1": {
					TranslationType: servicepb.BackendRule_APPEND_PATH_TO_ADDRESS,
					Path:            "/api/v1",
				},

				"selector_2": {
					TranslationType: servicepb.BackendRule_CONSTANT_ADDRESS,
					Path:            "/api/v2/const-path",
				},
			},
		},
		{
			desc: "Rules with no address are not captured",
			serviceConfigIn: &servicepb.Service{
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Selector: "selector_1",
							// In reality, path translation should not be set either.
							PathTranslation: servicepb.BackendRule_APPEND_PATH_TO_ADDRESS,
							Deadline:        82,
						},
						{
							Selector:        "selector_2",
							Address:         "https://my-backend.com:8080/api/v2/const-path",
							PathTranslation: servicepb.BackendRule_CONSTANT_ADDRESS,
						},
					},
				},
			},
			wantInfo: map[string]filtergen.TranslationInfo{
				"selector_2": {
					TranslationType: servicepb.BackendRule_CONSTANT_ADDRESS,
					Path:            "/api/v2/const-path",
				},
			},
		},
		{
			desc: "Rules with no translation behavior are not captured",
			serviceConfigIn: &servicepb.Service{
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Selector: "selector_1",
							Address:  "https://my-backend.com:8080/api/v1",
							// should never happen
							PathTranslation: servicepb.BackendRule_PATH_TRANSLATION_UNSPECIFIED,
						},
						{
							Selector:        "selector_2",
							Address:         "https://my-backend.com:8080/api/v2/const-path",
							PathTranslation: servicepb.BackendRule_CONSTANT_ADDRESS,
						},
					},
				},
			},
			wantInfo: map[string]filtergen.TranslationInfo{
				"selector_2": {
					TranslationType: servicepb.BackendRule_CONSTANT_ADDRESS,
					Path:            "/api/v2/const-path",
				},
			},
		},
		{
			desc: "backend address override disables feature",
			serviceConfigIn: &servicepb.Service{
				Backend: &servicepb.Backend{
					Rules: []*servicepb.BackendRule{
						{
							Selector:        "selector_1",
							Address:         "https://my-backend.com:8080/api/v1",
							PathTranslation: servicepb.BackendRule_APPEND_PATH_TO_ADDRESS,
						},
						{
							Selector:        "selector_2",
							Address:         "https://my-backend.com:8080/api/v2/const-path",
							PathTranslation: servicepb.BackendRule_CONSTANT_ADDRESS,
						},
					},
				},
			},
			optsIn: options.ConfigGeneratorOptions{
				EnableBackendAddressOverride: true,
			},
			wantInfo: nil,
		},
	}

	for _, tc := range testdata {
		t.Run(tc.desc, func(t *testing.T) {
			gotInfo, err := filtergen.GenTranslationInfoFromOPConfig(tc.serviceConfigIn, tc.optsIn)
			if err != nil {
				t.Fatalf("GenTranslationInfoFromOPConfig() got unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.wantInfo, gotInfo); diff != "" {
				t.Errorf("GenTranslationInfoFromOPConfig() diff (-want +got):\n%s", diff)
			}
		})
	}
}
