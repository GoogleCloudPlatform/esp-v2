// Copyright 2019 Google Cloud Platform Proxy Authors
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

package configinfo

import (
	"flag"
	"fmt"
	"reflect"
	"testing"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/genproto/protobuf/api"

	commonpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/common"
	scpb "cloudesf.googlesource.com/gcpproxy/src/go/proto/api/envoy/http/service_control"
	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
	conf "google.golang.org/genproto/googleapis/api/serviceconfig"
)

var (
	testProjectName = "bookstore.endpoints.project123.cloud.goog"
	testApiName     = "endpoints.examples.bookstore.Bookstore"
	testConfigID    = "2019-03-02r0"
)

func TestProcessEndpoints(t *testing.T) {
	testData := []struct {
		desc              string
		fakeServiceConfig *conf.Service
		wantedAllowCors   bool
	}{
		{
			desc: "Return true for endpoint name matching service name",
			fakeServiceConfig: &conf.Service{
				Name: testProjectName,
				Apis: []*api.Api{
					{
						Name: testApiName,
					},
				},
				Endpoints: []*conf.Endpoint{
					{
						Name:      testProjectName,
						AllowCors: true,
					},
				},
			},
			wantedAllowCors: true,
		},
		{
			desc: "Return false for not setting allow_cors",
			fakeServiceConfig: &conf.Service{
				Name: testProjectName,
				Apis: []*api.Api{
					{
						Name: testApiName,
					},
				},
				Endpoints: []*conf.Endpoint{
					{
						Name: testProjectName,
					},
				},
			},
			wantedAllowCors: false,
		},
		{
			desc: "Return false for endpoint name not matching service name",
			fakeServiceConfig: &conf.Service{
				Name: testProjectName,
				Apis: []*api.Api{
					{
						Name: testApiName,
					},
				},
				Endpoints: []*conf.Endpoint{
					{
						Name:      "echo.endpoints.project123.cloud.goog",
						AllowCors: true,
					},
				},
			},
			wantedAllowCors: false,
		},
		{
			desc: "Return false for empty endpoint field",
			fakeServiceConfig: &conf.Service{
				Name: testProjectName,
				Apis: []*api.Api{
					{
						Name: testApiName,
					},
				},
			},
			wantedAllowCors: false,
		},
	}

	for i, tc := range testData {
		flag.Set("backend_protocol", "grpc")
		serviceInfo, err := NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID)
		if err != nil {
			t.Fatal(err)
		}

		if serviceInfo.AllowCors != tc.wantedAllowCors {
			t.Errorf("Test Desc(%d): %s, allow CORS flag got: %v, want: %v", i, tc.desc, serviceInfo.AllowCors, tc.wantedAllowCors)
		}
	}
}

