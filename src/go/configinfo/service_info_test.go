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

package configinfo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/mux"
	"google.golang.org/protobuf/testing/protocmp"

	commonpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v11/http/common"
	scpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v11/http/service_control"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
	ptypepb "google.golang.org/genproto/protobuf/ptype"
)

var (
	testProjectName = "bookstore.endpoints.project123.cloud.goog"
	testApiName     = "endpoints.examples.bookstore.Bookstore"
	testConfigID    = "2019-03-02r0"
)

func TestProcessEndpoints(t *testing.T) {
	testData := []struct {
		desc              string
		fakeServiceConfig *confpb.Service
		wantedAllowCors   bool
	}{
		{
			desc: "Return true for endpoint name matching service name",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
					},
				},
				Endpoints: []*confpb.Endpoint{
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
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
					},
				},
				Endpoints: []*confpb.Endpoint{
					{
						Name: testProjectName,
					},
				},
			},
			wantedAllowCors: false,
		},
		{
			desc: "Return false for endpoint name not matching service name",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
					},
				},
				Endpoints: []*confpb.Endpoint{
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
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
					},
				},
			},
			wantedAllowCors: false,
		},
	}

	for i, tc := range testData {
		opts := options.DefaultConfigGeneratorOptions()
		serviceInfo, err := NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
		if err != nil {
			t.Fatal(err)
		}

		if serviceInfo.AllowCors != tc.wantedAllowCors {
			t.Errorf("Test Desc(%d): %s, allow CORS flag got: %v, want: %v", i, tc.desc, serviceInfo.AllowCors, tc.wantedAllowCors)
		}
	}
}

func TestProcessApiKeyLocations(t *testing.T) {
	testData := []struct {
		desc                                   string
		fakeServiceConfig                      *confpb.Service
		wantedSystemParameters                 map[string][]*confpb.SystemParameter
		wantedAllTranscodingIgnoredQueryParams map[string]bool
		wantMethods                            map[string]*MethodInfo
	}{
		{
			desc: "Succeed, only header",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "echo",
							},
						},
					},
				},
				SystemParameters: &confpb.SystemParameters{
					Rules: []*confpb.SystemParameterRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echo",
							Parameters: []*confpb.SystemParameter{
								{
									Name:       "api_key",
									HttpHeader: "header_name",
								},
							},
						},
					},
				},
			},
			wantedAllTranscodingIgnoredQueryParams: map[string]bool{},
			wantMethods: map[string]*MethodInfo{
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.echo": &MethodInfo{
					ShortName: "echo",
					ApiName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.POST,
							UriTemplate: parseUriTemplate("/1.echo_api_endpoints_cloudesf_testing_cloud_goog/echo"),
						},
					},
					ApiKeyLocations: []*scpb.ApiKeyLocation{
						{
							Key: &scpb.ApiKeyLocation_Header{
								Header: "header_name",
							},
						},
					},
				},
			},
		},
		{
			desc: "Succeed, only url query",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "echo",
							},
						},
					},
				},
				SystemParameters: &confpb.SystemParameters{
					Rules: []*confpb.SystemParameterRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echo",
							Parameters: []*confpb.SystemParameter{
								{
									Name:              "api_key",
									UrlQueryParameter: "query_name",
								},
							},
						},
					},
				},
			},
			wantedAllTranscodingIgnoredQueryParams: map[string]bool{
				"query_name": true,
			},
			wantMethods: map[string]*MethodInfo{
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.echo": &MethodInfo{
					ShortName: "echo",
					ApiName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.POST,
							UriTemplate: parseUriTemplate("/1.echo_api_endpoints_cloudesf_testing_cloud_goog/echo"),
						},
					},
					ApiKeyLocations: []*scpb.ApiKeyLocation{
						{
							Key: &scpb.ApiKeyLocation_Query{
								Query: "query_name",
							},
						},
					},
				},
			},
		},
		{
			desc: "Succeed, url query plus header",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "echo",
							},
						},
					},
				},
				SystemParameters: &confpb.SystemParameters{
					Rules: []*confpb.SystemParameterRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.echo",
							Parameters: []*confpb.SystemParameter{
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
			wantedAllTranscodingIgnoredQueryParams: map[string]bool{
				"query_name_1": true,
				"query_name_2": true,
			},
			wantMethods: map[string]*MethodInfo{
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.echo": &MethodInfo{
					ShortName: "echo",
					ApiName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.POST,
							UriTemplate: parseUriTemplate("/1.echo_api_endpoints_cloudesf_testing_cloud_goog/echo"),
						},
					},
					ApiKeyLocations: []*scpb.ApiKeyLocation{
						{
							Key: &scpb.ApiKeyLocation_Query{
								Query: "query_name_1",
							},
						},
						{
							Key: &scpb.ApiKeyLocation_Query{
								Query: "query_name_2",
							},
						},
						{
							Key: &scpb.ApiKeyLocation_Header{
								Header: "header_name_1",
							},
						},
						{
							Key: &scpb.ApiKeyLocation_Header{
								Header: "header_name_2",
							},
						},
					},
				},
			},
		},

		{
			desc: "Succeed, url query plus header for multiple apis with one using default ApiKeyLocation",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "foo",
							},
						},
					},
					{
						Name: "2.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "bar",
							},
						},
					},
					{
						Name: "3.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "baz",
							},
						},
					},
				},
				SystemParameters: &confpb.SystemParameters{
					Rules: []*confpb.SystemParameterRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.foo",
							Parameters: []*confpb.SystemParameter{
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
						{
							Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.bar",
							Parameters: []*confpb.SystemParameter{
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
			wantedAllTranscodingIgnoredQueryParams: map[string]bool{
				"api_key":      true,
				"key":          true,
				"query_name_1": true,
				"query_name_2": true,
			},
			wantMethods: map[string]*MethodInfo{
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.foo": {
					ShortName: "foo",
					ApiName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.POST,
							UriTemplate: parseUriTemplate("/1.echo_api_endpoints_cloudesf_testing_cloud_goog/foo"),
						},
					},
					ApiKeyLocations: []*scpb.ApiKeyLocation{
						{
							Key: &scpb.ApiKeyLocation_Query{
								Query: "query_name_1",
							},
						},
						{
							Key: &scpb.ApiKeyLocation_Query{
								Query: "query_name_2",
							},
						},
						{
							Key: &scpb.ApiKeyLocation_Header{
								Header: "header_name_1",
							},
						},
						{
							Key: &scpb.ApiKeyLocation_Header{
								Header: "header_name_2",
							},
						},
					},
				},

				"2.echo_api_endpoints_cloudesf_testing_cloud_goog.bar": {
					ShortName: "bar",
					ApiName:   "2.echo_api_endpoints_cloudesf_testing_cloud_goog",
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.POST,
							UriTemplate: parseUriTemplate("/2.echo_api_endpoints_cloudesf_testing_cloud_goog/bar"),
						},
					},
					ApiKeyLocations: []*scpb.ApiKeyLocation{
						{
							Key: &scpb.ApiKeyLocation_Query{
								Query: "query_name_1",
							},
						},
						{
							Key: &scpb.ApiKeyLocation_Query{
								Query: "query_name_2",
							},
						},
						{
							Key: &scpb.ApiKeyLocation_Header{
								Header: "header_name_1",
							},
						},
						{
							Key: &scpb.ApiKeyLocation_Header{
								Header: "header_name_2",
							},
						},
					},
				},
				"3.echo_api_endpoints_cloudesf_testing_cloud_goog.baz": {
					ShortName: "baz",
					ApiName:   "3.echo_api_endpoints_cloudesf_testing_cloud_goog",
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.POST,
							UriTemplate: parseUriTemplate("/3.echo_api_endpoints_cloudesf_testing_cloud_goog/baz"),
						},
					},
				},
			},
		},
	}
	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			opts := options.DefaultConfigGeneratorOptions()
			opts.BackendAddress = "grpc://127.0.0.1:80"
			serviceInfo, err := NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
			if err != nil {
				t.Fatal(err)
			}
			if len(serviceInfo.Methods) != len(tc.wantMethods) {
				t.Fatalf("Num methods mismatch \ngot : %v, want: %v", serviceInfo.Methods, tc.wantMethods)
			}
			if !reflect.DeepEqual(serviceInfo.AllTranscodingIgnoredQueryParams, tc.wantedAllTranscodingIgnoredQueryParams) {
				t.Fatalf("AllTranscodingIgnoredQueryParams mismatch \ngot : %v, want: %v", serviceInfo.AllTranscodingIgnoredQueryParams, tc.wantedAllTranscodingIgnoredQueryParams)
			}

			for key, gotMethod := range serviceInfo.Methods {
				wantMethod := tc.wantMethods[key]

				// We're not testing backend info here.
				gotMethod.BackendInfo = nil

				if eq := cmp.Equal(gotMethod, wantMethod, cmp.Comparer(proto.Equal)); !eq {
					t.Errorf("Method mistmatch \ngot : %+v,\nwant: %+v", gotMethod, wantMethod)
				}
			}
		})
	}
}

func TestProcessTranscodingIgnoredQueryParams(t *testing.T) {
	testData := []struct {
		desc                                   string
		fakeServiceConfig                      *confpb.Service
		transcodingIgnoredQueryParamsFlag      string
		wantedAllTranscodingIgnoredQueryParams map[string]bool
		wantedErrorPrefix                      string
	}{
		{
			desc: "Success. Default jwt locations with transcoding_ignore_query_params flag",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "echo",
							},
						},
					},
				},
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:     "auth_provider",
							Issuer: "issuer-0",
						},
					},
				},
			},
			transcodingIgnoredQueryParamsFlag: "foo,bar",
			wantedAllTranscodingIgnoredQueryParams: map[string]bool{
				"access_token": true,
				"foo":          true,
				"bar":          true,
			},
		},
		{
			desc: "Failure. Wrong jwt locations setting Query with valuePrefix in the same time",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "echo",
							},
						},
					},
				},
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:     "auth_provider",
							Issuer: "issuer-0",
							JwtLocations: []*confpb.JwtLocation{
								{
									In: &confpb.JwtLocation_Query{
										Query: "jwt_query_param",
									},
									ValuePrefix: "jwt_query_header_prefix",
								},
							},
						},
					},
				},
			},
			wantedErrorPrefix: `error processing authentication provider (auth_provider): JwtLocation type [Query] should be set without valuePrefix, but it was set to [jwt_query_header_prefix]`,
		},
		{
			desc: "Success. Custom jwt locations with transcoding_ignore_query_params flag",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "echo",
							},
						},
					},
				},
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:     "auth_provider",
							Issuer: "issuer-0",
							JwtLocations: []*confpb.JwtLocation{
								{
									In: &confpb.JwtLocation_Header{
										Header: "jwt_query_header",
									},
									ValuePrefix: "jwt_query_header_prefix",
								},
								{
									In: &confpb.JwtLocation_Query{
										Query: "jwt_query_param",
									},
								},
							},
						},
					},
				},
			},
			transcodingIgnoredQueryParamsFlag: "foo,bar",
			wantedAllTranscodingIgnoredQueryParams: map[string]bool{
				"jwt_query_param": true,
				"foo":             true,
				"bar":             true,
			},
		},
	}
	for i, tc := range testData {

		opts := options.DefaultConfigGeneratorOptions()
		opts.TranscodingIgnoreQueryParameters = tc.transcodingIgnoredQueryParamsFlag
		serviceInfo := &ServiceInfo{
			serviceConfig:                    tc.fakeServiceConfig,
			Methods:                          make(map[string]*MethodInfo),
			AllTranscodingIgnoredQueryParams: make(map[string]bool),
			Options:                          opts,
		}

		err := serviceInfo.processTranscodingIgnoredQueryParams()
		if err != nil {
			if !strings.HasPrefix(err.Error(), tc.wantedErrorPrefix) {
				// Error doesn't match with wantedError.
				t.Errorf("Test Desc(%d): %s, gotError: %v, wantedErrorPrefix: %v", i, tc.desc, err.Error(), tc.wantedErrorPrefix)
			}

		} else if tc.wantedErrorPrefix != "" {
			// Error is empty while wantedError is not.
			t.Errorf("Test Desc(%d): %s, gotError: %v, wantedErrorPrefix: %v", i, tc.desc, err.Error(), tc.wantedErrorPrefix)

		} else if !reflect.DeepEqual(serviceInfo.AllTranscodingIgnoredQueryParams, tc.wantedAllTranscodingIgnoredQueryParams) {
			// Generated TranscoderIgnoreApiKeyQueryParams is not expected.
			t.Errorf("Test Desc(%d): %s, gotAllTranscodingIgnoredQueryParams: %v, wantedAllTranscodingIgnoredQueryParams: %v", i, tc.desc, serviceInfo.AllTranscodingIgnoredQueryParams, tc.wantedAllTranscodingIgnoredQueryParams)
		}
	}
}

