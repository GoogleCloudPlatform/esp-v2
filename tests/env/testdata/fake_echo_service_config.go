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
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
	ptypepb "google.golang.org/genproto/protobuf/ptype"
)

var (
	FakeEchoConfig = &confpb.Service{
		Name:              "echo-api.endpoints.cloudesf-testing.cloud.goog",
		Id:                "test-config-id",
		Title:             "Endpoints Example",
		ProducerProjectId: "producer-project",
		Apis: []*apipb.Api{
			{
				Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
				Methods: []*apipb.Method{
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
						Name:            "WebsocketEcho",
						RequestTypeUrl:  "type.googleapis.com/EchoRequest",
						ResponseTypeUrl: "type.googleapis.com/EchoMessage",
					},
					{
						Name: "Simpleget",
					},
					{
						Name: "SimplegetNotModified",
					},
					{
						Name: "SimplegetUnauthorized",
					},
					{
						Name: "SimplegetForbidden",
					},
					{
						Name:            "Simplegetcors",
						RequestTypeUrl:  "type.googleapis.com/google.protobuf.Empty",
						ResponseTypeUrl: "type.googleapis.com/SimpleCorsMessage",
					},
					{
						Name: "IPVersion",
					},
					{
						Name:            "Auth_info_firebase",
						RequestTypeUrl:  "type.googleapis.com/google.protobuf.Empty",
						ResponseTypeUrl: "type.googleapis.com/AuthInfoResponse",
					},
					{
						Name: "Sleep",
					},
					{
						Name: "SleepWithBackendRule",
					},
					{
						Name: "Auth0",
					},
					{
						Name: "EchoHeader",
					},
					{
						Name: "EchoGetWithBody",
					},
					{
						Name: "echoGET",
					},
					{
						Name: "echoPOST",
					},
					{
						Name: "echoPUT",
					},
					{
						Name: "echoPATCH",
					},
					{
						Name: "echoDELETE",
					},
					{
						Name: "Root",
					},
					{
						Name: "Echo_nokey",
					},
					{
						Name: "_post_anypath",
					},
					{
						Name: "Echo_nokey_override_as_get",
					},
					{
						Name: "CorsShelves",
					},
					{
						Name: "GetShelf",
					},
					{
						Name: "UpdateShelf",
					},
					{
						Name: "DeleteShelf",
					},
				},
				Version: "1.0.0",
			},
		},
		Backend: &confpb.Backend{
			// ESPv2 supports backend rules even in sidecar mode.
			Rules: []*confpb.BackendRule{
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.SleepWithBackendRule",
					Deadline: 5,
					Authentication: &confpb.BackendRule_DisableAuth{
						DisableAuth: true,
					},
				},
			},
		},
		Http: &annotationspb.Http{
			Rules: []*annotationspb.HttpRule{
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth_info_google_jwt",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/auth/info/googlejwt",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth0",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/auth/info/auth0",
					},
				},
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
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.EchoGetWithBody",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/echo",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.WebsocketEcho",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/websocketecho",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Root",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey",
					Pattern: &annotationspb.HttpRule_Post{
						Post: "/echo/nokey",
					},
					Body: "message",
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simpleget",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/simpleget",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.SimplegetNotModified",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/simpleget/304",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.SimplegetUnauthorized",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/simpleget/401",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.SimplegetForbidden",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/simpleget/403",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simplegetcors",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/simplegetcors",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.IPVersion",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/ipversion",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog._post_anypath",
					Pattern: &annotationspb.HttpRule_Post{
						Post: "/anypath/**",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth_info_firebase",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/auth/info/firebase",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Sleep",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/sleep",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.SleepWithBackendRule",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/sleep/with/backend/rule",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey_override_as_get",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/echo/nokey/OverrideAsGet",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.CorsShelves",
					Pattern: &annotationspb.HttpRule_Custom{
						Custom: &annotationspb.CustomHttpPattern{
							Kind: "OPTIONS",
							Path: "/bookstore/shelves",
						},
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.GetShelf",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/bookstore/shelves/{shelf}",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoGET",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/echoMethod",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echoPOST",
					Pattern: &annotationspb.HttpRule_Post{
						Post: "/echoMethod",
					},
				},
			},
		},
		Types: []*ptypepb.Type{
			{
				Fields: []*ptypepb.Field{
					&ptypepb.Field{
						JsonName: "BOOK",
						Name:     "b_o_o_k",
					},
					&ptypepb.Field{
						JsonName: "SHELF",
						Name:     "s_h_e_l_f",
					},
				},
			},
		},
		Authentication: &confpb.Authentication{
			Rules: []*confpb.AuthenticationRule{
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth_info_google_jwt",
					Requirements: []*confpb.AuthRequirement{
						{
							ProviderId: GoogleJwtProvider,
						},
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth0",
					Requirements: []*confpb.AuthRequirement{
						{
							ProviderId: GoogleJwtProvider,
							Audiences:  "admin.cloud.goog",
						},
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.WebsocketEcho",
					Requirements: []*confpb.AuthRequirement{
						{
							ProviderId: GoogleJwtProvider,
							Audiences:  "admin.cloud.goog",
						},
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simplegetcors",
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog._post_anypath",
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth_info_firebase",
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
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Root",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog._post_anypath",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Sleep",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.SleepWithBackendRule",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_nokey_override_as_get",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.SimplegetNotModified",
					AllowUnregisteredCalls: true,
				},
				{
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.SimplegetForbidden",
					AllowUnregisteredCalls: true,
				},
			},
		},
		Endpoints: []*confpb.Endpoint{
			{
				Name: "echo-api.endpoints.cloudesf-testing.cloud.goog",
			},
		},
	}
)
