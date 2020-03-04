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
	"strings"

	"github.com/golang/protobuf/ptypes"

	authpb "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
)

const (
	defaultServerSslFilename  = "server"
	defaultBackendSslFilename = "backend"
)

// CreateUpstreamTransportSocket creates a TransportSocket for Upstream
func CreateUpstreamTransportSocket(hostname, rootCertsPath, sslBackendPath string, alpnProtocols []string) (*corepb.TransportSocket, error) {
	if rootCertsPath == "" {
		return nil, fmt.Errorf("root certs path cannot be empty.")
	}

	common_tls, err := createCommonTlsContext(rootCertsPath, sslBackendPath, defaultBackendSslFilename)
	if err != nil {
		return nil, err
	}
	if len(alpnProtocols) > 0 {
		common_tls.AlpnProtocols = alpnProtocols
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
func CreateDownstreamTransportSocket(sslServerPath string) (*corepb.TransportSocket, error) {
	if sslServerPath == "" {
		return nil, fmt.Errorf("SSL path cannot be empty.")
	}

	sslFileName := defaultServerSslFilename
	// Backward compatible for ESPv1
	if strings.Contains(sslServerPath, "/etc/nginx/ssl") {
		sslFileName = "nginx"
	}

	common_tls, err := createCommonTlsContext("", sslServerPath, sslFileName)
	if err != nil {
		return nil, err
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

func createCommonTlsContext(rootCertsPath, sslPath, sslFileName string) (*authpb.CommonTlsContext, error) {
	common_tls := &authpb.CommonTlsContext{}
	// Add TLS certificate
	if sslPath != "" && sslFileName != "" {
		if !strings.HasSuffix(sslPath, "/") {
			sslPath = fmt.Sprintf("%s/", sslPath)
		}

		common_tls.TlsCertificates = []*authpb.TlsCertificate{
			{
				CertificateChain: &corepb.DataSource{
					Specifier: &corepb.DataSource_Filename{
						Filename: fmt.Sprintf("%s%s.crt", sslPath, sslFileName),
					},
				},
				PrivateKey: &corepb.DataSource{
					Specifier: &corepb.DataSource_Filename{
						Filename: fmt.Sprintf("%s%s.key", sslPath, sslFileName),
					},
				},
			},
		}
	}

	// Add Validation Context
	if rootCertsPath != "" {
		common_tls.ValidationContextType = &authpb.CommonTlsContext_ValidationContext{
			ValidationContext: &authpb.CertificateValidationContext{
				TrustedCa: &corepb.DataSource{
					Specifier: &corepb.DataSource_Filename{
						Filename: rootCertsPath,
					},
				},
			},
		}
	}

	return common_tls, nil
}