func TestMethods(t *testing.T) {
	testData := []struct {
		desc                         string
		fakeServiceConfig            *confpb.Service
		BackendAddress               string
		healthz                      string
		enableBackendAddressOverride bool
		isNonGcp                     bool
		wantMethods                  map[string]*MethodInfo
		wantError                    string
	}{
		{
			desc: "Succeed for gRPC, no Http rule, with Healthz",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
						Methods: []*apipb.Method{
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
			BackendAddress: "grpc://127.0.0.1:80",
			healthz:        "/",
			wantMethods: map[string]*MethodInfo{
				fmt.Sprintf("%s.%s", testApiName, "ListShelves"): &MethodInfo{
					ShortName: "ListShelves",
					ApiName:   testApiName,
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.POST,
							UriTemplate: parseUriTemplate(fmt.Sprintf("/%s/%s", testApiName, "ListShelves")),
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
				fmt.Sprintf("%s.%s", testApiName, "CreateShelf"): &MethodInfo{
					ShortName: "CreateShelf",
					ApiName:   testApiName,
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.POST,
							UriTemplate: parseUriTemplate(fmt.Sprintf("/%s/%s", testApiName, "CreateShelf")),
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
				fmt.Sprintf("%s.%s", "espv2_deployment", "ESPv2_Autogenerated_HealthCheck"): &MethodInfo{
					ShortName:          "ESPv2_Autogenerated_HealthCheck",
					ApiName:            "espv2_deployment",
					SkipServiceControl: true,
					IsGenerated:        true,
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.GET,
							UriTemplate: parseUriTemplate("/"),
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
			},
		},
		{
			desc: "Succeed for HTTP, with Healthz; skip discovery API",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "Echo",
							},
							{
								Name: "Echo_Auth_Jwt",
							},
						},
					},
					{
						Name: "google.discovery.Discovery",
						Methods: []*apipb.Method{
							{
								Name: "GetDiscovery",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/auth/info/googlejwt",
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
							Selector: "google.discovery.Discovery.GetDiscover",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/$discovery",
							},
						},
					},
				},
			},
			BackendAddress: "http://127.0.0.1:80",
			healthz:        "/",
			wantMethods: map[string]*MethodInfo{
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo": &MethodInfo{
					ShortName: "Echo",
					ApiName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.POST,
							UriTemplate: parseUriTemplate("/echo"),
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt": &MethodInfo{
					ShortName: "Echo_Auth_Jwt",
					ApiName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.GET,
							UriTemplate: parseUriTemplate("/auth/info/googlejwt"),
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
				"espv2_deployment.ESPv2_Autogenerated_HealthCheck": &MethodInfo{
					ShortName:          "ESPv2_Autogenerated_HealthCheck",
					ApiName:            "espv2_deployment",
					SkipServiceControl: true,
					IsGenerated:        true,
					HttpRule: []*httppattern.Pattern{
						{
							UriTemplate: parseUriTemplate("/"),
							HttpMethod:  util.GET,
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
			},
		},
		{
			desc: "Succeed for HTTP with multiple apis",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "Echo",
							},
							{
								Name: "Echo_Auth_Jwt",
							},
						},
					},
					{
						Name: "2.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "Echo",
							},
							{
								Name: "Echo_Auth_Jwt",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/1/auth/info/googlejwt",
							},
						},
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
							Pattern: &annotationspb.HttpRule_Post{
								Post: "/1/echo",
							},
							Body: "message",
						},
						{
							Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/2/auth/info/googlejwt",
							},
						},
						{
							Selector: "2.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
							Pattern: &annotationspb.HttpRule_Post{
								Post: "/2/echo",
							},
							Body: "message",
						},
					},
				},
			},
			BackendAddress: "http://127.0.0.1:80",
			wantMethods: map[string]*MethodInfo{
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo": &MethodInfo{
					ShortName: "Echo",
					ApiName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					HttpRule: []*httppattern.Pattern{
						{
							UriTemplate: parseUriTemplate("/1/echo"),
							HttpMethod:  util.POST,
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt": &MethodInfo{
					ShortName: "Echo_Auth_Jwt",
					ApiName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					HttpRule: []*httppattern.Pattern{
						{
							UriTemplate: parseUriTemplate("/1/auth/info/googlejwt"),
							HttpMethod:  util.GET,
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
				"2.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo": &MethodInfo{
					ShortName: "Echo",
					ApiName:   "2.echo_api_endpoints_cloudesf_testing_cloud_goog",
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.POST,
							UriTemplate: parseUriTemplate("/2/echo"),
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
				"2.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt": &MethodInfo{
					ShortName: "Echo_Auth_Jwt",
					ApiName:   "2.echo_api_endpoints_cloudesf_testing_cloud_goog",
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.GET,
							UriTemplate: parseUriTemplate("/2/auth/info/googlejwt"),
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
			},
		},
		{
			desc: "Succeed for HTTP, with OPTIONS, and AllowCors, with Healthz",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Version: "1.2.3",
						Methods: []*apipb.Method{
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
							{
								Name: "GetBook",
							},
							{
								Name: "UpdateBook",
							},
							{
								Name: "DeleteBook",
							},
						},
					},
				},
				Endpoints: []*confpb.Endpoint{
					{
						Name:      testProjectName,
						AllowCors: true,
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.EchoCors",
							Pattern: &annotationspb.HttpRule_Custom{
								Custom: &annotationspb.CustomHttpPattern{
									Kind: "OPTIONS",
									Path: "/echo",
								},
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
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/auth/info/googlejwt",
							},
						},
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth",
							Pattern: &annotationspb.HttpRule_Post{
								Post: "/auth/info/googlejwt",
							},
						},
						{
							// This will also have a CORS method generated for it.
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.GetBook",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/shelves/{shelf_id}/books/{book.id}",
							},
						},
						{
							// No CORS method generated due to an equivalent path on different HTTP method.
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.UpdateBook",
							Pattern: &annotationspb.HttpRule_Patch{
								Patch: "/shelves/{shelf_id}/books/{book.id}",
							},
						},
						{
							// No CORS method generated due to a **semantically** equivalent path on different HTTP method.
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.DeleteBook",
							Pattern: &annotationspb.HttpRule_Delete{
								Delete: "/shelves/{shelf_id_different_param}/books/{book.id}",
							},
						},
					},
				},
			},
			BackendAddress: "http://127.0.0.1:80",
			healthz:        "/healthz",
			wantMethods: map[string]*MethodInfo{
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.EchoCors": &MethodInfo{
					ShortName:  "EchoCors",
					ApiName:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiVersion: "1.2.3",
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.OPTIONS,
							UriTemplate: parseUriTemplate("/echo"),
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo": &MethodInfo{
					ShortName:  "Echo",
					ApiName:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiVersion: "1.2.3",
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.POST,
							UriTemplate: parseUriTemplate("/echo"),
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.GetBook": &MethodInfo{
					ShortName:  "GetBook",
					ApiName:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiVersion: "1.2.3",
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.GET,
							UriTemplate: parseUriTemplate("/shelves/{shelf_id}/books/{book.id}"),
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.ESPv2_Autogenerated_CORS_GetBook": &MethodInfo{
					ShortName:  "ESPv2_Autogenerated_CORS_GetBook",
					ApiVersion: "1.2.3",
					ApiName:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.OPTIONS,
							UriTemplate: parseUriTemplate("/shelves/{shelf_id}/books/{book.id}"),
						},
					},
					IsGenerated: true,
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.UpdateBook": &MethodInfo{
					ShortName:  "UpdateBook",
					ApiName:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiVersion: "1.2.3",
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.PATCH,
							UriTemplate: parseUriTemplate("/shelves/{shelf_id}/books/{book.id}"),
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.DeleteBook": &MethodInfo{
					ShortName:  "DeleteBook",
					ApiName:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiVersion: "1.2.3",
					HttpRule: []*httppattern.Pattern{
						{
							UriTemplate: parseUriTemplate("/shelves/{shelf_id_different_param}/books/{book.id}"),
							HttpMethod:  util.DELETE,
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.ESPv2_Autogenerated_CORS_Echo_Auth_Jwt": &MethodInfo{
					ShortName:  "ESPv2_Autogenerated_CORS_Echo_Auth_Jwt",
					ApiName:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiVersion: "1.2.3",
					HttpRule: []*httppattern.Pattern{
						{
							UriTemplate: parseUriTemplate("/auth/info/googlejwt"),
							HttpMethod:  util.OPTIONS,
						},
					},
					IsGenerated: true,
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth_Jwt": &MethodInfo{
					ShortName:  "Echo_Auth_Jwt",
					ApiName:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiVersion: "1.2.3",
					HttpRule: []*httppattern.Pattern{
						{
							UriTemplate: parseUriTemplate("/auth/info/googlejwt"),
							HttpMethod:  util.GET,
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo_Auth": &MethodInfo{
					ShortName:  "Echo_Auth",
					ApiName:    "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					ApiVersion: "1.2.3",
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.POST,
							UriTemplate: parseUriTemplate("/auth/info/googlejwt"),
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
				"espv2_deployment.ESPv2_Autogenerated_HealthCheck": &MethodInfo{
					ShortName:          "ESPv2_Autogenerated_HealthCheck",
					ApiName:            "espv2_deployment",
					SkipServiceControl: true,
					IsGenerated:        true,
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.GET,
							UriTemplate: parseUriTemplate("/healthz"),
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
			},
		},
		{
			desc: "Succeed for HTTP with custom method and allow CORS",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "EchoCustom",
							},
						},
					},
				},
				Endpoints: []*confpb.Endpoint{
					{
						Name:      testProjectName,
						AllowCors: true,
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.EchoCustom",
							Pattern: &annotationspb.HttpRule_Custom{
								Custom: &annotationspb.CustomHttpPattern{
									Kind: "CustomMethod",
									Path: "/echo/test",
								},
							},
						},
					},
				},
			},
			BackendAddress: "http://127.0.0.1:80",
			wantMethods: map[string]*MethodInfo{
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.EchoCustom": &MethodInfo{
					ShortName: "EchoCustom",
					ApiName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  "CustomMethod",
							UriTemplate: parseUriTemplate("/echo/test"),
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
				// CORS method generated for the above path.
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.ESPv2_Autogenerated_CORS_EchoCustom": &MethodInfo{
					ShortName: "ESPv2_Autogenerated_CORS_EchoCustom",
					ApiName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.OPTIONS,
							UriTemplate: parseUriTemplate("/echo/test"),
						},
					},
					IsGenerated: true,
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
			},
		},
		{
			desc: "Succeed for HTTP with semantically equivalent additional bindings with AllowCors",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name: "Echo",
							},
						},
					},
				},
				Endpoints: []*confpb.Endpoint{
					{
						Name:      testProjectName,
						AllowCors: true,
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/echo/{test_one}/echo",
							},
							AdditionalBindings: []*annotationspb.HttpRule{
								{
									// Semantically equivalent path with different HTTP Method, should not be used when generating CORS http rules.
									Pattern: &annotationspb.HttpRule_Post{
										Post: "/echo/{test_two}/echo",
									},
								},
								{
									Pattern: &annotationspb.HttpRule_Delete{
										Delete: "/echo/{test_three}/a/b/c",
									},
								},
							},
						},
					},
				},
			},
			BackendAddress: "http://127.0.0.1:80",
			wantMethods: map[string]*MethodInfo{
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo": &MethodInfo{
					ShortName: "Echo",
					ApiName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					HttpRule: []*httppattern.Pattern{
						{
							UriTemplate: parseUriTemplate("/echo/{test_one}/echo"),
							HttpMethod:  util.GET,
						},
						{
							UriTemplate: parseUriTemplate("/echo/{test_two}/echo"),
							HttpMethod:  util.POST,
						},
						{
							UriTemplate: parseUriTemplate("/echo/{test_three}/a/b/c"),
							HttpMethod:  util.DELETE,
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
				// CORS method generated for the above method.
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.ESPv2_Autogenerated_CORS_Echo": &MethodInfo{
					ShortName: "ESPv2_Autogenerated_CORS_Echo",
					ApiName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					HttpRule: []*httppattern.Pattern{
						{
							UriTemplate: parseUriTemplate("/echo/{test_one}/echo"),
							HttpMethod:  util.OPTIONS,
						},
						// 2nd pattern should be missing.
						{
							UriTemplate: parseUriTemplate("/echo/{test_three}/a/b/c"),
							HttpMethod:  util.OPTIONS,
						},
					},
					IsGenerated: true,
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
			},
		},
		{
			desc: "Succeed for replacing snakeName with jsonName in autogen cors method's url template",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name:           "Echo",
								RequestTypeUrl: "type.googleapis.com/endpoints.examples.echo.EchoRequest",
							},
						},
					},
				},
				Endpoints: []*confpb.Endpoint{
					{
						Name:      testProjectName,
						AllowCors: true,
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "/echo/{test_one}/echo",
							},
						},
					},
				},
				Types: []*ptypepb.Type{
					{
						Name: "endpoints.examples.echo.EchoRequest",
						Fields: []*ptypepb.Field{
							{
								Name:     "test_one",
								JsonName: "testOne",
							},
						},
					},
				},
			},
			BackendAddress: "http://127.0.0.1:80",
			wantMethods: map[string]*MethodInfo{
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo": &MethodInfo{
					ShortName: "Echo",
					ApiName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					HttpRule: []*httppattern.Pattern{
						{
							UriTemplate: parseUriTemplate("/echo/{testOne=*}/echo"),
							HttpMethod:  util.GET,
						},
					},
					RequestTypeName: "endpoints.examples.echo.EchoRequest",
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
				// CORS method generated for the above method.
				"1.echo_api_endpoints_cloudesf_testing_cloud_goog.ESPv2_Autogenerated_CORS_Echo": &MethodInfo{
					ShortName: "ESPv2_Autogenerated_CORS_Echo",
					ApiName:   "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
					HttpRule: []*httppattern.Pattern{
						{
							UriTemplate: parseUriTemplate("/echo/{testOne=*}/echo"),
							HttpMethod:  util.OPTIONS,
						},
					},
					IsGenerated: true,
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
			},
		},
		{
			desc: "fail to processHttpRules due to invalid url template",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
						Methods: []*apipb.Method{
							{
								Name:           "Echo",
								RequestTypeUrl: "type.googleapis.com/endpoints.examples.echo.EchoRequest",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
							Pattern: &annotationspb.HttpRule_Get{
								Get: "invalid-uri-template",
							},
						},
					},
				},
				Types: []*ptypepb.Type{
					{
						Name: "endpoints.examples.echo.EchoRequest",
						Fields: []*ptypepb.Field{
							{
								Name:     "test_one",
								JsonName: "testOne",
							},
						},
					},
				},
			},
			BackendAddress: "http://127.0.0.1:80",
			wantError:      "(1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo): invalid uri template invalid-uri-template",
		},
		{
			desc: "Succeed for multiple url Pattern",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name:            "CreateBook",
								RequestTypeUrl:  "type.googleapis.com/endpoints.examples.bookstore.CreateBookRequest",
								ResponseTypeUrl: "type.googleapis.com/endpoints.examples.bookstore.Book",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.CreateBook",
							Pattern: &annotationspb.HttpRule_Post{
								Post: "/v1/shelves/{shelf}/books/{book.id}/{book.author}",
							},
							Body: "book.title",
						},
						{
							Selector: "endpoints.examples.bookstore.Bookstore.CreateBook",
							Pattern: &annotationspb.HttpRule_Post{
								Post: "/v1/shelves/{shelf}/books",
							},
							Body: "book",
						},
					},
				},
				Types: []*ptypepb.Type{
					{
						Name: "endpoints.examples.bookstore.CreateBookRequest",
					},
				},
			},
			BackendAddress: "grpc://127.0.0.1:80",
			wantMethods: map[string]*MethodInfo{
				"endpoints.examples.bookstore.Bookstore.CreateBook": &MethodInfo{
					ShortName:       "CreateBook",
					ApiName:         "endpoints.examples.bookstore.Bookstore",
					RequestTypeName: "endpoints.examples.bookstore.CreateBookRequest",
					HttpRule: []*httppattern.Pattern{
						{
							UriTemplate: parseUriTemplate("/v1/shelves/{shelf}/books/{book.id}/{book.author}"),
							HttpMethod:  util.POST,
						},
						{
							UriTemplate: parseUriTemplate("/v1/shelves/{shelf}/books"),
							HttpMethod:  util.POST,
						},
						{
							UriTemplate: parseUriTemplate("/endpoints.examples.bookstore.Bookstore/CreateBook"),
							HttpMethod:  util.POST,
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
			},
		},
		{
			desc: "Succeed for additional binding",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "endpoints.examples.bookstore.Bookstore",
						Methods: []*apipb.Method{
							{
								Name:            "CreateBook",
								RequestTypeUrl:  "type.googleapis.com/endpoints.examples.bookstore.CreateBookRequest",
								ResponseTypeUrl: "type.googleapis.com/endpoints.examples.bookstore.Book",
							},
						},
					},
				},
				Http: &annotationspb.Http{
					Rules: []*annotationspb.HttpRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.CreateBook",
							Pattern: &annotationspb.HttpRule_Post{
								Post: "/v1/shelves/{shelf}/books/{book.id}/{book.author}",
							},
							Body: "book.title",
							AdditionalBindings: []*annotationspb.HttpRule{
								{
									Pattern: &annotationspb.HttpRule_Post{
										Post: "/v1/shelves/{shelf}/books/foo",
									},
									Body: "book",
								},
								{
									Pattern: &annotationspb.HttpRule_Post{
										Post: "/v1/shelves/{shelf}/books/bar",
									},
									Body: "book",
								},
							},
						},
					},
				},
				Types: []*ptypepb.Type{
					{
						Name: "endpoints.examples.bookstore.CreateBookRequest",
					},
				},
			},
			BackendAddress: "grpc://127.0.0.1:80",
			wantMethods: map[string]*MethodInfo{
				"endpoints.examples.bookstore.Bookstore.CreateBook": &MethodInfo{
					ShortName:       "CreateBook",
					ApiName:         "endpoints.examples.bookstore.Bookstore",
					RequestTypeName: "endpoints.examples.bookstore.CreateBookRequest",
					HttpRule: []*httppattern.Pattern{
						{
							UriTemplate: parseUriTemplate("/v1/shelves/{shelf}/books/{book.id}/{book.author}"),
							HttpMethod:  util.POST,
						},
						{
							UriTemplate: parseUriTemplate("/v1/shelves/{shelf}/books/foo"),
							HttpMethod:  util.POST,
						},
						{
							UriTemplate: parseUriTemplate("/v1/shelves/{shelf}/books/bar"),
							HttpMethod:  util.POST,
						},
						{
							UriTemplate: parseUriTemplate("/endpoints.examples.bookstore.Bookstore/CreateBook"),
							HttpMethod:  util.POST,
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
					},
				},
			},
		},
		{
			desc:                         "Backend address override with non GCP results in local backend used instead of backend rule.",
			enableBackendAddressOverride: true,
			isNonGcp:                     true,
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: testApiName,
						Methods: []*apipb.Method{
							{
								Name: "ListShelves",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:         "grpc://abc.com/a/",
							Selector:        fmt.Sprintf("%s.%s", testApiName, "ListShelves"),
							PathTranslation: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
							Authentication: &confpb.BackendRule_JwtAudience{
								JwtAudience: "grpc://abc.com/a/",
							},
						},
					},
				},
			},
			BackendAddress: "grpc://127.0.0.1:80",
			wantMethods: map[string]*MethodInfo{
				fmt.Sprintf("%s.%s", testApiName, "ListShelves"): &MethodInfo{
					ShortName: "ListShelves",
					ApiName:   testApiName,
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.POST,
							UriTemplate: parseUriTemplate(fmt.Sprintf("/%s/%s", testApiName, "ListShelves")),
						},
					},
					BackendInfo: &backendInfo{
						ClusterName: "backend-cluster-bookstore.endpoints.project123.cloud.goog_local",
						Deadline:    util.DefaultResponseDeadline,
						RetryOns:    "reset,connect-failure,refused-stream",
						RetryNum:    1,
						// Even though translation type is specified, it won't matter: Path is empty.
						TranslationType: confpb.BackendRule_APPEND_PATH_TO_ADDRESS,
					},
				},
			},
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			opts := options.DefaultConfigGeneratorOptions()
			opts.BackendAddress = tc.BackendAddress
			opts.Healthz = tc.healthz
			opts.EnableBackendAddressOverride = tc.enableBackendAddressOverride
			opts.NonGCP = tc.isNonGcp
			serviceInfo, err := NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("Error mismatch \ngot : %v, \nwant: %v", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error \ngot err %v", err)
			}
			if len(serviceInfo.Methods) != len(tc.wantMethods) {
				t.Fatalf("diff in number of Methods,\ngot : %v,\nwant: %v", serviceInfo.Methods, tc.wantMethods)
			}
			for key, gotMethod := range serviceInfo.Methods {
				wantMethod, ok := tc.wantMethods[key]

				// Remove some items we have other specific tests for.
				gotMethod.GeneratedCorsMethod = nil
				gotMethod.BackendInfo.IdleTimeout = 0

				if !ok {
					t.Errorf("cannot find key: %v\n got methods : %+v\nwant methods: %+v", key, serviceInfo.Methods, tc.wantMethods)
				} else if diff := cmp.Diff(gotMethod, wantMethod, protocmp.Transform()); diff != "" {
					t.Errorf("methods mismtatch: %v", diff)
				}
			}
		})
	}
}

