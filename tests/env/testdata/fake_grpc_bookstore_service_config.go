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
)

var (
	FakeBookstoreConfig = &confpb.Service{
		Name:              "bookstore.endpoints.cloudesf-testing.cloud.goog",
		Id:                "test-config-id",
		Title:             "Bookstore gRPC API",
		ProducerProjectId: "producer project",
		Apis: []*apipb.Api{
			{
				Name: "endpoints.examples.bookstore.Bookstore",
				Methods: []*apipb.Method{
					{
						Name:            "ListShelves",
						RequestTypeUrl:  "type.googleapis.com/google.protobuf.Empty",
						ResponseTypeUrl: "type.googleapis.com/endpoints.examples.bookstore.ListShelvesResponse",
					},
					{
						Name:            "CreateShelf",
						RequestTypeUrl:  "type.googleapis.com/endpoints.examples.bookstore.CreateShelfRequest",
						ResponseTypeUrl: "type.googleapis.com/endpoints.examples.bookstore.Shelf",
					},
					{
						Name:            "GetShelf",
						RequestTypeUrl:  "type.googleapis.com/endpoints.examples.bookstore.GetShelf",
						ResponseTypeUrl: "type.googleapis.com/endpoints.examples.bookstore.Shelf",
					},
					{
						Name:            "DeleteShelf",
						RequestTypeUrl:  "type.googleapis.com/endpoints.examples.bookstore.DeleteShelfRequest",
						ResponseTypeUrl: "type.googleapis.com/google.protobuf.Empty",
					},
					{
						Name:            "ListBooks",
						RequestTypeUrl:  "type.googleapis.com/endpoints.examples.bookstore.ListBooksRequest",
						ResponseTypeUrl: "type.googleapis.com/endpoints.examples.bookstore.ListBooksResponse",
					},
					{
						Name:            "CreateBook",
						RequestTypeUrl:  "type.googleapis.com/endpoints.examples.bookstore.CreateBookRequest",
						ResponseTypeUrl: "type.googleapis.com/endpoints.examples.bookstore.Book",
					},
					{
						Name:            "CreateBookWithTrailingSingleWildcard",
						RequestTypeUrl:  "type.googleapis.com/endpoints.examples.bookstore.CreateBookRequest",
						ResponseTypeUrl: "type.googleapis.com/endpoints.examples.bookstore.Book",
					},
					{
						Name:            "CreateBookWithTrailingDoubleWildcard",
						RequestTypeUrl:  "type.googleapis.com/endpoints.examples.bookstore.CreateBookRequest",
						ResponseTypeUrl: "type.googleapis.com/endpoints.examples.bookstore.Book",
					},
					{
						Name:            "CreateBookWithCustomVerb",
						RequestTypeUrl:  "type.googleapis.com/endpoints.examples.bookstore.CreateBookRequest",
						ResponseTypeUrl: "type.googleapis.com/endpoints.examples.bookstore.Book",
					},
					{
						Name:            "GetBook",
						RequestTypeUrl:  "type.googleapis.com/endpoints.examples.bookstore.GetBookRequest",
						ResponseTypeUrl: "type.googleapis.com/endpoints.examples.bookstore.Book",
					},
					{
						Name:            "DeleteBook",
						RequestTypeUrl:  "type.googleapis.com/endpoints.examples.bookstore.DeleteBookRequest",
						ResponseTypeUrl: "type.googleapis.com/google.protobuf.Empty",
					},
					{
						Name:            "ReturnBadStatus",
						RequestTypeUrl:  "type.googleapis.com/google.protobuf.Empty",
						ResponseTypeUrl: "type.googleapis.com/google.protobuf.Empty",
					},
					{
						Name: "GetShelfAutoBind",
					},
					{
						Name: "Unspecified",
					},
				},
				Version: "1.0.0",
			},
			{
				Name: "endpoints.examples.bookstore.v2.Bookstore",
				Methods: []*apipb.Method{
					{
						Name:            "GetShelf",
						RequestTypeUrl:  "type.googleapis.com/endpoints.examples.bookstore.v2.GetShelf",
						ResponseTypeUrl: "type.googleapis.com/endpoints.examples.bookstore.v2.Shelf",
					},
					{
						Name:            "GetShelfAutoBind",
						RequestTypeUrl:  "type.googleapis.com/endpoints.examples.bookstore.v2.GetShelf",
						ResponseTypeUrl: "type.googleapis.com/endpoints.examples.bookstore.v2.Shelf",
					},
				},
				Version: "1.0.0",
			},
		},
		Http: &annotationspb.Http{
			Rules: []*annotationspb.HttpRule{
				{
					Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/v1/shelves",
					},
				},
				{
					Selector: "endpoints.examples.bookstore.v2.Bookstore.GetShelf",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/v2/shelves/{shelf=*}",
					},
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.CreateShelf",
					Pattern: &annotationspb.HttpRule_Post{
						Post: "/v1/shelves",
					},
					Body: "shelf",
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.GetShelf",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/v1/shelves/{shelf=*}",
					},
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.DeleteShelf",
					Pattern: &annotationspb.HttpRule_Delete{
						Delete: "/v1/shelves/{shelf}",
					},
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.ListBooks",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/v1/shelves/{shelf}/books",
					},
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.DeleteBook",
					Pattern: &annotationspb.HttpRule_Delete{
						Delete: "/v1/shelves/{shelf=*}/books/{book=*}",
					},
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.CreateBook",
					Pattern: &annotationspb.HttpRule_Post{
						Post: "/v1/shelves/{shelf}/books",
					},
					Body: "book",
					AdditionalBindings: []*annotationspb.HttpRule{
						{
							Pattern: &annotationspb.HttpRule_Post{
								Post: "/v1/shelves/{shelf}/books/{book.id}/{book.author}",
							},
							Body: "book",
						},
					},
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.CreateBookWithTrailingSingleWildcard",
					Pattern: &annotationspb.HttpRule_Post{
						Post: "/v1/shelves/{shelf}/single/*",
					},
					Body: "book",
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.CreateBookWithTrailingDoubleWildcard",
					Pattern: &annotationspb.HttpRule_Post{
						Post: "/v1/shelves/{shelf}/double/**",
					},
					Body: "book",
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.CreateBookWithCustomVerb",
					Pattern: &annotationspb.HttpRule_Post{
						Post: "/v1/shelves/{shelf}:registeredCustomVerb",
					},
					Body: "book",
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.GetBook",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/v1/shelves/{shelf=*}/books/{book}",
					},
				},
			},
		},
		Authentication: &confpb.Authentication{
			Rules: []*confpb.AuthenticationRule{
				{
					Selector: "endpoints.examples.bookstore.Bookstore.GetShelf",
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.CreateShelf",
					Requirements: []*confpb.AuthRequirement{
						{
							ProviderId: GoogleServiceAccountProvider,
						},
					},
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.GetBook",
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.DeleteBook",
					Requirements: []*confpb.AuthRequirement{
						{
							ProviderId: GoogleServiceAccountProvider,
						},
					},
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.CreateBook",
					Requirements: []*confpb.AuthRequirement{
						{
							ProviderId: GoogleServiceAccountProvider,
							Audiences:  "bookstore_test_client.cloud.goog, admin.cloud.goog",
						},
					},
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
					Requirements: []*confpb.AuthRequirement{
						{
							ProviderId: GoogleServiceAccountProvider,
							Audiences:  "bookstore_test_client.cloud.goog",
						},
					},
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.DeleteShelf",
					Requirements: []*confpb.AuthRequirement{
						{
							ProviderId: GoogleServiceAccountProvider,
							Audiences:  "bookstore_test_client.cloud.goog",
						},
						{
							ProviderId: EndpointsJwtProvider,
						},
					},
				},
			},
		},
		Usage: &confpb.Usage{
			Rules: []*confpb.UsageRule{
				{
					Selector:               "endpoints.examples.bookstore.Bookstore.Unspecified",
					AllowUnregisteredCalls: true,
				},
			},
		},
	}
)
