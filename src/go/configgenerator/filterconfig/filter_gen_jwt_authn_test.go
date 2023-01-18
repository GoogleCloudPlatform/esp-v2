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

package filterconfig

import (
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/jsonpb"

	anypb "github.com/golang/protobuf/ptypes/any"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

func TestJwtAuthnFilter(t *testing.T) {
	testData := []struct {
		desc                       string
		fakeServiceConfig          *confpb.Service
		disableJwksAsyncFetch      bool
		jwksAsyncFetchFastListener bool
		jwtCacheSize               uint
		disableJwtServiceName      bool
		wantJwtAuthnFilter         string
	}{
		{
			desc: "Success. Generate jwt authn filter with default jwt locations with an empty audiences.",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "testapi",
						Methods: []*apipb.Method{
							{
								Name: "foo",
							},
						},
					},
				},
				SourceInfo: &confpb.SourceInfo{
					SourceFiles: []*anypb.Any{content},
				},
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:      "auth_provider",
							Issuer:  "issuer-0",
							JwksUri: "https://fake-jwks.com?key=value",
						},
					},
					Rules: []*confpb.AuthenticationRule{
						{
							Selector: "testapi.foo",
							Requirements: []*confpb.AuthRequirement{
								{
									ProviderId: "auth_provider",
								},
							},
						},
					},
				},
			},
			// Service config AuthProvider.audiences is empty, envoy jwt_authn Provider.audiences is using service name.
			wantJwtAuthnFilter: `{
    "name": "envoy.filters.http.jwt_authn",
    "typedConfig": {
        "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication",
        "providers": {
            "auth_provider": {
                "audiences": [
                    "https://bookstore.endpoints.project123.cloud.goog"
                ],
                "forward": true,
                "forwardPayloadHeader": "X-Endpoint-API-UserInfo",
                "fromHeaders": [
                    {
                        "name": "Authorization",
                        "valuePrefix": "Bearer "
                    },
                    {
                        "name": "X-Goog-Iap-Jwt-Assertion"
                    }
                ],
                "fromParams": [
                    "access_token"
                ],
                "issuer": "issuer-0",
                "payloadInMetadata": "jwt_payloads",
                "remoteJwks": {
                    "cacheDuration": "300s",
                    "httpUri": {
                        "cluster": "jwt-provider-cluster-fake-jwks.com:443",
                        "timeout": "30s",
                        "uri": "https://fake-jwks.com?key=value"
                    },
                    "asyncFetch": {}
                }
            }
        },
        "requirementMap": {
            "testapi.foo": {
                "providerName": "auth_provider"
            }
        }
    }
}
`,
		},
		{
			desc: "Success. Generate jwt authn filter with default jwt locations with non empty audiences.",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "testapi",
						Methods: []*apipb.Method{
							{
								Name: "foo",
							},
						},
					},
				},
				SourceInfo: &confpb.SourceInfo{
					SourceFiles: []*anypb.Any{content},
				},
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:        "auth_provider",
							Issuer:    "issuer-0",
							JwksUri:   "https://fake-jwks.com?key=value",
							Audiences: "audience1,audience2",
						},
					},
					Rules: []*confpb.AuthenticationRule{
						{
							Selector: "testapi.foo",
							Requirements: []*confpb.AuthRequirement{
								{
									ProviderId: "auth_provider",
									Audiences:  "audience3",
								},
							},
						},
					},
				},
			},
			// Service config AuthProvider has non empty audiences, envoy jwt_authn Provider.audiences uses them directly.
			// Service config AuthRequirement has non empty audiences, envoy jwt_authn requirement_map uses "provider_and_audiences
			wantJwtAuthnFilter: `{
    "name": "envoy.filters.http.jwt_authn",
    "typedConfig": {
        "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication",
        "providers": {
            "auth_provider": {
                "audiences": [
                    "audience1",
                    "audience2"
                ],
                "forward": true,
                "forwardPayloadHeader": "X-Endpoint-API-UserInfo",
                "fromHeaders": [
                    {
                        "name": "Authorization",
                        "valuePrefix": "Bearer "
                    },
                    {
                        "name": "X-Goog-Iap-Jwt-Assertion"
                    }
                ],
                "fromParams": [
                    "access_token"
                ],
                "issuer": "issuer-0",
                "payloadInMetadata": "jwt_payloads",
                "remoteJwks": {
                    "cacheDuration": "300s",
                    "httpUri": {
                        "cluster": "jwt-provider-cluster-fake-jwks.com:443",
                        "timeout": "30s",
                        "uri": "https://fake-jwks.com?key=value"
                    },
                    "asyncFetch": {}
                }
            }
        },
        "requirementMap": {
            "testapi.foo": {
                "providerAndAudiences": {
                    "providerName": "auth_provider",
                    "audiences": [
                        "audience3"
                    ]
                }
            }
        }
    }
}
`,
		},
		{
			desc: "Success. Generate jwt authn filter with disabled service name check and an empty audiences.",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "testapi",
						Methods: []*apipb.Method{
							{
								Name: "foo",
							},
						},
					},
				},
				SourceInfo: &confpb.SourceInfo{
					SourceFiles: []*anypb.Any{content},
				},
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:      "auth_provider",
							Issuer:  "issuer-0",
							JwksUri: "https://fake-jwks.com?key=value",
						},
					},
					Rules: []*confpb.AuthenticationRule{
						{
							Selector: "testapi.foo",
							Requirements: []*confpb.AuthRequirement{
								{
									ProviderId: "auth_provider",
								},
							},
						},
					},
				},
			},
			disableJwtServiceName: true,
			// With JwtAudienceServiceNameCheck is disabled,
			// Service config AuthProvider has empty "audiences", and envoy jwt_authn Provider has empty audiences too.
			wantJwtAuthnFilter: `{
    "name": "envoy.filters.http.jwt_authn",
    "typedConfig": {
        "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication",
        "providers": {
            "auth_provider": {
                "forward": true,
                "forwardPayloadHeader": "X-Endpoint-API-UserInfo",
                "fromHeaders": [
                    {
                        "name": "Authorization",
                        "valuePrefix": "Bearer "
                    },
                    {
                        "name": "X-Goog-Iap-Jwt-Assertion"
                    }
                ],
                "fromParams": [
                    "access_token"
                ],
                "issuer": "issuer-0",
                "payloadInMetadata": "jwt_payloads",
                "remoteJwks": {
                    "cacheDuration": "300s",
                    "httpUri": {
                        "cluster": "jwt-provider-cluster-fake-jwks.com:443",
                        "timeout": "30s",
                        "uri": "https://fake-jwks.com?key=value"
                    },
                    "asyncFetch": {}
                }
            }
        },
        "requirementMap": {
            "testapi.foo": {
                "providerName": "auth_provider"
            }
        }
    }
}
`,
		},
		{
			desc: "Success. Generate jwt authn filter with disabled service name check and non empty audiences.",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "testapi",
						Methods: []*apipb.Method{
							{
								Name: "foo",
							},
						},
					},
				},
				SourceInfo: &confpb.SourceInfo{
					SourceFiles: []*anypb.Any{content},
				},
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:        "auth_provider",
							Issuer:    "issuer-0",
							JwksUri:   "https://fake-jwks.com?key=value",
							Audiences: "audience1,audience2",
						},
					},
					Rules: []*confpb.AuthenticationRule{
						{
							Selector: "testapi.foo",
							Requirements: []*confpb.AuthRequirement{
								{
									ProviderId: "auth_provider",
									Audiences:  "audience3",
								},
							},
						},
					},
				},
			},
			disableJwtServiceName: true,
			// With JwtAudienceServiceNameCheck is disabled, but since "audiences" is not empty, it should not have any impact.
			wantJwtAuthnFilter: `{
    "name": "envoy.filters.http.jwt_authn",
    "typedConfig": {
        "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication",
        "providers": {
            "auth_provider": {
                "audiences": [
                    "audience1",
                    "audience2"
                ],
                "forward": true,
                "forwardPayloadHeader": "X-Endpoint-API-UserInfo",
                "fromHeaders": [
                    {
                        "name": "Authorization",
                        "valuePrefix": "Bearer "
                    },
                    {
                        "name": "X-Goog-Iap-Jwt-Assertion"
                    }
                ],
                "fromParams": [
                    "access_token"
                ],
                "issuer": "issuer-0",
                "payloadInMetadata": "jwt_payloads",
                "remoteJwks": {
                    "cacheDuration": "300s",
                    "httpUri": {
                        "cluster": "jwt-provider-cluster-fake-jwks.com:443",
                        "timeout": "30s",
                        "uri": "https://fake-jwks.com?key=value"
                    },
                    "asyncFetch": {}
                }
            }
        },
        "requirementMap": {
            "testapi.foo": {
                "providerAndAudiences": {
                    "providerName": "auth_provider",
                    "audiences": [
                        "audience3"
                    ]
                }
            }
        }
    }
}
`,
		},
		{
			desc: "Success. Generate jwt authn filter with jwt_cache_size and async_fetch fast_listener",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "testapi",
						Methods: []*apipb.Method{
							{
								Name: "foo",
							},
						},
					},
				},
				SourceInfo: &confpb.SourceInfo{
					SourceFiles: []*anypb.Any{content},
				},
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:      "auth_provider",
							Issuer:  "issuer-0",
							JwksUri: "https://fake-jwks.com?key=value",
						},
					},
					Rules: []*confpb.AuthenticationRule{
						{
							Selector: "testapi.foo",
							Requirements: []*confpb.AuthRequirement{
								{
									ProviderId: "auth_provider",
								},
							},
						},
					},
				},
			},
			jwksAsyncFetchFastListener: true,
			jwtCacheSize:               1000,
			wantJwtAuthnFilter: `{
    "name": "envoy.filters.http.jwt_authn",
    "typedConfig": {
        "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication",
        "providers": {
            "auth_provider": {
                "audiences": [
                    "https://bookstore.endpoints.project123.cloud.goog"
                ],
                "forward": true,
                "forwardPayloadHeader": "X-Endpoint-API-UserInfo",
                "fromHeaders": [
                    {
                        "name": "Authorization",
                        "valuePrefix": "Bearer "
                    },
                    {
                        "name": "X-Goog-Iap-Jwt-Assertion"
                    }
                ],
                "fromParams": [
                    "access_token"
                ],
                "issuer": "issuer-0",
                "payloadInMetadata": "jwt_payloads",
                "remoteJwks": {
                    "cacheDuration": "300s",
                    "httpUri": {
                        "cluster": "jwt-provider-cluster-fake-jwks.com:443",
                        "timeout": "30s",
                        "uri": "https://fake-jwks.com?key=value"
                    },
                    "asyncFetch": {
                      "fastListener": true
                    }
                },
                "jwtCacheConfig": {
                   "jwtCacheSize": 1000
                }
            }
        },
        "requirementMap": {
            "testapi.foo": {
                "providerName": "auth_provider"
            }
        }
    }
}
`,
		},
		{
			desc: "Success. Generate jwt authn filter with default locations and disableJwksAsyncFetch",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "testapi",
						Methods: []*apipb.Method{
							{
								Name: "foo",
							},
						},
					},
				},
				SourceInfo: &confpb.SourceInfo{
					SourceFiles: []*anypb.Any{content},
				},
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:      "auth_provider",
							Issuer:  "issuer-0",
							JwksUri: "https://fake-jwks.com",
						},
					},
					Rules: []*confpb.AuthenticationRule{
						{
							Selector: "testapi.foo",
							Requirements: []*confpb.AuthRequirement{
								{
									ProviderId: "auth_provider",
								},
							},
						},
					},
				},
			},
			disableJwksAsyncFetch: true,
			wantJwtAuthnFilter: `{
    "name": "envoy.filters.http.jwt_authn",
    "typedConfig": {
        "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication",
        "providers": {
            "auth_provider": {
                "audiences": [
                    "https://bookstore.endpoints.project123.cloud.goog"
                ],
                "forward": true,
                "forwardPayloadHeader": "X-Endpoint-API-UserInfo",
                "fromHeaders": [
                    {
                        "name": "Authorization",
                        "valuePrefix": "Bearer "
                    },
                    {
                        "name": "X-Goog-Iap-Jwt-Assertion"
                    }
                ],
                "fromParams": [
                    "access_token"
                ],
                "issuer": "issuer-0",
                "payloadInMetadata": "jwt_payloads",
                "remoteJwks": {
                    "cacheDuration": "300s",
                    "httpUri": {
                        "cluster": "jwt-provider-cluster-fake-jwks.com:443",
                        "timeout": "30s",
                        "uri": "https://fake-jwks.com"
                    }
                }
            }
        },
        "requirementMap": {
            "testapi.foo": {
                "providerName": "auth_provider"
            }
        }
    }
}
`,
		},
		{
			desc: "Success. Generate jwt authn filter with custom jwt locations",
			fakeServiceConfig: &confpb.Service{
				Name: testProjectName,
				Apis: []*apipb.Api{
					{
						Name: "testapi",
						Methods: []*apipb.Method{
							{
								Name: "foo",
							},
						},
					},
				},
				SourceInfo: &confpb.SourceInfo{
					SourceFiles: []*anypb.Any{content},
				},
				Authentication: &confpb.Authentication{
					Providers: []*confpb.AuthProvider{
						{
							Id:      "auth_provider",
							Issuer:  "issuer-0",
							JwksUri: "https://fake-jwks.com",
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
					Rules: []*confpb.AuthenticationRule{
						{
							Selector:               "testapi.foo",
							AllowWithoutCredential: true,
							Requirements: []*confpb.AuthRequirement{
								{
									ProviderId: "auth_provider",
								},
							},
						},
					},
				},
			},
			wantJwtAuthnFilter: `{
    "name": "envoy.filters.http.jwt_authn",
    "typedConfig": {
        "@type": "type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication",
        "providers": {
            "auth_provider": {
                "audiences": [
                    "https://bookstore.endpoints.project123.cloud.goog"
                ],
                "forward": true,
                "forwardPayloadHeader": "X-Endpoint-API-UserInfo",
                "fromHeaders": [
                    {
                        "name": "jwt_query_header",
                        "valuePrefix": "jwt_query_header_prefix"
                    }
                ],
                "fromParams": [
                    "jwt_query_param"
                ],
                "issuer": "issuer-0",
                "payloadInMetadata": "jwt_payloads",
                "remoteJwks": {
                    "cacheDuration": "300s",
                    "httpUri": {
                        "cluster": "jwt-provider-cluster-fake-jwks.com:443",
                        "timeout": "30s",
                        "uri": "https://fake-jwks.com"
                    },
                    "asyncFetch": {}
                }
            }
        },
        "requirementMap": {
            "testapi.foo": {
                 "requiresAny":{
                    "requirements":[
                     {
                        "providerName":"auth_provider"
                     },
                     {
                        "allowMissing":{}
                     }
                   ]
                }
            }
        }
    }
}`,
		},
	}

	for i, tc := range testData {
		opts := options.DefaultConfigGeneratorOptions()
		opts.BackendAddress = "grpc://127.0.0.0:80"
		opts.DisableJwksAsyncFetch = tc.disableJwksAsyncFetch
		opts.JwksAsyncFetchFastListener = tc.jwksAsyncFetchFastListener
		opts.DisableJwtAudienceServiceNameCheck = tc.disableJwtServiceName
		opts.JwtCacheSize = tc.jwtCacheSize
		fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(tc.fakeServiceConfig, testConfigID, opts)
		if err != nil {
			t.Fatal(err)
		}

		marshaler := &jsonpb.Marshaler{}
		gotProto, _, _ := jaFilterGenFunc(fakeServiceInfo)
		gotFilter, err := marshaler.MarshalToString(gotProto)
		if err != nil {
			t.Fatal(err)
		}

		if err := util.JsonEqual(tc.wantJwtAuthnFilter, gotFilter); err != nil {
			t.Errorf("Test Desc(%d): %s, makeJwtAuthnFilter failed, %s", i, tc.desc, err)
		}
	}
}