func TestProcessBackendRuleForHTTPBackend(t *testing.T) {
	testData := []struct {
		desc              string
		fakeServiceConfig *confpb.Service
		// Map of selector to the expected deadline for the corresponding route.
		wantErrString string
	}{
		{
			desc: "Wrong protocol scheme caught for HTTP backend",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://abc.com/api/",
							Selector: "abc.com.api",
							Deadline: 10.5,
							OverridesByRequestProtocol: map[string]*confpb.BackendRule{
								"http": &confpb.BackendRule{
									Address:  "grpc://http.abc.com/api/",
									Deadline: 20.5,
								},
							},
						},
					},
				},
			},
			wantErrString: "gRPC protocol conflicted with http backend",
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			opts := options.DefaultConfigGeneratorOptions()
			_, err := NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)

			if tc.wantErrString != "" {
				if err == nil {
					t.Errorf("Service info got error nil; want non-nil error containing string %q", tc.wantErrString)
				} else if !strings.Contains(err.Error(), tc.wantErrString) {
					t.Errorf("Service info got error %v; want error containing string %q", err, tc.wantErrString)
				}
			} else if err != nil {
				t.Errorf("Service info got error %v; want nil error", err)
			}
		})
	}
}

func TestProcessBackendRuleForDeadline(t *testing.T) {
	testData := []struct {
		desc              string
		fakeServiceConfig *confpb.Service
		// Map of selector to the expected deadline for the corresponding route.
		wantedMethodDeadlines      map[string]time.Duration
		wantedHTTPBackendDeadlines map[string]time.Duration
	}{
		{
			desc: "Mixed deadlines across multiple backend rules",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
					{
						Name: "cnn.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://abc.com/api/",
							Selector: "abc.com.api",
							Deadline: 10.5,
							OverridesByRequestProtocol: map[string]*confpb.BackendRule{
								"http": &confpb.BackendRule{
									Address:  "http://http.abc.com/api/",
									Deadline: 20.5,
								},
							},
						},
						{
							Address:  "grpc://cnn.com/api/",
							Selector: "cnn.com.api",
							Deadline: 20,
						},
					},
				},
			},
			wantedMethodDeadlines: map[string]time.Duration{
				"abc.com.api": 10*time.Second + 500*time.Millisecond,
				"cnn.com.api": 20 * time.Second,
			},
			wantedHTTPBackendDeadlines: map[string]time.Duration{
				"abc.com.api": 20*time.Second + 500*time.Millisecond,
			},
		},
		{
			desc: "Deadline with high precision is rounded to milliseconds",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://abc.com/api/",
							Selector: "abc.com.api",
							Deadline: 30.0009, // 30s 0.9ms
						},
					},
				},
			},
			wantedMethodDeadlines: map[string]time.Duration{
				"abc.com.api": 30*time.Second + 1*time.Millisecond,
			},
		},
		{
			desc: "Deadline that is non-positive is overridden to default",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://abc.com/api/",
							Selector: "abc.com.api",
							Deadline: -10.5,
						},
					},
				},
			},
			wantedMethodDeadlines: map[string]time.Duration{
				"abc.com.api": util.DefaultResponseDeadline,
			},
		},
		{
			desc: "Missing deadline is defaulted",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
			},
			wantedMethodDeadlines: map[string]time.Duration{
				"abc.com.api": util.DefaultResponseDeadline,
			},
		},
		{
			desc: "Streaming methods have no deadline",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name:              "api",
								ResponseStreaming: true,
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://abc.com/api/",
							Selector: "abc.com.api",
							Deadline: 10.5,
						},
					},
				},
			},
			wantedMethodDeadlines: map[string]time.Duration{
				"abc.com.api": 0,
			},
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			opts := options.DefaultConfigGeneratorOptions()
			s, err := NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)

			if err != nil {
				t.Errorf("error not expected, got: %v", err)
				return
			}

			for operation, mi := range s.Methods {
				gotDeadline := mi.BackendInfo.Deadline
				wantDeadline := tc.wantedMethodDeadlines[operation]

				if wantDeadline != gotDeadline {
					t.Errorf("Deadline not expected, got: %v, want: %v", gotDeadline, wantDeadline)
				}

				if wantedHTTPBackendDeadlines, ok := tc.wantedHTTPBackendDeadlines[operation]; !ok {
					continue
				} else {
					if mi.HttpBackendInfo == nil {
						t.Fatalf("HTTP backend info is nil; want non-nil because the http backend rule is specified.")
					}
					gotHTTPBackendDeadline := mi.HttpBackendInfo.Deadline
					if wantedHTTPBackendDeadlines != gotHTTPBackendDeadline {
						t.Errorf("Deadline not expected, got: %v, want: %v", gotHTTPBackendDeadline, wantDeadline)
					}
				}
			}
		})
	}
}

