// Copyright 2020 Google LLC
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
	FakeBookstoreConfigForDynamicRouting = &confpb.Service{
		Name:              "bookstore.endpoints.cloudesf-testing.cloud.goog",
		Id:                "test-config-id",
		Title:             "Bookstore gRPC API",
		ProducerProjectId: "producer project",
		Apis: []*apipb.Api{
			{
				Name: "endpoints.examples.bookstore.Bookstore",
				Methods: []*apipb.Method{
					{
						Name:            "GetShelf",
						RequestTypeUrl:  "type.googleapis.com/endpoints.examples.bookstore.GetShelf",
						ResponseTypeUrl: "type.googleapis.com/endpoints.examples.bookstore.Shelf",
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
				},
				Version: "1.0.0",
			},
		},
		Http: &annotationspb.Http{
			Rules: []*annotationspb.HttpRule{
				{
					Selector: "endpoints.examples.bookstore.v2.Bookstore.GetShelf",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/v2/shelves/{shelf=*}",
					},
				},
				{
					Selector: "endpoints.examples.bookstore.Bookstore.GetShelf",
					Pattern: &annotationspb.HttpRule_Get{
						Get: "/v1/shelves/{shelf=*}",
					},
				},
			},
		},
		Backend: &confpb.Backend{
			Rules: []*confpb.BackendRule{
				{
					Selector: "endpoints.examples.bookstore.Bookstore.GetShelf",
					Address:  "grpcs://localhost:-1/",
					// No authentication on this rule, essentially the same as `disable_auth`
				},
			},
		},
		Usage: &confpb.Usage{
			Rules: []*confpb.UsageRule{},
		},
	}
)
