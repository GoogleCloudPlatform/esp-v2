package helpers

import (
	"fmt"

	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
)

var (
	// TLSTransportSockerName is the name of the Envoy transport socket that configures TLS.
	TLSTransportSockerName = "envoy.transport_sockets.tls"

	// DefaultClientSslFilename is the name to use when no SSL client file is provided.
	DefaultClientSslFilename = "client"
)

// ClusterTLSConfiger is a helper to set TLS config on a cluster.
type ClusterTLSConfiger struct {
	RootCertsPath string

	// TODO(nareddyt): Only set these 2 for backend cluster, no other ones.
	ClientCertsPath    string
	ClientCipherSuites string
}

// MakeTLSConfig creates a TransportSocket with TLS config for a cluster.
func (c *ClusterTLSConfiger) MakeTLSConfig(hostname string, alpnProtocols []string) (*corepb.TransportSocket, error) {
	if c.RootCertsPath == "" {
		return nil, fmt.Errorf("root certs path cannot be empty")
	}

	// TODO(nareddyt): Uncomment when util directory change is in PR.
	//sslFileName := DefaultClientSslFilename
	//// Backward compatible for ESPv1
	//if strings.Contains(c.ClientCertsPath, "/etc/nginx/ssl") {
	//	sslFileName = "backend"
	//}
	//
	//commonTls, err := util.CreateCommonTlsContext(c.RootCertsPath, c.ClientCertsPath, sslFileName, "", "", c.ClientCipherSuites)
	//if err != nil {
	//	return nil, err
	//}
	//if len(alpnProtocols) > 0 {
	//	commonTls.AlpnProtocols = alpnProtocols
	//}
	//
	//tlsContext, err := anypb.New(&tlspb.UpstreamTlsContext{
	//	Sni:              hostname,
	//	CommonTlsContext: commonTls,
	//})
	//if err != nil {
	//	return nil, err
	//}

	return &corepb.TransportSocket{
		Name: TLSTransportSockerName,
		ConfigType: &corepb.TransportSocket_TypedConfig{
			//TypedConfig: tlsContext,
		},
	}, nil
}