func TestProcessBackendRuleForIdleTimeout(t *testing.T) {
	testData := []struct {
		desc              string
		fakeServiceConfig *confpb.Service
		globalIdleTimeout time.Duration
		// Map of selector to the expected idle timeout for the corresponding route.
		wantedMethodIdleTimeout      map[string]time.Duration
		wantedHTTPBackendIdleTimeout map[string]time.Duration
	}{
		{
			desc:              "Global idle timeout takes priority over small deadline",
			globalIdleTimeout: util.DefaultIdleTimeout,
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://abc.com/api/",
							Selector: "abc.com.api",
							Deadline: 10.5,
							OverridesByRequestProtocol: map[string]*confpb.BackendRule{
								"http": &confpb.BackendRule{
									Address:  "http://abc.com/api/",
									Selector: "abc.com.api.by.override",
									Deadline: 11.5,
								},
							},
						},
					},
				},
			},
			wantedMethodIdleTimeout: map[string]time.Duration{
				"abc.com.api": util.DefaultIdleTimeout,
			},
			wantedHTTPBackendIdleTimeout: map[string]time.Duration{
				"abc.com.api": util.DefaultIdleTimeout,
			},
		},
		{
			desc:              "Deadline takes priority over small global idle timeout",
			globalIdleTimeout: 7 * time.Second,
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://abc.com/api/",
							Selector: "abc.com.api",
							Deadline: 10.5,
						},
					},
				},
			},
			wantedMethodIdleTimeout: map[string]time.Duration{
				"abc.com.api": 10*time.Second + 500*time.Millisecond + time.Second,
			},
		},
		{
			desc:              "Global idle timeout takes priority over missing deadline",
			globalIdleTimeout: 30 * time.Second,
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
			},
			wantedMethodIdleTimeout: map[string]time.Duration{
				"abc.com.api": 30 * time.Second,
			},
		},
		{
			desc:              "Global idle timeout takes priority over negative deadline",
			globalIdleTimeout: util.DefaultIdleTimeout,
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://abc.com/api/",
							Selector: "abc.com.api",
							Deadline: -10.5,
						},
					},
				},
			},
			wantedMethodIdleTimeout: map[string]time.Duration{
				"abc.com.api": util.DefaultIdleTimeout,
			},
		},
		{
			desc:              "Default deadline takes priority over small global idle timeout with missing deadline",
			globalIdleTimeout: 7 * time.Second,
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
			},
			wantedMethodIdleTimeout: map[string]time.Duration{
				"abc.com.api": util.DefaultResponseDeadline + time.Second,
			},
		},
		{
			desc:              "Default deadline takes priority over small global idle timeout and negative deadline",
			globalIdleTimeout: 7 * time.Second,
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://abc.com/api/",
							Selector: "abc.com.api",
							Deadline: -10.5,
						},
					},
				},
			},
			wantedMethodIdleTimeout: map[string]time.Duration{
				"abc.com.api": util.DefaultResponseDeadline + time.Second,
			},
		},
		{
			desc:              "Streaming methods set the idle timeout directly from the deadline, even if the global stream idle timeout is larger.",
			globalIdleTimeout: util.DefaultIdleTimeout,
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name:             "api",
								RequestStreaming: true,
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://abc.com/api/",
							Selector: "abc.com.api",
							Deadline: 10.5,
						},
					},
				},
			},
			wantedMethodIdleTimeout: map[string]time.Duration{
				"abc.com.api": 10*time.Second + 500*time.Millisecond,
			},
		},
		{
			desc:              "Streaming methods with NO deadline specified and the global timeout larger than the default deadline, use the global timeout.",
			globalIdleTimeout: 25 * time.Second,
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name:             "api",
								RequestStreaming: true,
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://abc.com/api/",
							Selector: "abc.com.api",
							// Missing deadline
						},
					},
				},
			},
			wantedMethodIdleTimeout: map[string]time.Duration{
				"abc.com.api": 25 * time.Second,
			},
		},
		{
			desc:              "Streaming methods with NO deadline specified and the global timeout smaller than the default deadline, use the default deadline.",
			globalIdleTimeout: 7 * time.Second,
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name:             "api",
								RequestStreaming: true,
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://abc.com/api/",
							Selector: "abc.com.api",
							// Missing deadline
						},
					},
				},
			},
			wantedMethodIdleTimeout: map[string]time.Duration{
				"abc.com.api": util.DefaultResponseDeadline + time.Second,
			},
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {

			opts := options.DefaultConfigGeneratorOptions()
			opts.StreamIdleTimeout = tc.globalIdleTimeout
			s, err := NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)

			if err != nil {
				t.Errorf("error not expected, got: %v", err)
				return
			}

			for operation, mi := range s.Methods {
				gotIdleTimeout := mi.BackendInfo.IdleTimeout
				wantIdleTimeout := tc.wantedMethodIdleTimeout[operation]

				if gotIdleTimeout != wantIdleTimeout {
					t.Errorf("IdleTimeout not expected, got: %v, want: %v", gotIdleTimeout, wantIdleTimeout)
				}

				if wantedHTTPBackendIdleTimeout, ok := tc.wantedHTTPBackendIdleTimeout[operation]; !ok {
					continue
				} else {
					if mi.HttpBackendInfo == nil {
						t.Fatalf("HTTP backend info is nil; want non-nil because the http backend rule is specified.")
					}
					gotHTTPBackendIdleTimeout := mi.HttpBackendInfo.IdleTimeout
					if wantedHTTPBackendIdleTimeout != gotHTTPBackendIdleTimeout {
						t.Errorf("IdleTimeout not expected, got: %v, want: %v", gotHTTPBackendIdleTimeout, wantedHTTPBackendIdleTimeout)
					}
				}
			}
		})
	}
}

