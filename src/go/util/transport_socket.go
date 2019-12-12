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
	"github.com/golang/protobuf/ptypes"

	authpb "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
)

// CreateTransportSocket creates a TransportSocket
func CreateTransportSocket(hostname, rootCertsPath string) (*corepb.TransportSocket, error) {
	tlsContext, err := ptypes.MarshalAny(&authpb.UpstreamTlsContext{
		Sni: hostname,
		CommonTlsContext: &authpb.CommonTlsContext{
			ValidationContextType: &authpb.CommonTlsContext_ValidationContext{
				ValidationContext: &authpb.CertificateValidationContext{
					TrustedCa: &corepb.DataSource{
						Specifier: &corepb.DataSource_Filename{
							Filename: rootCertsPath,
						},
					},
				},
			},
		},
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