func TestExtractAPIKeyLocations(t *testing.T) {
	testData := []struct {
		desc                   string
		fakeServiceConfig      *conf.Service
		wantedSystemParameters map[string][]*conf.SystemParameter
		wantMethods            map[string]*methodInfo
	}{

		{
			desc: "Succeed, only url query",
			fakeServiceConfig: &conf.Service{
				Apis: []*api.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*api.Method{
							{
								Name: "echo",
							},
						},
					},
				},
				SystemParameters: &conf.SystemParameters{
					Rules: []*conf.SystemParameterRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echo",
							Parameters: []*conf.SystemParameter{
								{
									Name:       "api_key",
									HttpHeader: "header_name",
								},
							},
						},
					},
				},
			},
			wantMethods: map[string]*methodInfo{
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.echo": &methodInfo{
					ShortName: "echo",
					APIKeyLocations: []*scpb.APIKeyLocation{
						{
							Key: &scpb.APIKeyLocation_Header{
								Header: "header_name",
							},
						},
					},
				},
			},
		},

		{
			desc: "Succeed, only header",
			fakeServiceConfig: &conf.Service{
				Apis: []*api.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*api.Method{
							{
								Name: "echo",
							},
						},
					},
				},
				SystemParameters: &conf.SystemParameters{
					Rules: []*conf.SystemParameterRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echo",
							Parameters: []*conf.SystemParameter{
								{
									Name:              "api_key",
									UrlQueryParameter: "query_name",
								},
							},
						},
					},
				},
			},
			wantMethods: map[string]*methodInfo{
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.echo": &methodInfo{
					ShortName: "echo",
					APIKeyLocations: []*scpb.APIKeyLocation{
						{
							Key: &scpb.APIKeyLocation_Query{
								Query: "query_name",
							},
						},
					},
				},
			},
		},

		{
			desc: "Succeed, url query plus header",
			fakeServiceConfig: &conf.Service{
				Apis: []*api.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*api.Method{
							{
								Name: "echo",
							},
						},
					},
				},
				SystemParameters: &conf.SystemParameters{
					Rules: []*conf.SystemParameterRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echo",
							Parameters: []*conf.SystemParameter{
								{
									Name:              "api_key",
									HttpHeader:        "header_name_1",
									UrlQueryParameter: "query_name_1",
								},
								{
									Name:              "api_key",
									HttpHeader:        "header_name_2",
									UrlQueryParameter: "query_name_2",
								},
							},
						},
					},
				},
			},
			wantMethods: map[string]*methodInfo{
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.echo": &methodInfo{
					ShortName: "echo",
					APIKeyLocations: []*scpb.APIKeyLocation{
						{
							Key: &scpb.APIKeyLocation_Query{
								Query: "query_name_1",
							},
						},
						{
							Key: &scpb.APIKeyLocation_Query{
								Query: "query_name_2",
							},
						},
						{
							Key: &scpb.APIKeyLocation_Header{
								Header: "header_name_1",
							},
						},
						{
							Key: &scpb.APIKeyLocation_Header{
								Header: "header_name_2",
							},
						},
					},
				},
			},
		},
	}
	for i, tc := range testData {
		flag.Set("backend_protocol", "grpc")
		serviceInfo, err := NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID)
		if err != nil {
			t.Fatal(err)
		}
		if len(serviceInfo.Methods) != len(tc.wantMethods) {
			t.Errorf("Test Desc(%d): %s, got: %v, wanted: %v", i, tc.desc, serviceInfo.Methods, tc.wantMethods)
		}
		for key, gotMethod := range serviceInfo.Methods {
			wantMethod := tc.wantMethods[key]
			if eq := reflect.DeepEqual(gotMethod, wantMethod); !eq {
				t.Errorf("Test Desc(%d): %s, \ngot: %v,\nwanted: %v", i, tc.desc, gotMethod, wantMethod)
			}
		}
	}
}