func TestProcessBackendRuleForProtocol(t *testing.T) {
	testData := []struct {
		desc              string
		fakeServiceConfig *confpb.Service
		// Map of cluster name to the expected backend protocol for the backend routing cluster.
		wantedClusterProtocols map[string]util.BackendProtocol
	}{
		{
			desc: "Mixed protocols across multiple backend rules",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
					{
						Name: "cnn.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "https://abc.com/api/",
							Selector: "abc.com.api",
							Protocol: "http/1.1",
						},
						{
							Address:  "https://cnn.com/api/",
							Selector: "cnn.com.api",
							Protocol: "h2",
						},
					},
				},
			},
			wantedClusterProtocols: map[string]util.BackendProtocol{
				"backend-cluster-abc.com:443": util.HTTP1,
				"backend-cluster-cnn.com:443": util.HTTP2,
			},
		},
		{
			// This case is not supported in practice, but we shouldn't break ordering if a user does it.
			desc: "When multiple backend rules with the same address have different protocols, only first one is used",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "api.test",
						Methods: []*apipb.Method{
							{
								Name: "1",
							},
							{
								Name: "2",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "https://abc.com/api/",
							Selector: "api.test.1",
							Protocol: "http/1.1",
						},
						{
							Address:  "https://abc.com/api/",
							Selector: "api.test.2",
							Protocol: "h2",
						},
					},
				},
			},
			wantedClusterProtocols: map[string]util.BackendProtocol{
				"backend-cluster-abc.com:443": util.HTTP1,
			},
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			opts := options.DefaultConfigGeneratorOptions()
			s, err := NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)

			if err != nil {
				t.Errorf("Test Desc(%s): error not expected, got: %v", tc.desc, err)
				return
			}

			for _, gotBackendRoutingCluster := range s.RemoteBackendClusters {
				gotProtocol := gotBackendRoutingCluster.Protocol
				wantProtocol, ok := tc.wantedClusterProtocols[gotBackendRoutingCluster.ClusterName]

				if !ok {
					t.Errorf("Test Desc(%s): Unknown backend routing cluster generated: %+v", tc.desc, gotBackendRoutingCluster)
					continue
				}

				if wantProtocol != gotProtocol {
					t.Errorf("Test Desc(%s): Protocol not expected, got: %v, want: %v", tc.desc, gotProtocol, wantProtocol)
				}
			}
		})
	}
}

func TestProcessBackendRuleForClusterName(t *testing.T) {
	testData := []struct {
		desc        string
		Address     string
		ClusterName string
	}{
		{
			desc:        "Domain name with default http port",
			Address:     "http://abc.com/api/",
			ClusterName: "backend-cluster-abc.com:80",
		},
		{
			desc:        "Domain name with default https port",
			Address:     "https://abc.com/api/",
			ClusterName: "backend-cluster-abc.com:443",
		},
		{
			desc:        "Domain name with default grpc port",
			Address:     "grpc://abc.com/api/",
			ClusterName: "backend-cluster-abc.com:80",
		},
		{
			desc:        "Domain name with default grpcs port",
			Address:     "grpcs://abc.com/api/",
			ClusterName: "backend-cluster-abc.com:443",
		},
		{
			desc:        "Domain name with custom http port",
			Address:     "http://abc.com:8080/api/",
			ClusterName: "backend-cluster-abc.com:8080",
		},
		{
			desc:        "Domain name with custom https port",
			Address:     "https://abc.com:8080/api/",
			ClusterName: "backend-cluster-abc.com:8080",
		},
		{
			desc:        "Domain name with custom grpc port",
			Address:     "grpc://abc.com:8080/api/",
			ClusterName: "backend-cluster-abc.com:8080",
		},
		{
			desc:        "Domain name with custom grpcs port",
			Address:     "grpcs://abc.com:8080/api/",
			ClusterName: "backend-cluster-abc.com:8080",
		},
		{
			desc:        "IP with default http port",
			Address:     "http://127.0.0.1/api/",
			ClusterName: "backend-cluster-127.0.0.1:80",
		},
		{
			desc:        "IP with default https port",
			Address:     "https://127.0.0.1/api/",
			ClusterName: "backend-cluster-127.0.0.1:443",
		},
		{
			desc:        "IP with default grpc port",
			Address:     "grpc://127.0.0.1/api/",
			ClusterName: "backend-cluster-127.0.0.1:80",
		},
		{
			desc:        "IP with default grpcs port",
			Address:     "grpcs://127.0.0.1/api/",
			ClusterName: "backend-cluster-127.0.0.1:443",
		},
		{
			desc:        "IP with custom http port",
			Address:     "http://127.0.0.1:8080/api/",
			ClusterName: "backend-cluster-127.0.0.1:8080",
		},
		{
			desc:        "IP with custom https port",
			Address:     "https://127.0.0.1:8080/api/",
			ClusterName: "backend-cluster-127.0.0.1:8080",
		},
		{
			desc:        "IP with custom grpc port",
			Address:     "grpc://127.0.0.1:8080/api/",
			ClusterName: "backend-cluster-127.0.0.1:8080",
		},
		{
			desc:        "IP with custom grpcs port",
			Address:     "grpcs://127.0.0.1:8080/api/",
			ClusterName: "backend-cluster-127.0.0.1:8080",
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			fakeServiceConfig := &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "http.abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  tc.Address,
							Selector: "http.abc.com.api",
						},
					},
				},
			}
			opts := options.DefaultConfigGeneratorOptions()
			s, err := NewServiceInfoFromServiceConfig(fakeServiceConfig, testConfigID, opts)

			if err != nil {
				t.Errorf("Test Desc(%s): error not expected, got: %v", tc.desc, err)
				return
			}

			if len(s.RemoteBackendClusters) != 1 {
				t.Errorf("Test Desc(%s): generated number of clusters is not 1", tc.desc)
				return
			}
			if tc.ClusterName != s.RemoteBackendClusters[0].ClusterName {
				t.Errorf("Test Desc(%s): cluster name is different, want: %s, got %s", tc.desc, tc.ClusterName, s.RemoteBackendClusters[0].ClusterName)
			}
		})
	}
}

func TestProcessBackendRuleForJwtAudience(t *testing.T) {
	testData := []struct {
		desc              string
		fakeServiceConfig *confpb.Service
		nonGcp            bool
		wantedJwtAudience map[string]string
	}{

		{
			desc: "DisableAuth is set to true",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{

						{
							Address:        "grpc://abc.com/api",
							Selector:       "abc.com.api",
							Deadline:       10.5,
							Authentication: &confpb.BackendRule_DisableAuth{DisableAuth: true},
						},
					},
				},
			},
			wantedJwtAudience: map[string]string{
				"abc.com.api": "",
			},
		},
		{
			desc: "DisableAuth is set to false",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{

						{
							Address:        "grpc://abc.com/api",
							Selector:       "abc.com.api",
							Deadline:       10.5,
							Authentication: &confpb.BackendRule_DisableAuth{DisableAuth: false},
						},
					},
				},
			},
			wantedJwtAudience: map[string]string{
				"abc.com.api": "http://abc.com",
			},
		},
		{
			desc: "Authentication field is empty and grpc scheme is changed to http",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{

						{
							Address:  "grpc://abc.com/api",
							Selector: "abc.com.api",
							Deadline: 10.5,
						},
					},
				},
			},
			wantedJwtAudience: map[string]string{
				"abc.com.api": "http://abc.com",
			},
		},
		{
			desc: "Authentication field is empty and grpcs scheme is changed to https",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{

						{
							Address:  "grpcs://abc.com/api",
							Selector: "abc.com.api",
							Deadline: 10.5,
						},
					},
				},
			},
			wantedJwtAudience: map[string]string{
				"abc.com.api": "https://abc.com",
			},
		},
		{
			desc: "JwtAudience is set",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:        "grpc://abc.com/api",
							Selector:       "abc.com.api",
							Deadline:       10.5,
							Authentication: &confpb.BackendRule_JwtAudience{JwtAudience: "audience-foo"},
						},
					},
				},
			},
			wantedJwtAudience: map[string]string{
				"abc.com.api": "audience-foo",
			},
		},
		{
			desc:   "JwtAudience is set, but non-GCP runtime disables backend auth",
			nonGcp: true,
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:        "grpc://abc.com/api",
							Selector:       "abc.com.api",
							Deadline:       10.5,
							Authentication: &confpb.BackendRule_JwtAudience{JwtAudience: "audience-foo"},
						},
					},
				},
			},
			wantedJwtAudience: map[string]string{
				"abc.com.api": "",
			},
		},
		{
			desc: "Mix all Authentication cases",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
					{
						Name: "def.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
					{
						Name: "ghi.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
					{
						Name: "jkl.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
					{
						Name: "mno.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:        "grpc://abc.com/api",
							Selector:       "abc.com.api",
							Deadline:       10.5,
							Authentication: &confpb.BackendRule_JwtAudience{JwtAudience: "audience-foo"},
						},
						{
							Address:        "grpc://def.com/api",
							Selector:       "def.com.api",
							Deadline:       10.5,
							Authentication: &confpb.BackendRule_JwtAudience{JwtAudience: "audience-bar"},
						},
						{
							Address:        "grpc://ghi.com/api",
							Selector:       "ghi.com.api",
							Deadline:       10.5,
							Authentication: &confpb.BackendRule_DisableAuth{DisableAuth: false},
						},
						{
							Address:        "grpc://jkl.com/api",
							Selector:       "jkl.com.api",
							Deadline:       10.5,
							Authentication: &confpb.BackendRule_DisableAuth{DisableAuth: true},
						},
						{
							Address:  "grpcs://mno.com/api",
							Selector: "mno.com.api",
							Deadline: 10.5,
						},
					},
				},
			},
			wantedJwtAudience: map[string]string{
				"abc.com.api": "audience-foo",
				"def.com.api": "audience-bar",
				"ghi.com.api": "http://ghi.com",
				"jkl.com.api": "",
				"mno.com.api": "https://mno.com",
			},
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			opts := options.DefaultConfigGeneratorOptions()
			opts.NonGCP = tc.nonGcp
			s, err := NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)

			if err != nil {
				t.Fatalf("error not expected, got: %v", err)
			}

			for _, rule := range tc.fakeServiceConfig.Backend.Rules {
				gotJwtAudience := s.Methods[rule.Selector].BackendInfo.JwtAudience
				wantedJwtAudience := tc.wantedJwtAudience[rule.Selector]

				if wantedJwtAudience != gotJwtAudience {
					t.Errorf("JwtAudience mismatch, got: %v, want: %v", gotJwtAudience, wantedJwtAudience)
				}
			}
		})
	}
}

