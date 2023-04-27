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
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	tlspb "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"google.golang.org/protobuf/types/known/anypb"
)

var (
	// TLSTransportSockerName is the name of the Envoy transport socket that configures TLS.
	TLSTransportSockerName = "envoy.transport_sockets.tls"

	// DefaultClientSslFilename is the name to use when no SSL client file is provided.
	DefaultClientSslFilename = "client"
)

// ClusterTLSConfiger is a helper to set TLS config on a cluster.
type ClusterTLSConfiger struct {
	RootCertsPath      string
	ClientCertsPath    string
	ClientCipherSuites string
}

// NewClusterTLSConfigerFromOPConfig creates a ClusterTLSConfiger from
// OP service config + descriptor + ESPv2 options.
func NewClusterTLSConfigerFromOPConfig(opts options.ConfigGeneratorOptions, isBackendCluster bool) *ClusterTLSConfiger {
	if !isBackendCluster {
		return &ClusterTLSConfiger{
			RootCertsPath: opts.SslSidestreamClientRootCertsPath,
		}
	}

	return &ClusterTLSConfiger{
		RootCertsPath:      opts.SslBackendClientRootCertsPath,
		ClientCertsPath:    opts.SslBackendClientCertPath,
		ClientCipherSuites: opts.SslBackendClientCipherSuites,
	}
}

// MakeTLSConfig creates a TransportSocket with TLS config for a cluster.
func (c *ClusterTLSConfiger) MakeTLSConfig(hostname string, alpnProtocols []string) (*corepb.TransportSocket, error) {
	if c.RootCertsPath == "" {
		return nil, fmt.Errorf("root certs path cannot be empty")
	}

	sslFileName := DefaultClientSslFilename
	// Backward compatible for ESPv1
	if strings.Contains(c.ClientCertsPath, "/etc/nginx/ssl") {
		sslFileName = "backend"
	}

	commonTls, err := util.CreateCommonTlsContext(c.RootCertsPath, c.ClientCertsPath, sslFileName, "", "", c.ClientCipherSuites)
	if err != nil {
		return nil, err
	}
	if len(alpnProtocols) > 0 {
		commonTls.AlpnProtocols = alpnProtocols
	}

	tlsContext, err := anypb.New(&tlspb.UpstreamTlsContext{
		Sni:              hostname,
		CommonTlsContext: commonTls,
	})
	if err != nil {
		return nil, err
	}

	return &corepb.TransportSocket{
		Name:       TLSTransportSockerName,
		ConfigType: &corepb.TransportSocket_TypedConfig{
			TypedConfig: tlsContext,
		},
	}, nil
}
