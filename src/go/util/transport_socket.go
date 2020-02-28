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
	"fmt"

	"github.com/golang/protobuf/ptypes"

	authpb "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
)

// CreateUpstreamTransportSocket creates a TransportSocket for Upstream
func CreateUpstreamTransportSocket(hostname, rootCertsPath string, alpn_protocols []string) (*corepb.TransportSocket, error) {
	common_tls := &authpb.CommonTlsContext{
		ValidationContextType: &authpb.CommonTlsContext_ValidationContext{
			ValidationContext: &authpb.CertificateValidationContext{
				TrustedCa: &corepb.DataSource{
					Specifier: &corepb.DataSource_Filename{
						Filename: rootCertsPath,
					},
				},
			},
		},
	}
	if len(alpn_protocols) > 0 {
		common_tls.AlpnProtocols = alpn_protocols
	}

	tlsContext, err := ptypes.MarshalAny(&authpb.UpstreamTlsContext{
		Sni:              hostname,
		CommonTlsContext: common_tls,
	},
	)
	if err != nil {
		return nil, err
	}
	return &corepb.TransportSocket{
		Name: TLSTransportSocket,
		ConfigType: &corepb.TransportSocket_TypedConfig{
			TypedConfig: tlsContext,
		},
	}, nil
}

// CreateDownstreamTransportSocket creates a TransportSocket for Downstream
func CreateDownstreamTransportSocket(sslPath string) (*corepb.TransportSocket, error) {
	common_tls := &authpb.CommonTlsContext{
		TlsCertificates: []*authpb.TlsCertificate{
			{
				CertificateChain: &corepb.DataSource{
					Specifier: &corepb.DataSource_Filename{
						Filename: fmt.Sprintf("%s/server.crt", sslPath),
					},
				},
				PrivateKey: &corepb.DataSource{
					Specifier: &corepb.DataSource_Filename{
						Filename: fmt.Sprintf("%s/server.key", sslPath),
					},
				},
			},
		},
	}
	common_tls.AlpnProtocols = []string{"h2", "http/1.1"}

	tlsContext, err := ptypes.MarshalAny(&authpb.DownstreamTlsContext{
		CommonTlsContext: common_tls,
	},
	)
	if err != nil {
		return nil, err
	}
	return &corepb.TransportSocket{
		Name: TLSTransportSocket,
		ConfigType: &corepb.TransportSocket_TypedConfig{
			TypedConfig: tlsContext,
		},
	}, nil
}
