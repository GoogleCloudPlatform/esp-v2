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

package testdata

import (
	"fmt"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
	ptypepb "google.golang.org/genproto/protobuf/ptype"
)

var (
	FakeEchoConfigForDynamicRouting = &confpb.Service{
		Name:              "echo-api.endpoints.cloudesf-testing.cloud.goog",
		Id:                "test-config-id",
		Title:             "Endpoints Example for Dynamic Routing",
		ProducerProjectId: "producer-project",
		Apis: []*apipb.Api{
			{
				Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
				Methods: []*apipb.Method{
					{
						Name:            "Echo",
						RequestTypeUrl:  "type.googleapis.com/EchoRequest",
						ResponseTypeUrl: "type.googleapis.com/EchoMessage",
					},
					{
						Name: "EchoHeader",
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
						Name:           "dynamic_routing_GetBookInfoWithSnakeCase",
						RequestTypeUrl: "type.googleapis.com/GetBookRequest",
					},
					{
						Name:           "dynamic_routing_GetBookIdWithSnakeCase",
						RequestTypeUrl: "type.googleapis.com/GetBookRequest",
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
					{
						Name: "dynamic_routing_Simplegetcors",
					},
					{
						Name: "dynamic_routing_Auth_info_firebase",
					},
					{
						// Uses the default response timeout.
						Name: "dynamic_routing_SleepDurationDefault",
					},
					{
						// "User" specified a shorter response timeout.
						Name: "dynamic_routing_SleepDurationShort",
					},
					{
						Name: "dynamic_routing_Re2ProgramSize",
					},
					{
						Name: "dynamic_routing_Wildcards",
					},
					{
						Name: "dynamic_routing_FieldPath",
					},
					{
						Name: "route_match_order_exact_match",
					},
					{
						Name: "route_match_order_single_wildcard_match",
					},
					{
						Name: "route_match_order_double_wildcard_match",
					},
				},
				Version: "1.0.0",
			},
		},
		Http: &annotationspb.Http{
			Rules: []*annotationspb.HttpRule{
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					Pattern: &annotationspb.HttpRule_Post{
						Post: "/echo",
					},
					Body: "message",
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.EchoHeader",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/echoHeader",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetById",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/pet/{pet_id}/num/{number}",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPet",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/searchpet",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchDogsWithSlash",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/searchdog",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AppendToRoot",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/searchroot",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AppendToRootWithSlash",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/searchrootwithslash",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListPets",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/pets/{category}/year/{no}",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListShelves",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/shelves",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetBookInfoWithSnakeCase",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/shelves/{s_h_e_l_f}/books/info/{b_o_o_k}",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetBookIdWithSnakeCase",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/shelves/{s_h_e_l_f.i_d}/books/id/{b_o_o_k.id}",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification",
					Pattern: &annotationspb.HttpRule_Post{
						Post: "/sc/searchpet",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetByIdWithServiceControlVerification",
					Pattern: &annotationspb.HttpRule_Post{
						Post: "/sc/pet/{pet_id}/num/{number}",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_BearertokenConstantAddress",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/bearertoken/constant/{foo}",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AuthenticationNotSet",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/authenticationnotset/constant/{foo}",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_DisableAuthSetToFalse",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/disableauthsettofalse/constant/{foo}",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_DisableAuthSetToTrue",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/disableauthsettotrue/constant/{foo}",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_BearertokenAppendAddress",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/bearertoken/append",
					},
				},

				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_EmptyPath",
					Pattern: &annotationspb.HttpRule_Post{
						Post: "/empty_path",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Simplegetcors",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/simplegetcors",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Auth_info_firebase",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/auth/info/firebase",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SleepDurationDefault",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/sleepDefault",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SleepDurationShort",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/sleepShort",
					},
				},
				{
					// Regression endpoint for b/148606900.
					// Envoy config validation will fail if the UriTemplate with path parameters is "too long" for regex parsing.
					// Before the program size limit was increased, this would cause Envoy to never be healthy across multiple tests.
					// Specifically, health checks would fail for all tests that relied on this entire backend.
					// After increasing the initial limit, this URL should pass config validation.
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Re2ProgramSize",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/test/{path}/template/test/{path}/template/test/{path}/template/test/{path}/template/test/{path}/template",
					},
				},
				{
					// Regression test for b/162888708.
					// Some of these patterns were not supported in dynamic routing mode.
					// This would cause ESPv2 to crash loop.
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Wildcards",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/wildcard/a/*/b/{name=*}/c/**",
					},
				},
				{
					// Regression test for b/170259809.
					// Envoy route config could not handle field path segment bindings.
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_FieldPath",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/field_path/{s_1=a/*/b/*}/{s_2=x/**}:upload",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.route_match_order_double_wildcard_match",
					Pattern: &annotationspb.HttpRule_Post{
						Post: "/allow-all/{double_wildcard=**}",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.route_match_order_single_wildcard_match",
					Pattern: &annotationspb.HttpRule_Post{
						Post: "/allow-all/{single_wildcard=*}",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.route_match_order_exact_match",
					Pattern: &annotationspb.HttpRule_Post{
						Post: "/allow-all/exact-match",
					},
				},
			},
		},
		Types: []*ptypepb.Type{
			{
				Fields: []*ptypepb.Field{
					{
						JsonName: "BOOK",
						Name:     "b_o_o_k",
					},
					{
						JsonName: "SHELF",
						Name:     "s_h_e_l_f",
					},
				},
				Name: "GetBookRequest",
			},
		},
		Authentication: &confpb.Authentication{
			Rules: []*confpb.AuthenticationRule{
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Auth_info_firebase",
					Requirements: []*confpb.AuthRequirement{
						{
							ProviderId: GoogleJwtProvider,
						},
					},
				},
			},
		},
		Usage: &confpb.Usage{
			Rules: []*confpb.UsageRule{
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.EchoHeader",
					AllowUnregisteredCalls: true,
				},
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
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AuthenticationNotSet",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_DisableAuthSetToTrue",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_DisableAuthSetToFalse",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_EmptyPath",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SleepDurationDefault",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SleepDurationShort",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Wildcards",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_FieldPath",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.route_match_order_exact_match",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.route_match_order_single_wildcard_match",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.route_match_order_double_wildcard_match",
					AllowUnregisteredCalls: true,
				},
			},
		},
		Endpoints: []*confpb.Endpoint{
			{
				Name: "echo-api.endpoints.cloudesf-testing.cloud.goog",
			},
		},
		Backend: &confpb.Backend{
			Rules: []*confpb.BackendRule{
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
					Address:         fmt.Sprintf("https://localhost:%s/echo", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
					// No authentication on this rule, essentially the same as `disable_auth`
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.EchoHeader",
					Address:         fmt.Sprintf("https://localhost:%s/echoHeader", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetById",
					Address:         fmt.Sprintf("https://localhost:%s/dynamicrouting/getpetbyid", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting/getpetbyid",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPet",
					Address:         fmt.Sprintf("https://localhost:%s/dynamicrouting/searchpet", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting/searchpet",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchDogsWithSlash",
					Address:         fmt.Sprintf("https://localhost:%s/dynamicrouting/searchdogs/", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting/searchpet",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AppendToRoot",
					Address:         fmt.Sprintf("https://localhost:%s", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/searchroot",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AppendToRootWithSlash",
					Address:         fmt.Sprintf("https://localhost:%s/", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/searchrootwithslash",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListPets",
					Address:         fmt.Sprintf("https://localhost:%s/dynamicrouting/listpet", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting/listpet",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_ListShelves",
					Address:         fmt.Sprintf("https://localhost:%s/dynamicrouting/shelves", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting/shelves",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_EmptyPath",
					Address:         fmt.Sprintf("https://localhost:%s", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting/emptypath",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetBookInfoWithSnakeCase",
					Address:         fmt.Sprintf("https://localhost:%s/dynamicrouting/bookinfo", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting/bookinfo",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetBookIdWithSnakeCase",
					Address:         fmt.Sprintf("https://localhost:%s/dynamicrouting/bookid", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting/bookid",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SearchPetWithServiceControlVerification",
					Address:         fmt.Sprintf("https://localhost:%s/dynamicrouting/", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_GetPetByIdWithServiceControlVerification",
					Address:         fmt.Sprintf("https://localhost:%s/dynamicrouting/", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/dynamicrouting",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_BearertokenConstantAddress",
					Address:         fmt.Sprintf("https://localhost:%s/bearertoken/constant", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/bearertoken/constant",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_BearertokenAppendAddress",
					Address:         fmt.Sprintf("https://localhost:%s", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/bearertoken/append",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_AuthenticationNotSet",
					Address:         fmt.Sprintf("https://localhost:%s/bearertoken/constant", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_DisableAuthSetToTrue",
					Address:         fmt.Sprintf("https://localhost:%s/bearertoken/constant", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
					Authentication:  &confpb.BackendRule_DisableAuth{DisableAuth: true},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_DisableAuthSetToFalse",
					Address:         fmt.Sprintf("https://localhost:%s/bearertoken/constant", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
					Authentication:  &confpb.BackendRule_DisableAuth{DisableAuth: false},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Simplegetcors",
					Address:         fmt.Sprintf("https://localhost:%s", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/simplegetcors",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Auth_info_firebase",
					Address:         fmt.Sprintf("https://localhost:%s", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/auth/info/firebase",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SleepDurationDefault",
					Address:         fmt.Sprintf("https://localhost:%s", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/sleepDefault",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_SleepDurationShort",
					Address:         fmt.Sprintf("https://localhost:%s", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/sleepShort",
					},
					Deadline: 5.0,
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Re2ProgramSize",
					Address:         fmt.Sprintf("https://localhost:%s", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
					Authentication: &confpb.BackendRule_JwtAudience{
						JwtAudience: "https://localhost/non-existant-url",
					},
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_Wildcards",
					Address:         fmt.Sprintf("https://localhost:%s/dynamicrouting/const_wildcard", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.dynamic_routing_FieldPath",
					Address:         fmt.Sprintf("https://localhost:%s/dynamicrouting/const_wildcard", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.route_match_order_exact_match",
					Address:         fmt.Sprintf("https://localhost:%s/dynamicrouting/const_wildcard", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.route_match_order_single_wildcard_match",
					Address:         fmt.Sprintf("https://localhost:%s/dynamicrouting/const_wildcard", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
				},
				{
					Selector:        "1.echo_api_endpoints_cloudesf_testing_cloud_goog.route_match_order_double_wildcard_match",
					Address:         fmt.Sprintf("https://localhost:%s/dynamicrouting/const_wildcard", platform.WorkingBackendPort),
					PathTranslation: confpb.BackendRule_CONSTANT_ADDRESS,
				},
			},
		},
	}
)
