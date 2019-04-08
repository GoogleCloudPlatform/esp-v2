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
	ptype "google.golang.org/genproto/protobuf/ptype"
)

var (
	FakeEchoConfigForDynamicRouting = &conf.Service{
		Name:              "echo-api.endpoints.cloudesf-testing.cloud.goog",
		Title:             "Endpoints Example for Dynamic Routing",
		ProducerProjectId: "producer-project",
		Apis: []*api.Api{
			{
				Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
				Methods: []*api.Method{
					{
						Name:            "Echo",
						RequestTypeUrl:  "type.googleapis.com/EchoRequest",
						ResponseTypeUrl: "type.googleapis.com/EchoMessage",
					},
					{
						Name: "dynamic_routing_GetPetById",
					},
					{
						Name: "dynamic_routing_SearchPet",
					},
					{
						Name: "dynamic_routing_SearchDogsWithSlash",
					},
					{
						Name: "dynamic_routing_AppendToRoot",
					},
					{
						Name: "dynamic_routing_AppendToRootWithSlash",
					},
					{
						Name: "dynamic_routing_ListPets",
					},
					{
						Name: "dynamic_routing_ListShelves",
					},
					{
						Name: "dynamic_routing_GetBookInfoWithSnakeCase",
					},
					{
						Name: "dynamic_routing_GetBookIdWithSnakeCase",
					},
					{
						Name: "dynamic_routing_SearchPetWithServiceControlVerification",
					},
					{
						Name: "dynamic_routing_GetPetByIdWithServiceControlVerification",
					},
					{
						Name: "dynamic_routing_BearertokenConstantAddress",
					},
					{
						Name: "dynamic_routing_BearertokenAppendAddress",
					},
				},
				Version: "1.0.0",
			},
		},
		Http: &annotations.Http{
			Rules: []*annotations.HttpRule{
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					Pattern: &annotations.HttpRule_Post{
						Post: "/echo",
					},
					Body: "message",
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetById",
					Pattern: &annotations.HttpRule_Get{
						Get: "/pet/{pet_id}/num/{number}",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPet",
					Pattern: &annotations.HttpRule_Get{
						Get: "/searchpet",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchDogsWithSlash",
					Pattern: &annotations.HttpRule_Get{
						Get: "/searchdog",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AppendToRoot",
					Pattern: &annotations.HttpRule_Get{
						Get: "/searchroot",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AppendToRootWithSlash",
					Pattern: &annotations.HttpRule_Get{
						Get: "/searchrootwithslash",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListPets",
					Pattern: &annotations.HttpRule_Get{
						Get: "/pets/{category}/year/{no}",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListShelves",
					Pattern: &annotations.HttpRule_Get{
						Get: "/shelves",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetBookInfoWithSnakeCase",
					Pattern: &annotations.HttpRule_Get{
						Get: "/shelves/{s_h_e_l_f}/books/info/{b_o_o_k}",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetBookIdWithSnakeCase",
					Pattern: &annotations.HttpRule_Get{
						Get: "/shelves/{s_h_e_l_f.i_d}/books/id/{b_o_o_k.id}",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification",
					Pattern: &annotations.HttpRule_Post{
						Post: "/sc/searchpet",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetByIdWithServiceControlVerification",
					Pattern: &annotations.HttpRule_Post{
						Post: "/sc/pet/{pet_id}/num/{number}",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_BearertokenConstantAddress",
					Pattern: &annotations.HttpRule_Get{
						Get: "/bearertoken/constant/{foo}",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_BearertokenAppendAddress",
					Pattern: &annotations.HttpRule_Get{
						Get: "/bearertoken/append",
					},
				},
			},
		},
		Types: []*ptype.Type{
			{
				Fields: []*ptype.Field{
					&ptype.Field{
						JsonName: "BOOK",
						Name:     "b_o_o_k",
					},
					&ptype.Field{
						JsonName: "SHELF",
						Name:     "s_h_e_l_f",
					},
				},
			},
		},
		Authentication: &conf.Authentication{
			Rules: []*conf.AuthenticationRule{
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
				},
			},
		},
		Usage: &conf.Usage{
			Rules: []*conf.UsageRule{
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetById",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPet",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchDogsWithSlash",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AppendToRoot",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AppendToRootWithSlash",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListPets",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListShelves",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetBookInfoWithSnakeCase",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetBookIdWithSnakeCase",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_BearertokenConstantAddress",
					AllowUnregisteredCalls: true,
				},
			},
		},
		Endpoints: []*conf.Endpoint{
			{
				Name: "echo-api.endpoints.cloudesf-testing.cloud.goog",
			},
		},
		Backend: &conf.Backend{
			Rules: []*conf.BackendRule{
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetById",
					Address:         "https://localhost:-1/dynamicrouting/getpetbyid",
					PathTranslation: conf.BackendRule_CONSTANT_ADDRESS,
					Authentication: &conf.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting/getpetbyid",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPet",
					Address:         "https://localhost:-1/dynamicrouting/searchpet",
					PathTranslation: conf.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &conf.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting/searchpet",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchDogsWithSlash",
					Address:         "https://localhost:-1/dynamicrouting/searchdogs/",
					PathTranslation: conf.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &conf.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting/searchpet",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AppendToRoot",
					Address:         "https://localhost:-1",
					PathTranslation: conf.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &conf.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/searchroot",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AppendToRootWithSlash",
					Address:         "https://localhost:-1/",
					PathTranslation: conf.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &conf.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/searchrootwithslash",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListPets",
					Address:         "https://localhost:-1/dynamicrouting/listpet",
					PathTranslation: conf.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &conf.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting/listpet",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListShelves",
					Address:         "https://localhost:-1/dynamicrouting/shelves",
					PathTranslation: conf.BackendRule_CONSTANT_ADDRESS,
					Authentication: &conf.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting/shelves",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetBookInfoWithSnakeCase",
					Address:         "https://localhost:-1/dynamicrouting/bookinfo",
					PathTranslation: conf.BackendRule_CONSTANT_ADDRESS,
					Authentication: &conf.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting/bookinfo",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetBookIdWithSnakeCase",
					Address:         "https://localhost:-1/dynamicrouting/bookid",
					PathTranslation: conf.BackendRule_CONSTANT_ADDRESS,
					Authentication: &conf.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting/bookid",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification",
					Address:         "https://localhost:-1/dynamicrouting/",
					PathTranslation: conf.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &conf.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetByIdWithServiceControlVerification",
					Address:         "https://localhost:-1/dynamicrouting/",
					PathTranslation: conf.BackendRule_CONSTANT_ADDRESS,
					Authentication: &conf.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_BearertokenConstantAddress",
					Address:         "https://localhost:-1/bearertoken/constant",
					PathTranslation: conf.BackendRule_CONSTANT_ADDRESS,
					Authentication: &conf.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/bearertoken/constant",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_BearertokenAppendAddress",
					Address:         "https://localhost:-1",
					PathTranslation: conf.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &conf.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/bearertoken/append",
					},
				},
			},
		},
	}
)
