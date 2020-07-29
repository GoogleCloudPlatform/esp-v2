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

package configgenerator

import (
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	sc "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
)

// MakeClusters provides dynamic cluster settings for Envoy
// This must be called before MakeListeners.
func MakeClusters(serviceInfo *sc.ServiceInfo) ([]*clusterpb.Cluster, error) {
	var clusters []*clusterpb.Cluster
	backendCluster, err := makeCatchAllBackendCluster(serviceInfo)
	if err != nil {
		return nil, err
	}
	if backendCluster != nil {
		clusters = append(clusters, backendCluster)
	}

	metadataCluster, err := makeMetadataCluster(serviceInfo)
	if err != nil {
		return nil, err
	}
	if metadataCluster != nil {
		clusters = append(clusters, metadataCluster)
	}

	if serviceInfo.Options.ServiceAccountKey != "" {
		saGenSaGenTokenCluster, err := makeSaGenTokenCluster(serviceInfo)
		if err != nil {
			return nil, err
		}

		if saGenSaGenTokenCluster != nil {
			clusters = append(clusters, saGenSaGenTokenCluster)
		}
	}

	iamCluster, err := makeIamCluster(serviceInfo)
	if err != nil {
		return nil, err
	}
	if iamCluster != nil {
		clusters = append(clusters, iamCluster)
	}

	// Note: makeServiceControlCluster should be called before makeListener
	// as makeServiceControlFilter is using m.serviceControlURI assigned by
	// makeServiceControlCluster
	scCluster, err := makeServiceControlCluster(serviceInfo)
	if err != nil {
		return nil, err
	}
	if scCluster != nil {
		clusters = append(clusters, scCluster)
	}

	brClusters, err := makeBackendRoutingClusters(serviceInfo)
	if err != nil {
		return nil, err
	}
	if brClusters != nil {
		clusters = append(clusters, brClusters...)
	}

	providerClusters, err := makeJwtProviderClusters(serviceInfo)
	if err != nil {
		return nil, err
	}

	if providerClusters != nil {
		clusters = append(clusters, providerClusters...)
	}

	if serviceInfo.Options.DnsResolverAddresses != "" {
		if err = addDnsResolversToClusters(serviceInfo.Options.DnsResolverAddresses, clusters); err != nil {
			return nil, fmt.Errorf("fail to add dns resovlers to clusters : %v", err)
		}
	}

	glog.Infof("generate clusters: %v", clusters)
	return clusters, nil
}

func addDnsResolversToClusters(dnsResolverAddresses string, clusters []*clusterpb.Cluster) error {
	dnsResolvers, err := util.DnsResolvers(dnsResolverAddresses)
	if err != nil {
		return err
	}

	for _, cluster := range clusters {
		cluster.DnsResolvers = dnsResolvers
	}

	return nil
}

func makeMetadataCluster(serviceInfo *sc.ServiceInfo) (*clusterpb.Cluster, error) {
	scheme, hostname, port, _, err := util.ParseURI(serviceInfo.Options.MetadataURL)
	if err != nil {
		return nil, err
	}

	connectTimeoutProto := ptypes.DurationProto(serviceInfo.Options.ClusterConnectTimeout)
	c := &clusterpb.Cluster{
		Name:           util.MetadataServerClusterName,
		LbPolicy:       clusterpb.Cluster_ROUND_ROBIN,
		ConnectTimeout: connectTimeoutProto,
		ClusterDiscoveryType: &clusterpb.Cluster_Type{
			Type: clusterpb.Cluster_STRICT_DNS,
		},
		LoadAssignment: util.CreateLoadAssignment(hostname, port),
	}

	if scheme == "https" {
		transportSocket, err := util.CreateUpstreamTransportSocket(hostname, serviceInfo.Options.RootCertsPath, "", nil)
		if err != nil {
			return nil, fmt.Errorf("error marshaling tls context to transport_socket config for cluster %s, err=%v",
				c.Name, err)
		}
		c.TransportSocket = transportSocket
	}

	return c, nil
}

