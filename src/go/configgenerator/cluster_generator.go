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

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	sc "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	v2pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
)

// MakeClusters provides dynamic cluster settings for Envoy
// This must be called before MakeListeners.
func MakeClusters(serviceInfo *sc.ServiceInfo) ([]*v2pb.Cluster, error) {
	var clusters []*v2pb.Cluster
	backendCluster, err := makeBackendCluster(serviceInfo)
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
	return clusters, nil
}

func makeMetadataCluster(serviceInfo *sc.ServiceInfo) (*v2pb.Cluster, error) {
	scheme, hostname, port, _, err := util.ParseURI(serviceInfo.Options.MetadataURL)
	if err != nil {
		return nil, err
	}

	connectTimeoutProto := ptypes.DurationProto(serviceInfo.Options.ClusterConnectTimeout)
	c := &v2pb.Cluster{
		Name:           util.MetadataServerClusterName,
		LbPolicy:       v2pb.Cluster_ROUND_ROBIN,
		ConnectTimeout: connectTimeoutProto,
		ClusterDiscoveryType: &v2pb.Cluster_Type{
			Type: v2pb.Cluster_STRICT_DNS,
		},
		LoadAssignment: util.CreateLoadAssignment(hostname, port),
	}

	if scheme == "https" {
		transportSocket, err := util.CreateTransportSocket(hostname, serviceInfo.Options.RootCertsPath, nil)
		if err != nil {
			return nil, fmt.Errorf("error marshaling tls context to transport_socket config for cluster %s, err=%v",
				c.Name, err)
		}
		c.TransportSocket = transportSocket
	}

	return c, nil
}

func makeIamCluster(serviceInfo *sc.ServiceInfo) (*v2pb.Cluster, error) {
	if serviceInfo.Options.ServiceControlCredentials == nil && serviceInfo.Options.BackendAuthCredentials == nil {
		return nil, nil
	}
	scheme, hostname, port, _, err := util.ParseURI(serviceInfo.Options.IamURL)
	if err != nil {
		return nil, err
	}

	connectTimeoutProto := ptypes.DurationProto(serviceInfo.Options.ClusterConnectTimeout)
	c := &v2pb.Cluster{
		Name:            util.IamServerClusterName,
		LbPolicy:        v2pb.Cluster_ROUND_ROBIN,
		DnsLookupFamily: v2pb.Cluster_V4_ONLY,
		ConnectTimeout:  connectTimeoutProto,
		ClusterDiscoveryType: &v2pb.Cluster_Type{
			Type: v2pb.Cluster_STRICT_DNS,
		},
		LoadAssignment: util.CreateLoadAssignment(hostname, port),
	}

	if scheme == "https" {
		transportSocket, err := util.CreateTransportSocket(hostname, serviceInfo.Options.RootCertsPath, nil)
		if err != nil {
			return nil, fmt.Errorf("error marshaling tls context to transport_socket config for cluster %s, err=%v",
				c.Name, err)
		}
		c.TransportSocket = transportSocket
	}

	return c, nil
}

func makeJwtProviderClusters(serviceInfo *sc.ServiceInfo) ([]*v2pb.Cluster, error) {
	var providerClusters []*v2pb.Cluster
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

		c := &v2pb.Cluster{
			Name:           clusterName,
			LbPolicy:       v2pb.Cluster_ROUND_ROBIN,
			ConnectTimeout: connectTimeoutProto,
			// Note: It may not be V4.
			DnsLookupFamily:      v2pb.Cluster_V4_ONLY,
			ClusterDiscoveryType: &v2pb.Cluster_Type{v2pb.Cluster_LOGICAL_DNS},
			LoadAssignment:       util.CreateLoadAssignment(hostname, port),
		}
		if scheme == "https" {
			transportSocket, err := util.CreateTransportSocket(hostname, serviceInfo.Options.RootCertsPath, nil)
			if err != nil {
				return nil, fmt.Errorf("error marshaling tls context to transport_socket config for cluster %s, err=%v",
					c.Name, err)
			}
			c.TransportSocket = transportSocket
		}

		providerClusters = append(providerClusters, c)

		glog.Infof("Add provider cluster configuration for %v: %v", provider.JwksUri, c)
	}
	return providerClusters, nil
}

