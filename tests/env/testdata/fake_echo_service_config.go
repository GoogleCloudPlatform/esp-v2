// Copyright 2018 Google Cloud Platform Proxy Authors
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

package testdata

import (
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/genproto/protobuf/api"

	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

var (
	FakeEchoConfig = &conf.Service{
		Name:              "echo-api.endpoints.cloudesf-testing.cloud.goog",
		Title:             "Endpoints Example",
		ProducerProjectId: "producer-project",
		Apis: []*api.Api{
			{
				Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
				Methods: []*api.Method{
					{
						Name:            "Auth_info_google_jwt",
						RequestTypeUrl:  "type.googleapis.com/google.protobuf.Empty",
						ResponseTypeUrl: "type.googleapis.com/AuthInfoResponse",
					},
					{
						Name:            "Echo",
						RequestTypeUrl:  "type.googleapis.com/EchoRequest",
						ResponseTypeUrl: "type.googleapis.com/EchoMessage",
					},
					{
						Name:            "Simplegetcors",
						RequestTypeUrl:  "type.googleapis.com/google.protobuf.Empty",
						ResponseTypeUrl: "type.googleapis.com/SimpleCorsMessage",
					},
					{
						Name:            "Auth_info_firebase",
						RequestTypeUrl:  "type.googleapis.com/google.protobuf.Empty",
						ResponseTypeUrl: "type.googleapis.com/AuthInfoResponse",
					},
				},
				Version: "1.0.0",
			},
		},
		Http: &annotations.Http{
			Rules: []*annotations.HttpRule{
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth_info_google_jwt",
					Pattern: &annotations.HttpRule_Get{
						Get: "/auth/info/googlejwt",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth0",
					Pattern: &annotations.HttpRule_Get{
						Get: "/auth/info/auth0",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					Pattern: &annotations.HttpRule_Post{
						Post: "/echo",
					},
					Body: "message",
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey",
					Pattern: &annotations.HttpRule_Post{
						Post: "/echo/nokey",
					},
					Body: "message",
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simplegetcors",
					Pattern: &annotations.HttpRule_Get{
						Get: "/simplegetcors",
					},
				},
				{
					Selector: "_post_anypath",
					Pattern: &annotations.HttpRule_Post{
						Post: "/**",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth_info_firebase",
					Pattern: &annotations.HttpRule_Get{
						Get: "/auth/info/firebase",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Bearertoken",
					Pattern: &annotations.HttpRule_Get{
						Get: "/bearertoken",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing.GetPetById",
					Pattern: &annotations.HttpRule_Get{
						Get: "/pet/{pet_id}/num/{number}",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing.SearchPet",
					Pattern: &annotations.HttpRule_Get{
						Get: "/searchpet",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing.SearchDogsWithSlash",
					Pattern: &annotations.HttpRule_Get{
						Get: "/searchdog",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing.ListPets",
					Pattern: &annotations.HttpRule_Get{
						Get: "/pets/{category}/year/{no}",
					},
				},
			},
		},
		Authentication: &conf.Authentication{
			Rules: []*conf.AuthenticationRule{
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth_info_google_jwt",
					Requirements: []*conf.AuthRequirement{
						{
							ProviderId: "google_jwt",
						},
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth0",
					Requirements: []*conf.AuthRequirement{
						{
							ProviderId: "google_jwt",
							Audiences:  "admin.cloud.goog",
						},
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simplegetcors",
				},
				{
					Selector: "_post_anypath",
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth_info_firebase",
					Requirements: []*conf.AuthRequirement{
						{
							ProviderId: "google_jwt",
						},
					},
				},
			},
		},
		Usage: &conf.Usage{
			Rules: []*conf.UsageRule{
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "_post_anypath",
					AllowUnregisteredCalls: true,
				},
			},
		},
		Endpoints: []*conf.Endpoint{
			{
				Name:      "echo-api.endpoints.cloudesf-testing.cloud.goog",
				AllowCors: true,
			},
		},
		Backend: &conf.Backend{
			Rules: []*conf.BackendRule{
				&conf.BackendRule{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
				},
				&conf.BackendRule{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Bearertoken",
					Authentication: &conf.BackendRule_JwtAudience{
						JwtAudience: "mybackend.com",
					},
				},
				&conf.BackendRule{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing.GetPetById",
					Address:         "https://localhost:-1/dynamicrouting/getpetbyid",
					PathTranslation: conf.BackendRule_CONSTANT_ADDRESS,
					Authentication: &conf.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting/getpetbyid",
					},
				},
				&conf.BackendRule{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing.SearchPet",
					Address:         "https://localhost:-1/dynamicrouting/searchpet",
					PathTranslation: conf.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &conf.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting/searchpet",
					},
				},
				&conf.BackendRule{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing.SearchDogsWithSlash",
					Address:         "https://localhost:-1/dynamicrouting/searchdogs/",
					PathTranslation: conf.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &conf.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting/searchpet",
					},
				},
				&conf.BackendRule{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing.ListPets",
					Address:         "https://localhost:-1/dynamicrouting/listpet",
					PathTranslation: conf.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &conf.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting/listpet",
					},
				},
			},
		},
	}
)
