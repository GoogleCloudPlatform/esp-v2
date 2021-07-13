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
	"github.com/golang/protobuf/ptypes/wrappers"

	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	tlspb "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
)

const (
	defaultServerSslFilename = "server"
	defaultClientSslFilename = "client"
)

var (
	tlsProtocolVersionMap = map[string]tlspb.TlsParameters_TlsProtocol{
		"TLSv1.0": tlspb.TlsParameters_TLSv1_0,
		"TLSv1.1": tlspb.TlsParameters_TLSv1_1,
		"TLSv1.2": tlspb.TlsParameters_TLSv1_2,
		"TLSv1.3": tlspb.TlsParameters_TLSv1_3,
	}
)

// CreateUpstreamTransportSocket creates a TransportSocket for Upstream
func CreateUpstreamTransportSocket(hostname, rootCertsPath, sslClientPath string, alpnProtocols []string, cipherSuites string) (*corepb.TransportSocket, error) {
	if rootCertsPath == "" {
		return nil, fmt.Errorf("root certs path cannot be empty.")
	}

	sslFileName := defaultClientSslFilename
	// Backward compatible for ESPv1
	if strings.Contains(sslClientPath, "/etc/nginx/ssl") {
		sslFileName = "backend"
	}

	commonTls, err := createCommonTlsContext(rootCertsPath, sslClientPath, sslFileName, "", "", cipherSuites)
	if err != nil {
		return nil, err
	}
	if len(alpnProtocols) > 0 {
		commonTls.AlpnProtocols = alpnProtocols
	}

	tlsContext, err := ptypes.MarshalAny(&tlspb.UpstreamTlsContext{
		Sni:              hostname,
		CommonTlsContext: commonTls,
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
func CreateDownstreamTransportSocket(sslServerPath, sslServerRootPath, sslMinimumProtocol, sslMaximumProtocol string, cipherSuites string) (*corepb.TransportSocket, error) {
	if sslServerPath == "" {
		return nil, fmt.Errorf("SSL path cannot be empty.")
	}

	sslFileName := defaultServerSslFilename
	// Backward compatible for ESPv1
	if strings.Contains(sslServerPath, "/etc/nginx/ssl") {
		sslFileName = "nginx"
	}

	commonTls, err := createCommonTlsContext(sslServerRootPath, sslServerPath, sslFileName, sslMinimumProtocol, sslMaximumProtocol, cipherSuites)
	if err != nil {
		return nil, err
	}
	commonTls.AlpnProtocols = []string{"h2", "http/1.1"}
	downstreamTlsContext := &tlspb.DownstreamTlsContext{
		CommonTlsContext: commonTls,
	}
	if sslServerRootPath != "" {
		downstreamTlsContext.RequireClientCertificate = &wrappers.BoolValue{
			Value: true,
		}
	}
	tlsContext, err := ptypes.MarshalAny(downstreamTlsContext)
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

func createCommonTlsContext(rootCertsPath, sslPath, sslFileName, sslMinimumProtocol, sslMaximumProtocol string, cipherSuites string) (*tlspb.CommonTlsContext, error) {
	commonTls := &tlspb.CommonTlsContext{}
	// Add TLS certificate
	if sslPath != "" && sslFileName != "" {
		if !strings.HasSuffix(sslPath, "/") {
			sslPath = fmt.Sprintf("%s/", sslPath)
		}

		commonTls.TlsCertificates = []*tlspb.TlsCertificate{
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
		commonTls.ValidationContextType = &tlspb.CommonTlsContext_ValidationContext{
			ValidationContext: &tlspb.CertificateValidationContext{
				TrustedCa: &corepb.DataSource{
					Specifier: &corepb.DataSource_Filename{
						Filename: rootCertsPath,
					},
				},
			},
		}
	}

	if sslMinimumProtocol != "" || sslMaximumProtocol != "" || cipherSuites != "" {
		commonTls.TlsParams = &tlspb.TlsParameters{}
		if minVersion, ok := tlsProtocolVersionMap[sslMinimumProtocol]; ok {
			commonTls.TlsParams.TlsMinimumProtocolVersion = minVersion
		}
		if maxVersion, ok := tlsProtocolVersionMap[sslMaximumProtocol]; ok {
			commonTls.TlsParams.TlsMaximumProtocolVersion = maxVersion
		}

		if cipherSuites != "" {
			cipherSuitesList := strings.Split(cipherSuites, ",")
			commonTls.TlsParams.CipherSuites = cipherSuitesList
		}
	}

	return commonTls, nil
}
