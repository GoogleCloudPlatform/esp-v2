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

const (
	// TODO(jilinxia): instead of using test Jwt from
	// https://github.com/istio/istio/blob/master/security/tools/jwt/samples/demo.jwt
	// implement a mock jwt server.

	FakeEchoConfig = `
	  {
      "name": "echo-api.endpoints.cloudesf-testing.cloud.goog",
      "title": "Endpoints Example",
      "apis": [
        {
          "name": "1.echo_api_endpoints_cloudesf_testing_cloud_goog",
          "version": "1.0.0"
        }
      ],
      "http": {
        "rules": [
          {
            "selector": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth_info_google_jwt",
            "get": "/auth/info/googlejwt"
          },
          {
            "selector": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth0",
            "get": "/auth/info/auth0"
          },
          {
            "selector": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo",
            "post": "/echo",
            "body": "message"
          }
        ]
      },
      "authentication": {
        "rules": [
          {
            "selector": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth_info_google_jwt",
            "requirements": [
              {
                "providerId": "google_jwt"
              }
            ]
          },
          {
            "selector": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Auth0",
            "requirements": [
              {
                "providerId": "google_jwt",
                "audiences": "admin.cloud.goog"
              }
            ]
          },
          {
            "selector": "1.echo_api_endpoints_cloudesf_testing_cloud_goog.Echo"
          }
        ],
        "providers": [
          {
            "id": "google_jwt",
            "issuer": "testing@secure.istio.io",
            "jwksUri": "https://raw.githubusercontent.com/istio/istio/master/security/tools/jwt/samples/jwks.json"
          }
        ]
      }
    }
  `
)