func makeSaGenTokenCluster(serviceInfo *sc.ServiceInfo) (*clusterpb.Cluster, error) {
	connectTimeoutProto := ptypes.DurationProto(serviceInfo.Options.ClusterConnectTimeout)
	c := &clusterpb.Cluster{
		Name:           util.SaGenTokenClusterName,
		LbPolicy:       clusterpb.Cluster_ROUND_ROBIN,
		ConnectTimeout: connectTimeoutProto,
		ClusterDiscoveryType: &clusterpb.Cluster_Type{
			Type: clusterpb.Cluster_STRICT_DNS,
		},
		LoadAssignment: util.CreateLoadAssignment("127.0.0.1", uint32(serviceInfo.Options.SaGenTokenPort)),
	}
	return c, nil
}

func makeIamCluster(serviceInfo *sc.ServiceInfo) (*clusterpb.Cluster, error) {
	if serviceInfo.Options.ServiceControlCredentials == nil && serviceInfo.Options.BackendAuthCredentials == nil {
		return nil, nil
	}
	scheme, hostname, port, _, err := util.ParseURI(serviceInfo.Options.IamURL)
	if err != nil {
		return nil, err
	}

	connectTimeoutProto := ptypes.DurationProto(serviceInfo.Options.ClusterConnectTimeout)
	c := &clusterpb.Cluster{
		Name:            util.IamServerClusterName,
		LbPolicy:        clusterpb.Cluster_ROUND_ROBIN,
		DnsLookupFamily: clusterpb.Cluster_V4_ONLY,
		ConnectTimeout:  connectTimeoutProto,
		ClusterDiscoveryType: &clusterpb.Cluster_Type{
			Type: clusterpb.Cluster_STRICT_DNS,
		},
		LoadAssignment: util.CreateLoadAssignment(hostname, port),
	}

	if scheme == "https" {
		transportSocket, err := util.CreateUpstreamTransportSocket(hostname, serviceInfo.Options.RootCertsPath, "", nil)
		if err != nil {
			return nil, fmt.Errorf("error marshaling tls context to transport_socket config for cluster %s, err=%v",
				c.Name, err)
		}
		c.TransportSocket = transportSocket
	}

	return c, nil
}

func makeJwtProviderClusters(serviceInfo *sc.ServiceInfo) ([]*clusterpb.Cluster, error) {
	var providerClusters []*clusterpb.Cluster
	authn := serviceInfo.ServiceConfig().GetAuthentication()
	generatedClusters := map[string]bool{}

	for _, provider := range authn.GetProviders() {
		jwksUri := provider.GetJwksUri()
		clusterName, err := util.ExtraAddressFromURI(jwksUri)
		if err != nil {
			return nil, err
		}
		if ok, _ := generatedClusters[clusterName]; ok {
			continue
		}
		generatedClusters[clusterName] = true

		scheme, hostname, port, _, err := util.ParseURI(jwksUri)

		if err != nil {
			return nil, fmt.Errorf("Fail to parse jwksUri %s with error %v", jwksUri, err)
		}

		connectTimeoutProto := ptypes.DurationProto(serviceInfo.Options.ClusterConnectTimeout)

		c := &clusterpb.Cluster{
			Name:           clusterName,
			LbPolicy:       clusterpb.Cluster_ROUND_ROBIN,
			ConnectTimeout: connectTimeoutProto,
			// Note: It may not be V4.
			DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
			ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
			LoadAssignment:       util.CreateLoadAssignment(hostname, port),
		}
		if scheme == "https" {
			transportSocket, err := util.CreateUpstreamTransportSocket(hostname, serviceInfo.Options.RootCertsPath, "", nil)
			if err != nil {
				return nil, fmt.Errorf("error marshaling tls context to transport_socket config for cluster %s, err=%v",
					c.Name, err)
			}
			c.TransportSocket = transportSocket
		}

		providerClusters = append(providerClusters, c)
	}
	return providerClusters, nil
}

