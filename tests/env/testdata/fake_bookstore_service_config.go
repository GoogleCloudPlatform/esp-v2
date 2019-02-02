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
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/genproto/protobuf/api"
)

var (
	FakeBookstoreConfig = &conf.Service{
		Name:              "bookstore.endpoints.cloudesf-testing.cloud.goog",
		Title:             "Bookstore gRPC API",
		ProducerProjectId: "producer project",
		Apis: []*api.Api{
			{
				Name: "endpoints.examples.bookstore.Bookstore",
				Methods: []*api.Method{
					{
						Name:            "ListShelves",
						RequestTypeUrl:  "type.googleapis.com/google.protobuf.Empty",
						ResponseTypeUrl: "type.googleapis.com/endpoints.examples.bookstore.ListShelvesResponse",
					},
				},
			},
		},
		Http: &annotations.Http{
			Rules: []*annotations.HttpRule{
				{
					Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
					Pattern: &annotations.HttpRule_Get{
						Get: "/v1/shelves",
					},
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.CreateShelf",
					Pattern: &annotations.HttpRule_Post{
						Post: "/v1/shelves",
					},
					Body: "shelf",
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.GetShelf",
					Pattern: &annotations.HttpRule_Get{
						Get: "/v1/shelves/{shelf=*}",
					},
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.DeleteShelf",
					Pattern: &annotations.HttpRule_Delete{
						Delete: "/v1/shelves/{shelf}",
					},
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.DeleteBook",
					Pattern: &annotations.HttpRule_Delete{
						Delete: "/v1/shelves/{shelf=*}/books/{book=*}",
					},
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.CreateBook",
					Pattern: &annotations.HttpRule_Post{
						Post: "/v1/shelves/{shelf}/books",
					},
					Body: "book",
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.GetBook",
					Pattern: &annotations.HttpRule_Get{
						Get: "/v1/shelves/{shelf=*}/books/{book}",
					},
				},
			},
		},
		Authentication: &conf.Authentication{
			Rules: []*conf.AuthenticationRule{
				{
					Selector: "endpoints.examples.bookstore.Bookstore.GetShelf",
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.CreateShelf",
					Requirements: []*conf.AuthRequirement{
						{
							ProviderId: "google_service_account",
						},
					},
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.GetBook",
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.DeleteBook",
					Requirements: []*conf.AuthRequirement{
						{
							ProviderId: "google_service_account",
						},
					},
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.CreateBook",
					Requirements: []*conf.AuthRequirement{
						{
							ProviderId: "google_service_account",
							Audiences:  "bookstore_test_client.cloud.goog, admin.cloud.goog",
						},
					},
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
					Requirements: []*conf.AuthRequirement{
						{
							ProviderId: "google_service_account",
							Audiences:  "bookstore_test_client.cloud.goog",
						},
					},
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.DeleteShelf",
					Requirements: []*conf.AuthRequirement{
						{
							ProviderId: "google_service_account",
							Audiences:  "bookstore_test_client.cloud.goog",
						},
						{
							ProviderId: "endpoints_jwt",
						},
					},
				},
			},
		},
	}
)