func TestMethods(t *testing.T) {
	testData := []struct {
		desc              string
		fakeServiceConfig *conf.Service
		backendProtocol   string
		wantMethods       map[string]*methodInfo
	}{
		{
			desc: "Succeed for gRPC, no Http rule",
			fakeServiceConfig: &conf.Service{
				Name: testProjectName,
				Apis: []*api.Api{
					{
						Name: testApiName,
						Methods: []*api.Method{
							{
								Name: "ListShelves",
							},
							{
								Name: "CreateShelf",
							},
						},
					},
				},
			},
			backendProtocol: "gRPC",
			wantMethods: map[string]*methodInfo{
				fmt.Sprintf("%s.%s", testApiName, "ListShelves"): &methodInfo{
					ShortName: "ListShelves",
				},
				fmt.Sprintf("%s.%s", testApiName, "CreateShelf"): &methodInfo{
					ShortName: "CreateShelf",
				},
			},
		},
		{
			desc: "Succeed for HTTP",
			fakeServiceConfig: &conf.Service{
				Name: testProjectName,
				Apis: []*api.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*api.Method{
							{
								Name: "Echo",
							},
							{
								Name: "Echo_Auth_Jwt",
							},
						},
					},
				},
				Http: &annotations.Http{
					Rules: []*annotations.HttpRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt",
							Pattern: &annotations.HttpRule_Get{
								Get: "/auth/info/googlejwt",
							},
						},
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
							Pattern: &annotations.HttpRule_Post{
								Post: "/echo",
							},
							Body: "message",
						},
					},
				},
			},
			backendProtocol: "http2",
			wantMethods: map[string]*methodInfo{
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo": &methodInfo{
					ShortName: "Echo",
					HttpRule: commonpb.Pattern{
						UriTemplate: "/echo",
						HttpMethod:  ut.POST,
					},
				},
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt": &methodInfo{
					ShortName: "Echo_Auth_Jwt",
					HttpRule: commonpb.Pattern{
						UriTemplate: "/auth/info/googlejwt",
						HttpMethod:  ut.GET,
					},
				},
			},
		},
		{
			desc: "Succeed for HTTP, with OPTIONS, and AllowCors",
			fakeServiceConfig: &conf.Service{
				Name: testProjectName,
				Apis: []*api.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*api.Method{
							{
								Name: "Echo",
							},
							{
								Name: "Echo_Auth",
							},
							{
								Name: "Echo_Auth_Jwt",
							},
							{
								Name: "EchoCors",
							},
						},
					},
				},
				Endpoints: []*conf.Endpoint{
					{
						Name:      testProjectName,
						AllowCors: true,
					},
				},
				Http: &annotations.Http{
					Rules: []*annotations.HttpRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.EchoCors",
							Pattern: &annotations.HttpRule_Custom{
								Custom: &annotations.CustomHttpPattern{
									Kind: "OPTIONS",
									Path: "/echo",
								},
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
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt",
							Pattern: &annotations.HttpRule_Get{
								Get: "/auth/info/googlejwt",
							},
						},
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth",
							Pattern: &annotations.HttpRule_Post{
								Post: "/auth/info/googlejwt",
							},
						},
					},
				},
			},
			backendProtocol: "http1",
			wantMethods: map[string]*methodInfo{
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.EchoCors": &methodInfo{
					ShortName: "EchoCors",
					HttpRule: commonpb.Pattern{
						UriTemplate: "/echo",
						HttpMethod:  ut.OPTIONS,
					},
				},
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo": &methodInfo{
					ShortName: "Echo",
					HttpRule: commonpb.Pattern{
						UriTemplate: "/echo",
						HttpMethod:  ut.POST,
					},
				},
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.CORS_0": &methodInfo{
					ShortName: "CORS_0",
					HttpRule: commonpb.Pattern{
						UriTemplate: "/auth/info/googlejwt",
						HttpMethod:  ut.OPTIONS,
					},
					IsGeneratedOption: true,
				},
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt": &methodInfo{
					ShortName: "Echo_Auth_Jwt",
					HttpRule: commonpb.Pattern{
						UriTemplate: "/auth/info/googlejwt",
						HttpMethod:  ut.GET,
					},
				},
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth": &methodInfo{
					ShortName: "Echo_Auth",
					HttpRule: commonpb.Pattern{
						UriTemplate: "/auth/info/googlejwt",
						HttpMethod:  ut.POST,
					},
				},
			},
		},
	}

	for i, tc := range testData {
		flag.Set("backend_protocol", tc.backendProtocol)
		serviceInfo, err := NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID)
		if err != nil {
			t.Fatal(err)
		}
		if len(serviceInfo.Methods) != len(tc.wantMethods) {
			t.Errorf("Test Desc(%d): %s, got Methods: %v, want: %v", i, tc.desc, serviceInfo.Methods, tc.wantMethods)
		}
		for key, gotMethod := range serviceInfo.Methods {
			wantMethod := tc.wantMethods[key]
			if eq := reflect.DeepEqual(gotMethod, wantMethod); !eq {
				t.Errorf("Test Desc(%d): %s, got Method: %v, want: %v", i, tc.desc, gotMethod, wantMethod)
			}
		}
	}
}

func TestProcessBackendRule(t *testing.T) {
	testData := []struct {
		desc              string
		fakeServiceConfig *conf.Service
		wantedAllowCors   bool
		wantedErr         string
	}{
		{
			desc: "Failed for dynamic routing only supports HTTPS",
			fakeServiceConfig: &conf.Service{
				Apis: []*api.Api{
					{
						Name: testApiName,
					},
				},
				Backend: &conf.Backend{
					Rules: []*conf.BackendRule{
						{
							Address:         "http://192.168.0.1/api/",
							PathTranslation: conf.BackendRule_CONSTANT_ADDRESS,
						},
					},
				},
			},
			wantedErr: "Failed for dynamic routing only supports HTTPS",
		},
		{
			desc: "Fail, dynamic routing only supports domain name, got IP address: 192.168.0.1",
			fakeServiceConfig: &conf.Service{
				Apis: []*api.Api{
					{
						Name: testApiName,
					},
				},
				Backend: &conf.Backend{
					Rules: []*conf.BackendRule{
						{
							Address:         "https://192.168.0.1/api/",
							PathTranslation: conf.BackendRule_CONSTANT_ADDRESS,
						},
					},
				},
			},
			wantedErr: "dynamic routing only supports domain name, got IP address: 192.168.0.1",
		},
	}

	for i, tc := range testData {
		flag.Set("backend_protocol", "grpc")
		flag.Set("enable_backend_routing", "true")
		_, err := NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID)
		if (err == nil && tc.wantedErr != "") || (err != nil && tc.wantedErr == "") {
			t.Errorf("Test Desc(%d): %s, extract backend address got: %v, want: %v", i, tc.desc, err, tc.wantedErr)
		}
	}
}