func makeBackendCluster(serviceInfo *sc.ServiceInfo) (*v2pb.Cluster, error) {
	connectTimeoutProto := ptypes.DurationProto(serviceInfo.Options.ClusterConnectTimeout)
	c := &v2pb.Cluster{
		Name:                 serviceInfo.BackendClusterName(),
		LbPolicy:             v2pb.Cluster_ROUND_ROBIN,
		ConnectTimeout:       connectTimeoutProto,
		ClusterDiscoveryType: &v2pb.Cluster_Type{v2pb.Cluster_STRICT_DNS},
		LoadAssignment:       util.CreateLoadAssignment(serviceInfo.Options.ClusterAddress, uint32(serviceInfo.Options.ClusterPort)),
	}
	// gRPC and HTTP/2 need this configuration.
	if serviceInfo.BackendProtocol != util.HTTP1 {
		c.Http2ProtocolOptions = &corepb.Http2ProtocolOptions{}
	}
	glog.Infof("Backend cluster configuration for service %s: %v", serviceInfo.Name, c)
	return c, nil
}

func makeServiceControlCluster(serviceInfo *sc.ServiceInfo) (*v2pb.Cluster, error) {
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
	serviceInfo.ServiceControlURI = scheme + "://" + hostname + "/v1/services/"
	c := &v2pb.Cluster{
		Name:                 util.ServiceControlClusterName,
		LbPolicy:             v2pb.Cluster_ROUND_ROBIN,
		ConnectTimeout:       connectTimeoutProto,
		DnsLookupFamily:      v2pb.Cluster_V4_ONLY,
		ClusterDiscoveryType: &v2pb.Cluster_Type{v2pb.Cluster_LOGICAL_DNS},
		LoadAssignment:       util.CreateLoadAssignment(hostname, port),
	}

	if scheme == "https" {
		transportSocket, err := util.CreateTransportSocket(hostname, serviceInfo.Options.RootCertsPath, nil)
		if err != nil {
			return nil, fmt.Errorf("error marshaling tls context to transport_socket config for cluster %s, err=%v",
				c.Name, err)
		}
		c.TransportSocket = transportSocket
	}
	glog.Infof("adding cluster Configuration for uri: %s: %v", uri, c)
	return c, nil
}

func makeBackendRoutingClusters(serviceInfo *sc.ServiceInfo) ([]*v2pb.Cluster, error) {
	var brClusters []*v2pb.Cluster

	connectTimeoutProto := ptypes.DurationProto(serviceInfo.Options.ClusterConnectTimeout)
	for _, v := range serviceInfo.BackendRoutingClusters {
		isHttp2 := serviceInfo.BackendProtocol != util.HTTP1

		var alpnProtocols []string
		if isHttp2 {
			alpnProtocols = []string{"h2"}
		}
		transportSocket, err := util.CreateTransportSocket(v.Hostname, serviceInfo.Options.RootCertsPath, alpnProtocols)
		if err != nil {
			return nil, fmt.Errorf("error marshaling tls context to transport_socket config for cluster %s, err=%v",
				v.ClusterName, err)
		}

		c := &v2pb.Cluster{
			Name:                 v.ClusterName,
			LbPolicy:             v2pb.Cluster_ROUND_ROBIN,
			ConnectTimeout:       connectTimeoutProto,
			ClusterDiscoveryType: &v2pb.Cluster_Type{v2pb.Cluster_LOGICAL_DNS},
			LoadAssignment:       util.CreateLoadAssignment(v.Hostname, v.Port),
			TransportSocket:      transportSocket,
		}

		if isHttp2 {
			c.Http2ProtocolOptions = &corepb.Http2ProtocolOptions{}
		}

		switch serviceInfo.Options.BackendDnsLookupFamily {
		case "auto":
			c.DnsLookupFamily = v2pb.Cluster_AUTO
		case "v4only":
			c.DnsLookupFamily = v2pb.Cluster_V4_ONLY
		case "v6only":
			c.DnsLookupFamily = v2pb.Cluster_V6_ONLY
		default:
			return nil, fmt.Errorf("Invalid DnsLookupFamily: %s; Only auto, v4only or v6only are valid.", serviceInfo.Options.BackendDnsLookupFamily)
		}

		brClusters = append(brClusters, c)
		glog.Infof("Add backend routing cluster configuration for %v: %v", v.ClusterName, c)
	}
	return brClusters, nil
}
