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

package clustergen

import (
	"fmt"
	"time"

	helpers2 "github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"github.com/golang/glog"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/types/known/durationpb"
)

// JWTProviderCluster is an Envoy cluster to communicate with a remote JWKS
// provider. Each cluster talks to one remote server.
type JWTProviderCluster struct {
	ID                    string
	JWKSURI               string
	ClusterConnectTimeout time.Duration

	DNS *helpers2.ClusterDNSConfiger
	TLS *helpers2.ClusterTLSConfiger
}

// NewJWTProviderClustersFromOPConfig creates all JWTProviderCluster from
// OP service config + descriptor + ESPv2 options. It is a ClusterGeneratorOPFactory.
//
// Generates multiple clusters, one per each JWT provider address.
// Automatically de-duplicates multiple clusters with the same remote socket address.
func NewJWTProviderClustersFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) ([]ClusterGenerator, error) {
	var gens []ClusterGenerator
	dedupClusterNames := make(map[string]bool)

	for _, provider := range serviceConfig.GetAuthentication().GetProviders() {
		jwksURI, err := maybeGetJWKSURIByOpenID(provider, opts)
		if err != nil {
			return nil, err
		}

		addr, err := util.ExtractAddressFromURI(jwksURI)
		if err != nil {
			return nil, fmt.Errorf("failed to extract address from JWKS URI: %v", err)
		}

		if _, exist := dedupClusterNames[addr]; exist {
			glog.Infof("Ignoring authn provider with ID %q and JWKS URI %q because it already has a config.", provider.GetId(), jwksURI)
			continue
		}
		dedupClusterNames[addr] = true

		gen := &JWTProviderCluster{
			ID:                    provider.GetId(),
			JWKSURI:               jwksURI,
			ClusterConnectTimeout: opts.ClusterConnectTimeout,
			DNS:                   helpers2.NewClusterDNSConfigerFromOPConfig(opts),
			TLS:                   helpers2.NewClusterTLSConfigerFromOPConfig(opts, false),
		}
		gens = append(gens, gen)
	}

	return gens, nil
}

// maybeGetJWKSURIByOpenID will return the best option for JWKS URI, or an error
// if it can't find one.
func maybeGetJWKSURIByOpenID(provider *servicepb.AuthProvider, opts options.ConfigGeneratorOptions) (string, error) {
	if provider.GetJwksUri() != "" {
		return provider.GetJwksUri(), nil
	}

	if opts.DisableOidcDiscovery {
		return "", fmt.Errorf("error processing authentication provider %q: "+
			"jwks_uri is empty, but OpenID Connect Discovery is disabled via startup option. "+
			"Consider specifying the jwks_uri in the provider config", provider.GetId())
	}

	glog.Infof("jwks_uri is empty for provider (%v), using OpenID Connect Discovery protocol (remote RPC during config gen)", provider.GetId())
	jwksURIByOpenID, err := util.ResolveJwksUriUsingOpenID(provider.GetIssuer())
	if err != nil {
		return "", fmt.Errorf("error processing authentication provider (%v): failed OpenID Connect Discovery protocol: %v", provider.Id, err)
	}

	return jwksURIByOpenID, nil
}

// GetName implements the ClusterGenerator interface.
func (c *JWTProviderCluster) GetName() string {
	return c.ID
}

// GenConfig implements the ClusterGenerator interface.
func (c *JWTProviderCluster) GenConfig() (*clusterpb.Cluster, error) {
	addr, err := util.ExtractAddressFromURI(c.JWKSURI)
	if err != nil {
		return nil, fmt.Errorf("failed to extract address from JWKS URI: %v", err)
	}

	scheme, hostname, port, _, err := util.ParseURI(c.JWKSURI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWKS URI: %v", err)
	}

	config := &clusterpb.Cluster{
		Name:                 fmt.Sprintf("jwt-provider-cluster-%s", addr),
		LbPolicy:             clusterpb.Cluster_ROUND_ROBIN,
		ConnectTimeout:       durationpb.New(c.ClusterConnectTimeout),
		DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
		ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
		LoadAssignment:       util.CreateLoadAssignment(hostname, port),
	}
	if scheme == "https" {
		transportSocket, err := c.TLS.MakeTLSConfig(hostname, nil)
		if err != nil {
			return nil, err
		}
		config.TransportSocket = transportSocket
	}

	if err := helpers2.MaybeAddDNSResolver(c.DNS, config); err != nil {
		return nil, err
	}

	return config, nil
}
