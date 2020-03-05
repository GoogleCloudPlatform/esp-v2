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

package util

import (
	"testing"

	"github.com/golang/protobuf/jsonpb"
)

func TestCreateUpstreamTransportSocket(t *testing.T) {
	testData := []struct {
		hostName            string
		desc                string
		rootCertsPath       string
		sslBackendPath      string
		alpnProtocols       []string
		wantTransportSocket string
	}{
		{
			desc:          "Upstream Transport Socket for TLS",
			hostName:      "https://echo-http-12345-uc.a.run.app",
			rootCertsPath: "/etc/ssl/certs/ca-certificates.crt",
			alpnProtocols: []string{"h2"},
			wantTransportSocket: `{
				"name":"envoy.transport_sockets.tls",
				"typedConfig":{
					"@type":"type.googleapis.com/envoy.api.v2.auth.UpstreamTlsContext",
					"commonTlsContext":{
						"alpnProtocols":["h2"],
						"validationContext":{
							"trustedCa":{
								"filename":"/etc/ssl/certs/ca-certificates.crt"
							}
						}
					},
				  "sni":"https://echo-http-12345-uc.a.run.app"
				}
			}`,
		},
		{
			desc:           "Upstream Transport Socket for mTLS",
			hostName:       "https://echo-http-12345-uc.a.run.app",
			rootCertsPath:  "/etc/ssl/certs/ca-certificates.crt",
			sslBackendPath: "/etc/endpoint/ssl/",
			alpnProtocols:  []string{"h2"},
			wantTransportSocket: `{
				"name":"envoy.transport_sockets.tls",
				"typedConfig":{
					"@type":"type.googleapis.com/envoy.api.v2.auth.UpstreamTlsContext",
					"commonTlsContext":{
						"alpnProtocols":["h2"],
						"tlsCertificates":[
							{
								"certificateChain":{
									"filename":"/etc/endpoint/ssl/client.crt"
								},
								"privateKey":{
									"filename":"/etc/endpoint/ssl/client.key"
								}
							}
						],
						"validationContext":{
							"trustedCa":{
								"filename":"/etc/ssl/certs/ca-certificates.crt"
								}
							}
						},
						"sni":"https://echo-http-12345-uc.a.run.app"}}`,
		},
		{
			desc:           "Upstream Transport Socket for mTLS, for legacy ESPv1",
			hostName:       "https://echo-http-12345-uc.a.run.app",
			rootCertsPath:  "/etc/ssl/certs/ca-certificates.crt",
			sslBackendPath: "/etc/nginx/ssl",
			alpnProtocols:  []string{"h2"},
			wantTransportSocket: `{
				"name":"envoy.transport_sockets.tls",
				"typedConfig":{
					"@type":"type.googleapis.com/envoy.api.v2.auth.UpstreamTlsContext",
					"commonTlsContext":{
						"alpnProtocols":["h2"],
						"tlsCertificates":[
							{
								"certificateChain":{
									"filename":"/etc/nginx/ssl/backend.crt"
								},
								"privateKey":{
									"filename":"/etc/nginx/ssl/backend.key"
								}
							}
						],
						"validationContext":{
							"trustedCa":{
								"filename":"/etc/ssl/certs/ca-certificates.crt"
								}
							}
						},
						"sni":"https://echo-http-12345-uc.a.run.app"}}`,
		},
	}

	for i, tc := range testData {
		gotTransportSocket, err := CreateUpstreamTransportSocket(tc.hostName, tc.rootCertsPath, tc.sslBackendPath, tc.alpnProtocols)
		if err != nil {
			t.Fatal(err)
		}
		marshaler := &jsonpb.Marshaler{}
		gotConfig, err := marshaler.MarshalToString(gotTransportSocket)
		if err != nil {
			t.Fatal(err)
		}
		if err := JsonEqual(tc.wantTransportSocket, gotConfig); err != nil {
			t.Errorf("Test Desc(%d): %s, CreateUpstreamTransportSocket failed,\n %v", i, tc.desc, err)
		}
	}
}

func TestCreateDownstreamTransportSocket(t *testing.T) {
	testData := []struct {
		desc                string
		sslPath             string
		wantTransportSocket string
	}{
		{
			desc:    "Downstream Transport Socket for TLS",
			sslPath: "/etc/ssl/endpoints/",
			wantTransportSocket: `{
				"name":"envoy.transport_sockets.tls",
				"typedConfig":{
					"@type":"type.googleapis.com/envoy.api.v2.auth.DownstreamTlsContext",
					"commonTlsContext":{
						"alpnProtocols":["h2","http/1.1"],
						"tlsCertificates":[
							{
								"certificateChain":{
									"filename":"/etc/ssl/endpoints/server.crt"
								},
								"privateKey":{
									"filename":"/etc/ssl/endpoints/server.key"
								}
							}
						]
					}
				}
			} `,
		},
		{
			desc:    "Downstream Transport Socket for TLS, for legacy ESPv1",
			sslPath: "/etc/nginx/ssl",
			wantTransportSocket: `{
				"name":"envoy.transport_sockets.tls",
				"typedConfig":{
					"@type":"type.googleapis.com/envoy.api.v2.auth.DownstreamTlsContext",
					"commonTlsContext":{
						"alpnProtocols":["h2","http/1.1"],
						"tlsCertificates":[
							{
								"certificateChain":{
									"filename":"/etc/nginx/ssl/nginx.crt"
								},
								"privateKey":{
									"filename":"/etc/nginx/ssl/nginx.key"
								}
							}
						]
					}
				}
			}`,
		},
	}

	for i, tc := range testData {
		gotTransportSocket, err := CreateDownstreamTransportSocket(tc.sslPath)
		if err != nil {
			t.Fatal(err)
		}
		marshaler := &jsonpb.Marshaler{}
		gotConfig, err := marshaler.MarshalToString(gotTransportSocket)
		if err != nil {
			t.Fatal(err)
		}
		if err := JsonEqual(tc.wantTransportSocket, gotConfig); err != nil {
			t.Errorf("Test Desc(%d): %s, CreateDownstreamTransportSocket failed,\n %v", i, tc.desc, err)
		}
	}
}
