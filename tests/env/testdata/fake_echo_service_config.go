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
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Simplegetcors",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/simplegetcors",
					},
				},
				{
					Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog._post_anypath",
					Pattern: &annotationspb.HttpRule_Post{
						Post: "/**",
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
					Selector:               "1.echo_api_endpoints_cloudesf_testing_cloud_goog.WebsocketEcho",
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