func TestProcessBackendRuleForRetry(t *testing.T) {
	testData := []struct {
		desc                          string
		fakeServiceConfig             *confpb.Service
		backendRetryOns               string
		backendRetryNum               uint
		backendPerTryTimeout          time.Duration
		backendRetryOnStatusCode      string
		wantBackendRetryOns           string
		wantBackendRetryNum           uint
		wantBackendPerTryTimeout      time.Duration
		wantBackendRetryOnStatusCodes []uint32
		wantError                     string
	}{
		{
			desc: "pass backend retry parameters",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
			},
			backendRetryOns:          "foo,bar",
			backendRetryNum:          5,
			backendPerTryTimeout:     time.Second * 60,
			wantBackendRetryOns:      "foo,bar",
			wantBackendRetryNum:      5,
			wantBackendPerTryTimeout: time.Second * 60,
		},
		{
			desc: "invalid retriable status code in wrong format",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
			},
			backendRetryOns:          "",
			backendRetryOnStatusCode: "invalid-status-code",
			wantError:                "invalid http status code",
		},
		{
			desc: "invalid retriable status code in wrong range",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
			},
			backendRetryOns:          "",
			backendRetryOnStatusCode: "600",
			wantError:                "invalid http status code",
		},
		{
			desc: "set RetryOnStatusCodes and add `retriable-status-codes` to retryOns if it is empty",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
			},
			backendRetryOns:               "",
			backendRetryOnStatusCode:      "500,501",
			wantBackendRetryOns:           "retriable-status-codes",
			wantBackendRetryOnStatusCodes: []uint32{500, 501},
		},
		{
			desc: "set RetryOnStatusCodes and add `retriable-status-codes` to retryOns if it is un-empty but doesn't have `retriable-status-codes`",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
			},
			backendRetryOns:               "foo,bar",
			backendRetryOnStatusCode:      "500,501",
			wantBackendRetryOns:           "foo,bar,retriable-status-codes",
			wantBackendRetryOnStatusCodes: []uint32{500, 501},
		},
		{
			desc: "set RetryOnStatusCodes and no op on retryOns when it contains `retriable-status-codes`",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "api",
							},
						},
					},
				},
			},
			backendRetryOns:               "foo,bar,retriable-status-codes",
			backendRetryOnStatusCode:      "500,501",
			wantBackendRetryOns:           "foo,bar,retriable-status-codes",
			wantBackendRetryOnStatusCodes: []uint32{500, 501},
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			opts := options.DefaultConfigGeneratorOptions()
			opts.BackendRetryOns = tc.backendRetryOns
			opts.BackendRetryNum = tc.backendRetryNum
			opts.BackendPerTryTimeout = tc.backendPerTryTimeout
			opts.BackendRetryOnStatusCodes = tc.backendRetryOnStatusCode
			s, err := NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("different error, want: %s, get: %v", tc.wantError, err)
				}
				return
			}

			for _, method := range s.Methods {
				backendInfo := method.BackendInfo
				if backendInfo.RetryOns != tc.wantBackendRetryOns {
					t.Errorf("different RetryOns, want: %v, get: %v", tc.wantBackendRetryOns, backendInfo.RetryOns)
				}
				if backendInfo.RetryNum != tc.wantBackendRetryNum {
					t.Errorf("different RetryNum, want: %v, get: %v", tc.wantBackendRetryNum, backendInfo.RetryNum)
				}
				if backendInfo.PerTryTimeout != tc.wantBackendPerTryTimeout {
					t.Errorf("different PerTryTimeout, want: %v, get: %v", tc.wantBackendPerTryTimeout, backendInfo.PerTryTimeout)
				}
				if !reflect.DeepEqual(backendInfo.RetriableStatusCodes, tc.wantBackendRetryOnStatusCodes) {
					t.Errorf("different RetriableStatusCodes, want: %v, get: %v", tc.wantBackendRetryOnStatusCodes, backendInfo.RetriableStatusCodes)
				}
			}
		})
	}
}

func TestBackendAddressOverride(t *testing.T) {
	testData := []struct {
		desc                         string
		backendAddress               string
		enableBackendAddressOverride bool
		// Map of selector to the expected backend cluster.
		wantedMethodBackendCluster map[string]string
	}{
		{
			desc:                         "When disabled, backend.rule.address has priority",
			backendAddress:               "http://127.0.0.1:9000",
			enableBackendAddressOverride: false,
			wantedMethodBackendCluster: map[string]string{
				"abc.com.a": "backend-cluster-abc.com:80",
				"abc.com.b": "backend-cluster-echo.endpoints_local",
			},
		},
		{
			desc:                         "When enabled, backend.rule.address losses priority",
			backendAddress:               "http://127.0.0.1:9000",
			enableBackendAddressOverride: true,
			wantedMethodBackendCluster: map[string]string{
				"abc.com.a": "backend-cluster-echo.endpoints_local",
				"abc.com.b": "backend-cluster-echo.endpoints_local",
			},
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {

			fakeServiceConfig := &confpb.Service{
				Name: "echo.endpoints",
				Apis: []*apipb.Api{
					{
						Name: "abc.com",
						Methods: []*apipb.Method{
							{
								Name: "a",
							},
							{
								Name: "b",
							},
						},
					},
				},
				Backend: &confpb.Backend{
					Rules: []*confpb.BackendRule{
						{
							Address:  "grpc://abc.com/a/",
							Selector: "abc.com.a",
						},
					},
				},
			}

			opts := options.DefaultConfigGeneratorOptions()
			opts.BackendAddress = tc.backendAddress
			opts.EnableBackendAddressOverride = tc.enableBackendAddressOverride
			s, err := NewServiceInfoFromServiceConfig(fakeServiceConfig, testConfigID, opts)

			if err != nil {
				t.Errorf("error not expected, got: %v", err)
				return
			}

			for operation, mi := range s.Methods {
				gotBackendCluster := mi.BackendInfo.ClusterName
				wantBackendCluster := tc.wantedMethodBackendCluster[operation]

				if gotBackendCluster != wantBackendCluster {
					t.Errorf("Backend cluster name not expected, got: %v, want: %v", gotBackendCluster, wantBackendCluster)
				}
			}
		})
	}
}

func TestProcessQuota(t *testing.T) {
	testData := []struct {
		desc              string
		fakeServiceConfig *confpb.Service
		wantMethods       map[string]*MethodInfo
		wantError         string
	}{
		{
			desc: "Succeed, simple case",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: testApiName,
						Methods: []*apipb.Method{
							{
								Name: "ListShelves",
							},
						},
					},
				},
				Quota: &confpb.Quota{
					MetricRules: []*confpb.MetricRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
							MetricCosts: map[string]int64{
								"metric_a": 2,
								"metric_b": 1,
							},
						},
					},
				},
			},
			wantMethods: map[string]*MethodInfo{
				fmt.Sprintf("%s.%s", testApiName, "ListShelves"): &MethodInfo{
					ShortName: "ListShelves",
					ApiName:   testApiName,
					HttpRule: []*httppattern.Pattern{
						{
							UriTemplate: parseUriTemplate(fmt.Sprintf("/%s/%s", testApiName, "ListShelves")),
							HttpMethod:  util.POST,
						},
					},
					MetricCosts: []*scpb.MetricCost{
						{
							Name: "metric_a",
							Cost: 2,
						},
						{
							Name: "metric_b",
							Cost: 1,
						},
					},
				},
			},
		},
		{
			desc: "Succeed, two metric cost items",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: testApiName,
						Methods: []*apipb.Method{
							{
								Name: "ListShelves",
							},
						},
					},
				},
				Quota: &confpb.Quota{
					MetricRules: []*confpb.MetricRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.ListShelves",
							MetricCosts: map[string]int64{
								"metric_c": 2,
								"metric_a": 3,
							},
						},
					},
				},
			},
			wantMethods: map[string]*MethodInfo{
				fmt.Sprintf("%s.%s", testApiName, "ListShelves"): &MethodInfo{
					ShortName: "ListShelves",
					ApiName:   testApiName,
					HttpRule: []*httppattern.Pattern{
						{
							UriTemplate: parseUriTemplate(fmt.Sprintf("/%s/%s", testApiName, "ListShelves")),
							HttpMethod:  util.POST,
						},
					},
					MetricCosts: []*scpb.MetricCost{
						{
							Name: "metric_a",
							Cost: 3,
						},
						{
							Name: "metric_c",
							Cost: 2,
						},
					},
				},
			},
		},
		{
			desc: "Typo in operation name does not crash",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: testApiName,
					},
				},
				Quota: &confpb.Quota{
					MetricRules: []*confpb.MetricRule{
						{
							Selector: "endpoints.examples.bookstore.Bookstore.BadOperationName",
							MetricCosts: map[string]int64{
								"metric_a": 2,
								"metric_b": 1,
							},
						},
					},
				},
			},
			wantError: "error processing quota metric rule: selector (endpoints.examples.bookstore.Bookstore.BadOperationName) was not defined in the API",
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			opts := options.DefaultConfigGeneratorOptions()
			opts.BackendAddress = "grpc://127.0.0.1:80"
			serviceInfo, err := NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)

			if err != nil {
				if tc.wantError == "" || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("error mismatch, \ngot : %s, \nwant: %s", err.Error(), tc.wantError)
				}
				return
			}

			for key, gotMethod := range serviceInfo.Methods {
				wantMethod := tc.wantMethods[key]

				// We're not testing backend info here.
				gotMethod.BackendInfo = nil

				sort.Slice(gotMethod.MetricCosts, func(i, j int) bool { return gotMethod.MetricCosts[i].Name < gotMethod.MetricCosts[j].Name })
				if eq := cmp.Equal(gotMethod, wantMethod, cmp.Comparer(proto.Equal)); !eq {
					t.Errorf("Method mismatch \ngot : %+v,\nwant: %+v", gotMethod, wantMethod)
				}
			}
		})
	}
}

func TestProcessEmptyJwksUriByOpenID(t *testing.T) {
	r := mux.NewRouter()
	jwksUriEntry, _ := json.Marshal(map[string]string{"jwks_uri": "this-is-jwksUri"})
	r.Path(util.OpenIDDiscoveryCfgURLSuffix).Methods("GET").Handler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(jwksUriEntry)
		}))
	openIDServer := httptest.NewServer(r)

	testData := []struct {
		desc                 string
		fakeServiceConfig    *confpb.Service
		disableOidcDiscovery bool
		wantedJwksUri        string
		wantErr              bool
	}{
		{
			desc: "Success, empty JWKS URI, so it's acquired using OpenID Connect Discovery.",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: testApiName,
					},
				},
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:     "auth_provider",
							Issuer: openIDServer.URL,
						},
					},
				},
			},
			wantedJwksUri: "this-is-jwksUri",
		},
		{
			desc: "Fail, empty JWKS URI and Open ID Connect Discovery failed.",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: testApiName,
					},
				},
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:     "auth_provider",
							Issuer: "aaaaa.bbbbbb.ccccc/inaccessible_uri/",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			desc: "Fail, empty JWKS URI but OpenID Connect Discovery disabled.",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: testApiName,
					},
				},
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:     "auth_provider",
							Issuer: openIDServer.URL,
						},
					},
				},
			},
			disableOidcDiscovery: true,
			wantErr:              true,
		},
	}

	for i, tc := range testData {
		opts := options.DefaultConfigGeneratorOptions()
		opts.DisableOidcDiscovery = tc.disableOidcDiscovery
		serviceInfo, err := NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)

		if tc.wantErr {
			if err == nil {
				t.Errorf("Test Desc(%d): %s, process jwksUri got: no err, but expected err", i, tc.desc)
			}
		} else if err != nil {
			t.Errorf("Test Desc(%d): %s, process jwksUri got: %v, but expected no err", i, tc.desc, err)
		} else if jwksUri := serviceInfo.serviceConfig.Authentication.Providers[0].JwksUri; jwksUri != tc.wantedJwksUri {
			t.Errorf("Test Desc(%d): %s, process jwksUri got: %v, want: %v", i, tc.desc, jwksUri, tc.wantedJwksUri)
		}
	}
}

