// Copyright 2023 Google LLC
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

package helpers

import (
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/imdario/mergo"
)

func TestNewClusterTLSConfigerFromOPConfig(t *testing.T) {
	testData := []struct {
		desc                string
		opts                options.ConfigGeneratorOptions
		isBackendCluster    bool
		hostname            string
		alpnProtocols       []string
		wantTransportSocket string
	}{
		{
			desc: "Upstream Transport Socket for TLS",
			opts: options.ConfigGeneratorOptions{
				SslBackendClientRootCertsPath: "/etc/ssl/certs/ca-certificates.crt",
				SslBackendClientCertPath:      "",
				SslBackendClientCipherSuites:  "ECDHE-ECDSA-AES128-GCM-SHA256,ECDHE-RSA-AES128-GCM-SHA256",
			},
			isBackendCluster: true,
			hostname:         "https://echo-http-12345-uc.a.run.app",
			alpnProtocols:    []string{"h2"},
			wantTransportSocket: `
{
   "name":"envoy.transport_sockets.tls",
   "typedConfig":{
      "@type":"type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext",
      "commonTlsContext":{
         "alpnProtocols":[
            "h2"
         ],
         "tlsParams":{
            "cipherSuites":[
               "ECDHE-ECDSA-AES128-GCM-SHA256",
               "ECDHE-RSA-AES128-GCM-SHA256"
            ]
         },
         "validationContext":{
            "trustedCa":{
               "filename":"/etc/ssl/certs/ca-certificates.crt"
            }
         }
      },
      "sni":"https://echo-http-12345-uc.a.run.app"
   }
}
`,
		},
		{
			desc: "Upstream Transport Socket for mTLS",
			opts: options.ConfigGeneratorOptions{
				SslBackendClientRootCertsPath: "/etc/ssl/certs/ca-certificates.crt",
				SslBackendClientCertPath:      "/etc/endpoint/ssl/",
			},
			isBackendCluster: true,
			hostname:         "https://echo-http-12345-uc.a.run.app",
			alpnProtocols:    []string{"h2"},
			wantTransportSocket: `
{
   "name":"envoy.transport_sockets.tls",
   "typedConfig":{
      "@type":"type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext",
      "commonTlsContext":{
         "alpnProtocols":[
            "h2"
         ],
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
      "sni":"https://echo-http-12345-uc.a.run.app"
   }
}
`,
		},
		{
			desc: "Upstream Transport Socket for mTLS, for legacy ESPv1",
			opts: options.ConfigGeneratorOptions{
				SslBackendClientRootCertsPath: "/etc/ssl/certs/ca-certificates.crt",
				SslBackendClientCertPath:      "/etc/nginx/ssl",
			},
			isBackendCluster: true,
			hostname:         "https://echo-http-12345-uc.a.run.app",
			alpnProtocols:    []string{"h2"},
			wantTransportSocket: `
{
   "name":"envoy.transport_sockets.tls",
   "typedConfig":{
      "@type":"type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext",
      "commonTlsContext":{
         "alpnProtocols":[
            "h2"
         ],
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
      "sni":"https://echo-http-12345-uc.a.run.app"
   }
}
`,
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			opts := options.DefaultConfigGeneratorOptions()
			if err := mergo.Merge(&opts, tc.opts, mergo.WithOverride); err != nil {
				t.Fatalf("Merge() of test opts into default opts got err: %v", err)
			}

			configer := NewClusterTLSConfigerFromOPConfig(tc.opts, tc.isBackendCluster)
			gotTransportSocket, err := configer.MakeTLSConfig(tc.hostname, tc.alpnProtocols)
			if err != nil {
				t.Fatal(err)
			}
			gotConfig, err := util.ProtoToJson(gotTransportSocket)
			if err != nil {
				t.Fatal(err)
			}
			if err := util.JsonEqual(tc.wantTransportSocket, gotConfig); err != nil {
				t.Errorf("NewClusterTLSConfigerFromOPConfig failed,\n %v", err)
			}
		})
	}
}