func makeBackendCluster(opt *options.ConfigGeneratorOptions, brc *sc.BackendRoutingCluster) (*clusterpb.Cluster, error) {
	c := &clusterpb.Cluster{
		Name:                 brc.ClusterName,
		LbPolicy:             clusterpb.Cluster_ROUND_ROBIN,
		ConnectTimeout:       ptypes.DurationProto(opt.ClusterConnectTimeout),
		ClusterDiscoveryType: &clusterpb.Cluster_Type{Type: clusterpb.Cluster_LOGICAL_DNS},
		LoadAssignment:       util.CreateLoadAssignment(brc.Hostname, brc.Port),
	}

	isHttp2 := brc.Protocol == util.GRPC || brc.Protocol == util.HTTP2

	if brc.UseTLS {
		var alpnProtocols []string
		if isHttp2 {
			alpnProtocols = []string{"h2"}
		}
		transportSocket, err := util.CreateUpstreamTransportSocket(brc.Hostname, opt.RootCertsPath, opt.SslClientCertPath, alpnProtocols)
		if err != nil {
			return nil, fmt.Errorf("error marshaling tls context to transport_socket config for cluster %s, err=%v",
				brc.ClusterName, err)
		}
		c.TransportSocket = transportSocket
	}

	if isHttp2 {
		c.Http2ProtocolOptions = &corepb.Http2ProtocolOptions{}
	}

	switch opt.BackendDnsLookupFamily {
	case "auto":
		c.DnsLookupFamily = clusterpb.Cluster_AUTO
	case "v4only":
		c.DnsLookupFamily = clusterpb.Cluster_V4_ONLY
	case "v6only":
		c.DnsLookupFamily = clusterpb.Cluster_V6_ONLY
	default:
		return nil, fmt.Errorf("Invalid DnsLookupFamily: %s; Only auto, v4only or v6only are valid.", opt.BackendDnsLookupFamily)
	}
	return c, nil
}

func makeCatchAllBackendCluster(serviceInfo *sc.ServiceInfo) (*clusterpb.Cluster, error) {
	c, err := makeBackendCluster(&serviceInfo.Options, serviceInfo.CatchAllBackend)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func makeServiceControlCluster(serviceInfo *sc.ServiceInfo) (*clusterpb.Cluster, error) {
	uri := serviceInfo.ServiceConfig().GetControl().GetEnvironment()
	if uri == "" {
		return nil, nil
	}

	// The assumption about control.environment field. Its format:
	//   [scheme://] +  host + [:port]
	// * It should not have any path part
	// * If scheme is missed, https is the default

	scheme, hostname, port, path, err := util.ParseURI(uri)
	if err != nil {
		return nil, err
	}
	if path != "" {
		return nil, fmt.Errorf("Invalid uri: service control should not have path part: %s, %s", uri, path)
	}

	connectTimeoutProto := ptypes.DurationProto(5 * time.Second)
	serviceInfo.ServiceControlURI = scheme + "://" + hostname + "/v1/services"
	c := &clusterpb.Cluster{
		Name:                 util.ServiceControlClusterName,
		LbPolicy:             clusterpb.Cluster_ROUND_ROBIN,
		ConnectTimeout:       connectTimeoutProto,
		DnsLookupFamily:      clusterpb.Cluster_V4_ONLY,
		ClusterDiscoveryType: &clusterpb.Cluster_Type{clusterpb.Cluster_LOGICAL_DNS},
		LoadAssignment:       util.CreateLoadAssignment(hostname, port),
	}

	if scheme == "https" {
		transportSocket, err := util.CreateUpstreamTransportSocket(hostname, serviceInfo.Options.RootCertsPath, "", nil)
		if err != nil {
			return nil, fmt.Errorf("error marshaling tls context to transport_socket config for cluster %s, err=%v",
				c.Name, err)
		}
		c.TransportSocket = transportSocket
	}

	return c, nil
}

func makeBackendRoutingClusters(serviceInfo *sc.ServiceInfo) ([]*clusterpb.Cluster, error) {
	var brClusters []*clusterpb.Cluster

	for _, v := range serviceInfo.BackendRoutingClusters {
		c, err := makeBackendCluster(&serviceInfo.Options, v)
		if err != nil {
			return nil, err
		}

		brClusters = append(brClusters, c)

	}
	return brClusters, nil
}