func TestProcessApis(t *testing.T) {
	testData := []struct {
		desc              string
		fakeServiceConfig *confpb.Service
		wantMethods       map[string]*MethodInfo
		wantApiNames      []string
	}{
		{
			desc: "Succeed, process multiple apis",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "api-1",
						Methods: []*apipb.Method{
							{
								Name:           "foo",
								RequestTypeUrl: "type.googleapis.com/google.protobuf.Empty",
							},
							{
								Name:           "bar",
								RequestTypeUrl: "type.googleapis.com/CreateShelfRequest",
							},
						},
					},
					{
						Name: "api-2",
						Methods: []*apipb.Method{
							{
								Name:           "foo",
								RequestTypeUrl: "type.googleapis.com/CreateBookRequest",
							},
							{
								Name:           "bar",
								RequestTypeUrl: "type.googleapis.com/CreateShelfRequest",
							},
						},
					},
					{
						Name:    "api-3",
						Methods: []*apipb.Method{},
					},
					{
						Name: "api-4",
						Methods: []*apipb.Method{
							{
								Name: "bar",
								// Missing request type.
							},
							{
								Name:           "baz",
								RequestTypeUrl: "types.MALFORMED.com/google.protobuf.Empty",
							},
						},
					},
				},
			},
			wantMethods: map[string]*MethodInfo{
				"api-1.foo": {
					ShortName:       "foo",
					ApiName:         "api-1",
					RequestTypeName: "google.protobuf.Empty",
				},
				"api-1.bar": {
					ShortName:       "bar",
					ApiName:         "api-1",
					RequestTypeName: "CreateShelfRequest",
				},
				"api-2.foo": {
					ShortName:       "foo",
					ApiName:         "api-2",
					RequestTypeName: "CreateBookRequest",
				},
				"api-2.bar": {
					ShortName:       "bar",
					ApiName:         "api-2",
					RequestTypeName: "CreateShelfRequest",
				},
				"api-4.bar": {
					ShortName: "bar",
					ApiName:   "api-4",
				},
				"api-4.baz": {
					ShortName: "baz",
					ApiName:   "api-4",
				},
			},
			wantApiNames: []string{
				"api-1",
				"api-2",
				"api-3",
				"api-4",
			},
		},
	}

	for _, tc := range testData {

		serviceInfo := &ServiceInfo{
			serviceConfig: tc.fakeServiceConfig,
			Methods:       make(map[string]*MethodInfo),
		}
		_ = serviceInfo.processApis()

		for key, gotMethod := range serviceInfo.Methods {
			wantMethod := tc.wantMethods[key]
			if eq := cmp.Equal(gotMethod, wantMethod, cmp.Comparer(proto.Equal)); !eq {
				t.Errorf("Test(%v) failed: \n got Method: %v,\nwant Method: %v", tc.desc, gotMethod, wantMethod)
			}
		}
		for idx, gotApiName := range serviceInfo.ApiNames {
			wantApiName := tc.wantApiNames[idx]
			if gotApiName != wantApiName {
				t.Errorf("Test(%v) failed: \n got ApiName: %v,\nwant Apiname: %v", tc.desc, gotApiName, wantApiName)
			}
		}
	}
}

func TestProcessApisForGrpc(t *testing.T) {
	testData := []struct {
		desc              string
		fakeServiceConfig *confpb.Service
		wantMethods       map[string]*MethodInfo
		wantApiNames      []string
		wantError         string
	}{
		{
			desc: "Process API with unary and streaming gRPC methods; skip discovery API",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "api-streaming-test",
						Methods: []*apipb.Method{
							{
								Name: "unary",
							},
							{
								Name:             "streaming_request",
								RequestStreaming: true,
							},
							{
								Name:              "streaming_response",
								ResponseStreaming: true,
							},
						},
					},
					{
						Name: "google.discovery.Discovery",
						Methods: []*apipb.Method{
							{
								Name: "GetDiscovery",
							},
						},
					},
				},
			},
			wantMethods: map[string]*MethodInfo{
				"api-streaming-test.unary": {
					ShortName: "unary",
					ApiName:   "api-streaming-test",
					HttpRule: []*httppattern.Pattern{
						{
							UriTemplate: parseUriTemplate("/api-streaming-test/unary"),
							HttpMethod:  util.POST,
						},
					},
				},
				"api-streaming-test.streaming_request": {
					ShortName:   "streaming_request",
					ApiName:     "api-streaming-test",
					IsStreaming: true,
					HttpRule: []*httppattern.Pattern{
						{
							UriTemplate: parseUriTemplate("/api-streaming-test/streaming_request"),
							HttpMethod:  util.POST,
						},
					},
				},
				"api-streaming-test.streaming_response": {
					ShortName:   "streaming_response",
					ApiName:     "api-streaming-test",
					IsStreaming: true,
					HttpRule: []*httppattern.Pattern{
						{
							UriTemplate: parseUriTemplate("/api-streaming-test/streaming_response"),
							HttpMethod:  util.POST,
						},
					},
				},
			},
			wantApiNames: []string{
				"api-streaming-test",
			},
		},
		{
			desc: "fail add httpRule for Invalid grpc config",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "api-streaming-test",
						Methods: []*apipb.Method{
							{
								Name: "***",
							},
						},
					},
				},
			},
			wantError: "error parsing auto-generated gRPC http rule's URI template",
		},
	}

	for i, tc := range testData {

		serviceInfo := &ServiceInfo{
			serviceConfig:       tc.fakeServiceConfig,
			GrpcSupportRequired: true,
			Methods:             make(map[string]*MethodInfo),
		}
		_ = serviceInfo.processApis()
		if err := serviceInfo.addGrpcHttpRules(); err != nil {
			if tc.wantError == "" || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("For processGrpcHttpRules error,\ngot : %s, \nwant: %s", err.Error(), tc.wantError)
			}
			continue
		}

		for key, gotMethod := range serviceInfo.Methods {
			wantMethod := tc.wantMethods[key]
			if eq := cmp.Equal(gotMethod, wantMethod, cmp.Comparer(proto.Equal)); !eq {
				t.Errorf("Test Desc(%d): %s,\ngot Method: %v,\nwant Method: %v", i, tc.desc, gotMethod, wantMethod)
			}
		}
		for idx, gotApiName := range serviceInfo.ApiNames {
			wantApiName := tc.wantApiNames[idx]
			if gotApiName != wantApiName {
				t.Errorf("Test Desc(%d): %s,\ngot ApiName: %v,\nwant Apiname: %v", i, tc.desc, gotApiName, wantApiName)
			}
		}
	}
}

func TestProcessTypes(t *testing.T) {
	testData := []struct {
		desc                            string
		fakeServiceConfig               *confpb.Service
		fakeRequestTypeNamesByOperation map[string]string
		urlTemplateByOperation          map[string]string
		wantUrlTemplateByOperation      map[string]string
		wantErr                         error
	}{
		{
			desc: "Success for single type with multiple distinct fields",
			fakeRequestTypeNamesByOperation: map[string]string{
				"api-1.operation-1": "CreateShelvesRequest",
			},
			fakeServiceConfig: &confpb.Service{
				Types: []*ptypepb.Type{
					{
						Name: "CreateShelvesRequest",
						Fields: []*ptypepb.Field{
							{
								Name:     "foo_bar",
								JsonName: "fooBar",
							},
							{
								Name:     "x_y",
								JsonName: "xY",
							},
						},
					},
				},
			},
			urlTemplateByOperation: map[string]string{
				"api-1.operation-1": "/foo/{foo_bar}/bar/{x_y}",
			},
			wantUrlTemplateByOperation: map[string]string{
				"api-1.operation-1": "/foo/{fooBar=*}/bar/{xY=*}",
			},
		},
		{
			desc: "Success for single type with multiple duplicated fields, which can be de-duped",
			fakeRequestTypeNamesByOperation: map[string]string{
				"api-1.operation-1": "CreateShelvesRequest",
			},
			fakeServiceConfig: &confpb.Service{
				Types: []*ptypepb.Type{
					{
						Name: "CreateShelvesRequest",
						Fields: []*ptypepb.Field{
							{
								Name:     "foo_bar",
								JsonName: "fooBar",
							},
							{
								Name:     "foo_bar",
								JsonName: "fooBar",
							},
						},
					},
				},
			},

			urlTemplateByOperation: map[string]string{
				"api-1.operation-1": "/foo/{foo_bar}/bar",
			},
			wantUrlTemplateByOperation: map[string]string{
				"api-1.operation-1": "/foo/{fooBar=*}/bar",
			},
		},
		{
			desc: "Success for single type with multiple non-conflicting fields",
			fakeRequestTypeNamesByOperation: map[string]string{
				"api-1.operation-1": "CreateShelvesRequest",
			},
			fakeServiceConfig: &confpb.Service{
				Types: []*ptypepb.Type{
					{
						Name: "CreateShelvesRequest",
						Fields: []*ptypepb.Field{
							{
								// This one is ignored, its doesn't map anything.
								Name:     "fooBar",
								JsonName: "fooBar",
							},
							{
								Name:     "foo_bar",
								JsonName: "fooBar",
							},
						},
					},
				},
			},
			urlTemplateByOperation: map[string]string{
				"api-1.operation-1": "/foo/{foo_bar}",
			},
			wantUrlTemplateByOperation: map[string]string{
				"api-1.operation-1": "/foo/{fooBar=*}",
			},
		},
		{
			desc: "Failure for single type with multiple conflicting fields",
			fakeRequestTypeNamesByOperation: map[string]string{
				"api-1.operation-1": "CreateShelvesRequest",
			},
			fakeServiceConfig: &confpb.Service{
				Types: []*ptypepb.Type{
					{
						Name: "CreateShelvesRequest",
						Fields: []*ptypepb.Field{
							{
								Name:     "foo_bar",
								JsonName: "foo-bar",
							},
							{
								Name:     "foo_bar",
								JsonName: "fooBar",
							},
						},
					},
				},
			},
			wantErr: fmt.Errorf("for operation (api-1.operation-1): detected two types with same snake_name (foo_bar) but mistmatching json_name"),
		},
		{
			desc: "Success for multiple types with distinct fields",
			fakeRequestTypeNamesByOperation: map[string]string{
				"api-1.operation-1": "CreateShelvesRequest",
				"api-1.operation-2": "CreateBookRequest",
				"api-2.operation-1": "google.protobuf.Empty",
				// This will be ignored, it doesn't exist in types.
				"api-3.operation-1": "NonExistingType",
				// This will be ignored, it is empty.
				"api-3.operation-2": "",
			},
			fakeServiceConfig: &confpb.Service{
				Types: []*ptypepb.Type{
					{
						Name: "CreateShelvesRequest",
						Fields: []*ptypepb.Field{
							{
								Name:     "foo_bar",
								JsonName: "fooBar",
							},
							{
								Name:     "x_y",
								JsonName: "xY",
							},
						},
					},
					{
						Name: "CreateBookRequest",
						Fields: []*ptypepb.Field{
							{
								// This one will be ignored, the names match.
								Name:     "baz",
								JsonName: "baz",
							},
							{
								Name:     "a_b",
								JsonName: "aB",
							},
						},
					},
					{
						Name: "google.protobuf.Empty",
						// This will be ignored, no fields at all.
						Fields: []*ptypepb.Field{},
					},
					{
						// This will be ignored, it's not directly any operation's request type.
						Name: "Library",
						Fields: []*ptypepb.Field{
							{
								Name:     "lib_name",
								JsonName: "libName",
							},
						},
					},
				},
			},
			urlTemplateByOperation: map[string]string{
				"api-1.operation-1": "/foo/{foo_bar}/bar/{x_y}",
				"api-1.operation-2": "/foo/{a_b}",
			},
			wantUrlTemplateByOperation: map[string]string{
				"api-1.operation-1": "/foo/{fooBar=*}/bar/{xY=*}",
				"api-1.operation-2": "/foo/{aB=*}",
			},
		},
		{
			desc: "Success for multiple types with conflicting fields across types, but no conflicts within a single type",
			fakeRequestTypeNamesByOperation: map[string]string{
				"api-1.operation-1": "CreateShelvesRequest",
				"api-1.operation-2": "CreateBookRequest",
			},
			fakeServiceConfig: &confpb.Service{
				Types: []*ptypepb.Type{
					{
						Name: "CreateShelvesRequest",
						Fields: []*ptypepb.Field{
							{
								Name:     "foo_bar",
								JsonName: "foo-bar",
							},
						},
					},
					{
						Name: "CreateBookRequest",
						Fields: []*ptypepb.Field{
							{
								Name:     "foo_bar",
								JsonName: "fooBar",
							},
						},
					},
				},
			},
			urlTemplateByOperation: map[string]string{
				"api-1.operation-1": "/foo/{foo_bar}",
				"api-1.operation-2": "/foo/{foo_bar}",
			},
			wantUrlTemplateByOperation: map[string]string{
				"api-1.operation-1": "/foo/{foo-bar=*}",
				"api-1.operation-2": "/foo/{fooBar=*}",
			},
		},
	}

	for _, tc := range testData {
		serviceInfo := &ServiceInfo{
			serviceConfig: tc.fakeServiceConfig,
			Methods:       make(map[string]*MethodInfo),
		}
		// Emulate a part of processApis().
		for operation, requestType := range tc.fakeRequestTypeNamesByOperation {
			mi := &MethodInfo{
				RequestTypeName: requestType,
				HttpRule: []*httppattern.Pattern{
					{
						UriTemplate: parseUriTemplate(tc.urlTemplateByOperation[operation]),
					},
				},
			}
			serviceInfo.Methods[operation] = mi
		}

		err := serviceInfo.processTypes()

		if err != nil {
			if tc.wantErr == nil || !strings.Contains(err.Error(), tc.wantErr.Error()) {
				t.Errorf("Test(%v): Expected err (%v), got err (%v)", tc.desc, tc.wantErr, err)
			}
			continue
		}

		if tc.wantErr != nil {
			t.Errorf("Test(%v): Expected err (%v), got no err", tc.desc, tc.wantErr)
		}

		for operation, wantUrlTemplate := range tc.wantUrlTemplateByOperation {
			getUrlTemplate := serviceInfo.Methods[operation].HttpRule[0].UriTemplate.ExactMatchString(false)
			if getUrlTemplate != wantUrlTemplate {
				t.Errorf("Test(%v): For operation (%v), expected urlTemplate (%v), got urlTemplate(%v)", tc.desc, operation, wantUrlTemplate, getUrlTemplate)
			}
		}

	}
}

