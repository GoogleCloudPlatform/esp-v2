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

func TestCreateDownstreamTransportSocket(t *testing.T) {
	testData := []struct {
		desc                string
		sslPath             string
		sslRootCertPath     string
		sslMinimumProtocol  string
		sslMaximumProtocol  string
		cipherSuites        string
		wantTransportSocket string
	}{
		{
			desc:               "Downstream Transport Socket for TLS",
			sslPath:            "/etc/ssl/endpoints/",
			sslMinimumProtocol: "TLSv1.1",
			wantTransportSocket: `{
				"name":"envoy.transport_sockets.tls",
				"typedConfig":{
					"@type":"type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext",
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
						],
						"tlsParams":{
							"tlsMinimumProtocolVersion":"TLSv1_1"
						}
					}
				}
			} `,
		},
		{
			desc:               "Downstream Transport Socket for mTLS",
			sslPath:            "/etc/ssl/endpoints/",
			sslRootCertPath:    "/etc/ssl/endpoints/root.crt",
			sslMinimumProtocol: "TLSv1.1",
			wantTransportSocket: `{
				"name": "envoy.transport_sockets.tls",
				"typedConfig": {
					"@type": "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext",
					"commonTlsContext": {
						"alpnProtocols": [
							"h2",
							"http/1.1"
						],
						"tlsCertificates": [
							{
								"certificateChain": {
									"filename": "/etc/ssl/endpoints/server.crt"
								},
								"privateKey": {
									"filename": "/etc/ssl/endpoints/server.key"
								}
							}
						],
						"tlsParams": {
							"tlsMinimumProtocolVersion": "TLSv1_1"
						},
						"validationContext": {
							"trustedCa": {
								"filename": "/etc/ssl/endpoints/root.crt"
							}
						}
					},
					"requireClientCertificate": true
				}
			}`,
		},
		{
			desc:               "Downstream Transport Socket for TLS, with version requirements",
			sslPath:            "/etc/ssl/endpoints/",
			sslMinimumProtocol: "TLSv1.1",
			sslMaximumProtocol: "TLSv1.3",
			cipherSuites:       "ECDHE-ECDSA-AES128-GCM-SHA256,ECDHE-RSA-AES128-GCM-SHA256",
			wantTransportSocket: `{
				"name":"envoy.transport_sockets.tls",
				"typedConfig":{
					"@type":"type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext",
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
						],
						"tlsParams":{
							"cipherSuites":[
								"ECDHE-ECDSA-AES128-GCM-SHA256",
								"ECDHE-RSA-AES128-GCM-SHA256"
							],
							"tlsMaximumProtocolVersion":"TLSv1_3",
							"tlsMinimumProtocolVersion":"TLSv1_1"
						}
					}
				}
			} `,
		},
		{
			desc:               "Downstream Transport Socket for TLS, for legacy ESPv1",
			sslPath:            "/etc/nginx/ssl",
			sslMaximumProtocol: "TLSv1.3",
			wantTransportSocket: `{
				"name":"envoy.transport_sockets.tls",
				"typedConfig":{
					"@type":"type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext",
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
						],
						"tlsParams":{
							"tlsMaximumProtocolVersion":"TLSv1_3"
						}
					}
				}
			}`,
		},
	}

	for i, tc := range testData {
		gotTransportSocket, err := CreateDownstreamTransportSocket(tc.sslPath, tc.sslRootCertPath, tc.sslMinimumProtocol, tc.sslMaximumProtocol, tc.cipherSuites)
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