func TestProcessAccessToken(t *testing.T) {
	fakeServiceConfig := &confpb.Service{
		Apis: []*apipb.Api{
			{
				Name: testApiName,
			},
		},
	}
	testCases := []struct {
		desc              string
		serviceAccountKey string
		wantAccessToken   *commonpb.AccessToken
	}{
		{
			desc: "get access token from imds",
			wantAccessToken: &commonpb.AccessToken{
				TokenType: &commonpb.AccessToken_RemoteToken{
					RemoteToken: &commonpb.HttpUri{
						Uri:     "http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token",
						Cluster: "metadata-cluster",
						Timeout: ptypes.DurationProto(30 * time.Second),
					},
				},
			},
		},
		{
			desc:              "get access token from lmds",
			serviceAccountKey: "this-is-service-account-key",
			wantAccessToken: &commonpb.AccessToken{
				TokenType: &commonpb.AccessToken_RemoteToken{
					RemoteToken: &commonpb.HttpUri{
						Uri:     "http://127.0.0.1:8791/local/access_token",
						Cluster: "token-agent-cluster",
						Timeout: ptypes.DurationProto(30 * time.Second),
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		opts := options.DefaultConfigGeneratorOptions()
		opts.ServiceAccountKey = tc.serviceAccountKey
		serviceInfo, err := NewServiceInfoFromServiceConfig(fakeServiceConfig, "ConfigID", opts)
		if err != nil {
			t.Fatal(err)
		}

		serviceInfo.processAccessToken()
		if !reflect.DeepEqual(serviceInfo.AccessToken, tc.wantAccessToken) {
			t.Errorf("fail(%s): expect accessToken: %v, get accessToken: %v", tc.desc, tc.wantAccessToken, serviceInfo.AccessToken)
		}
	}

}

func TestProcessUsageRule(t *testing.T) {
	testData := []struct {
		desc              string
		fakeServiceConfig *confpb.Service
		wantMethods       map[string]*MethodInfo
	}{
		{
			desc: "Make health check methods skip service control by default",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "grpc.health.v1.Health",
						Methods: []*apipb.Method{
							{
								Name: "Check",
							},
							{
								Name: "Watch",
							},
						},
					},
				},
			},
			wantMethods: map[string]*MethodInfo{
				"grpc.health.v1.Health.Check": &MethodInfo{
					ShortName: "Check",
					ApiName:   "grpc.health.v1.Health",
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.POST,
							UriTemplate: parseUriTemplate("/grpc.health.v1.Health/Check"),
						},
					},
					SkipServiceControl: true,
				},
				"grpc.health.v1.Health.Watch": &MethodInfo{
					ShortName: "Watch",
					ApiName:   "grpc.health.v1.Health",
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.POST,
							UriTemplate: parseUriTemplate("/grpc.health.v1.Health/Watch"),
						},
					},
					SkipServiceControl: true,
				},
			},
		},
		{
			desc: "User overrides hardcoded skip service control with usage rule",
			fakeServiceConfig: &confpb.Service{
				Apis: []*apipb.Api{
					{
						Name: "grpc.health.v1.Health",
						Methods: []*apipb.Method{
							{
								Name: "Check",
							},
						},
					},
				},
				Usage: &confpb.Usage{
					Rules: []*confpb.UsageRule{
						{
							Selector:           "grpc.health.v1.Health.Check",
							SkipServiceControl: false,
						},
					},
				},
			},
			wantMethods: map[string]*MethodInfo{
				"grpc.health.v1.Health.Check": &MethodInfo{
					ShortName: "Check",
					ApiName:   "grpc.health.v1.Health",
					HttpRule: []*httppattern.Pattern{
						{
							HttpMethod:  util.POST,
							UriTemplate: parseUriTemplate("/grpc.health.v1.Health/Check"),
						},
					},
					SkipServiceControl: false,
				},
			},
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			opts := options.DefaultConfigGeneratorOptions()
			opts.BackendAddress = "grpc://127.0.0.1:80"
			serviceInfo, err := NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
			if err != nil {
				t.Fatal(err)
			}
			if len(serviceInfo.Methods) != len(tc.wantMethods) {
				t.Fatalf("Num methods mismatch \ngot : %v, want: %v", serviceInfo.Methods, tc.wantMethods)
			}
			for key, gotMethod := range serviceInfo.Methods {
				wantMethod := tc.wantMethods[key]

				// We're not testing backend info here.
				gotMethod.BackendInfo = nil

				if eq := cmp.Equal(gotMethod, wantMethod, cmp.Comparer(proto.Equal)); !eq {
					t.Errorf("Method mistmatch \ngot : %+v,\nwant: %+v", gotMethod, wantMethod)
				}
			}
		})
	}
}

func TestProcessServiceControlURL(t *testing.T) {
	testData := []struct {
		desc                  string
		serviceConfigIn       *confpb.Service
		optionsIn             options.ConfigGeneratorOptions
		wantServiceControlURI url.URL
	}{
		{
			desc: "URL from service config by default",
			serviceConfigIn: &confpb.Service{
				Control: &confpb.Control{
					Environment: "https://staging-servicecontrol.sandbox.googleapis.com",
				},
			},
			wantServiceControlURI: url.URL{
				Scheme: "https",
				Host:   "staging-servicecontrol.sandbox.googleapis.com:443",
			},
		},
		{
			desc: "option overrides service config",
			serviceConfigIn: &confpb.Service{
				Control: &confpb.Control{
					// not used due to non-empty option
					Environment: "https://staging-servicecontrol.sandbox.googleapis.com",
				},
			},
			optionsIn: options.ConfigGeneratorOptions{
				ServiceControlURL: "https://servicecontrol.googleapis.com",
			},
			wantServiceControlURI: url.URL{
				Scheme: "https",
				Host:   "servicecontrol.googleapis.com:443",
			},
		},
		{
			desc:                  "Empty inputs results in empty URL",
			serviceConfigIn:       &confpb.Service{},
			wantServiceControlURI: url.URL{},
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			// Fill in required fields not relevant to test.
			tc.serviceConfigIn.Apis = []*apipb.Api{
				{
					Name: testApiName,
				},
			}

			serviceInfo, err := NewServiceInfoFromServiceConfig(tc.serviceConfigIn, testConfigID, tc.optionsIn)
			if err != nil {
				t.Fatalf("processServiceControlURL(...) has wrong error, got: %v, want no error", err)
			}

			if diff := cmp.Diff(tc.wantServiceControlURI, serviceInfo.ServiceControlURI); diff != "" {
				t.Errorf("processServiceControlURL(...) has unexpected diff for ServiceControlURI (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProcessServiceControlURL_BadInput(t *testing.T) {
	testData := []struct {
		desc            string
		serviceConfigIn *confpb.Service
		optionsIn       options.ConfigGeneratorOptions
		wantErr         string
	}{
		{
			desc: "url parsing fails",
			serviceConfigIn: &confpb.Service{
				Control: &confpb.Control{
					Environment: "https://[::1:80",
				},
			},
			wantErr: `parse "https://[::1:80": missing ']' in host`,
		},
		{
			desc: "url should not have path segment",
			serviceConfigIn: &confpb.Service{
				Control: &confpb.Control{
					Environment: "https://servicecontrol.googleapis.com/v1/services",
				},
			},
			wantErr: `should not have path part: /v1/services`,
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			// Fill in required fields not relevant to test.
			tc.serviceConfigIn.Apis = []*apipb.Api{
				{
					Name: testApiName,
				},
			}

			_, err := NewServiceInfoFromServiceConfig(tc.serviceConfigIn, testConfigID, tc.optionsIn)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("processServiceControlURL(...) has wrong error, got: %v, want: %q", err, tc.wantErr)
			}
		})
	}
}

func parseUriTemplate(input string) *httppattern.UriTemplate {
	u, _ := httppattern.ParseUriTemplate(input)
	return u
}
