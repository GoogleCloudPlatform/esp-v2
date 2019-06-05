// Copyright 2019 Google Cloud Platform Proxy Authors
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

	"cloudesf.googlesource.com/gcpproxy/src/go/flags"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/golang/glog"

	sc "cloudesf.googlesource.com/gcpproxy/src/go/configinfo"
	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

const (
	serviceControlClusterName = "service-control-cluster"
	metadataServerClusterName = "metadata-cluster"
)

// MakeClusters provides dynamic cluster settings for Envoy
func MakeClusters(serviceInfo *sc.ServiceInfo) ([]cache.Resource, error) {
	var clusters []cache.Resource
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

func makeMetadataCluster(serviceInfo *sc.ServiceInfo) (*v2.Cluster, error) {
	scheme, hostname, port, _, err := ut.ParseURI(*flags.MetadataURL)
	if err != nil {
		return nil, err
	}

	c := &v2.Cluster{
		Name:           metadataServerClusterName,
		LbPolicy:       v2.Cluster_ROUND_ROBIN,
		ConnectTimeout: *flags.ClusterConnectTimeout,
		ClusterDiscoveryType: &v2.Cluster_Type{
			Type: v2.Cluster_STRICT_DNS,
		},
		LoadAssignment: createLoadAssignment(hostname, port),
	}

	if scheme == "https" {
		c.TlsContext = &auth.UpstreamTlsContext{
			Sni: hostname,
		}
	}

	return c, nil
}

func createLoadAssignment(hostname string, port uint32) *v2.ClusterLoadAssignment {
	return &v2.ClusterLoadAssignment{
		ClusterName: hostname,
		Endpoints: []endpoint.LocalityLbEndpoints{{
			LbEndpoints: []endpoint.LbEndpoint{{
				HostIdentifier: &endpoint.LbEndpoint_Endpoint{
					Endpoint: &endpoint.Endpoint{
						Address: &core.Address{
							Address: &core.Address_SocketAddress{
								SocketAddress: &core.SocketAddress{
									Address: hostname,
									PortSpecifier: &core.SocketAddress_PortValue{
										PortValue: port,
									},
								},
							},
						},
					},
				},
			},
			}},
		},
	}
}

func makeJwtProviderClusters(serviceInfo *sc.ServiceInfo) ([]cache.Resource, error) {
	var providerClusters []cache.Resource
	authn := serviceInfo.ServiceConfig().GetAuthentication()
	for _, provider := range authn.GetProviders() {
		jwksUri := provider.GetJwksUri()

		// Note: When jwksUri is empty, proxy will try to find jwksUri by openID
		// discovery. If error happens during this process, a fake and unaccessible
		// jwksUri will be filled instead.
		if jwksUri == "" {
			jwksUriByOpenID, err := ut.ResolveJwksUriUsingOpenID(provider.GetIssuer())
			if err != nil {
				glog.Warning(err.Error())
				jwksUri = ut.FakeJwksUri
			} else {
				jwksUri = jwksUriByOpenID
			}
			provider.JwksUri = jwksUri
		}

		scheme, hostname, port, _, err := ut.ParseURI(jwksUri)
		if err != nil {
			glog.Warningf("Fail to parse jwksUri %s with error %v", jwksUri, err)
			scheme, hostname, port, _, _ = ut.ParseURI(ut.FakeJwksUri)
			provider.JwksUri = ut.FakeJwksUri
		}

		c := &v2.Cluster{
			Name:           provider.GetIssuer(),
			LbPolicy:       v2.Cluster_ROUND_ROBIN,
			ConnectTimeout: *flags.ClusterConnectTimeout,
			// Note: It may not be V4.
			DnsLookupFamily:      v2.Cluster_V4_ONLY,
			ClusterDiscoveryType: &v2.Cluster_Type{v2.Cluster_LOGICAL_DNS},
			LoadAssignment:       createLoadAssignment(hostname, port),
		}
		providerClusters = append(providerClusters, c)

		if scheme == "https" {
			c.TlsContext = &auth.UpstreamTlsContext{
				Sni: hostname,
			}
		}
		glog.Infof("Add provider cluster configuration for %v: %v", provider.JwksUri, c)
	}
	return providerClusters, nil
}

func makeBackendCluster(serviceInfo *sc.ServiceInfo) (*v2.Cluster, error) {
	c := &v2.Cluster{
		Name:                 serviceInfo.ApiName,
		LbPolicy:             v2.Cluster_ROUND_ROBIN,
		ConnectTimeout:       *flags.ClusterConnectTimeout,
		ClusterDiscoveryType: &v2.Cluster_Type{v2.Cluster_STRICT_DNS},
		LoadAssignment:       createLoadAssignment(*flags.ClusterAddress, uint32(*flags.ClusterPort)),
	}
	// gRPC and HTTP/2 need this configuration.
	if serviceInfo.BackendProtocol != ut.HTTP1 {
		c.Http2ProtocolOptions = &core.Http2ProtocolOptions{}
	}
	glog.Infof("Backend cluster configuration for service %s: %v", serviceInfo.Name, c)
	return c, nil
}

func makeServiceControlCluster(serviceInfo *sc.ServiceInfo) (*v2.Cluster, error) {
	uri := serviceInfo.ServiceConfig().GetControl().GetEnvironment()
	if uri == "" {
		return nil, nil
	}

	// The assumption about control.environment field. Its format:
	//   [scheme://] +  host + [:port]
	// * It should not have any path part
	// * If scheme is missed, https is the default

	scheme, hostname, port, path, err := ut.ParseURI(uri)
	if err != nil {
		return nil, err
	}
	if path != "" {
		return nil, fmt.Errorf("Invalid uri: service control should not have path part: %s, %s", uri, path)
	}

	serviceInfo.ServiceControlURI = scheme + "://" + hostname + "/v1/services/"
	c := &v2.Cluster{
		Name:                 serviceControlClusterName,
		LbPolicy:             v2.Cluster_ROUND_ROBIN,
		ConnectTimeout:       5 * time.Second,
		DnsLookupFamily:      v2.Cluster_V4_ONLY,
		ClusterDiscoveryType: &v2.Cluster_Type{v2.Cluster_LOGICAL_DNS},
		LoadAssignment:       createLoadAssignment(hostname, port),
	}

	if scheme == "https" {
		c.TlsContext = &auth.UpstreamTlsContext{
			Sni: hostname,
		}
	}
	glog.Infof("adding cluster Configuration for uri: %s: %v", uri, c)
	return c, nil
}

func makeBackendRoutingClusters(serviceInfo *sc.ServiceInfo) ([]cache.Resource, error) {
	var brClusters []cache.Resource
	for _, v := range serviceInfo.BackendRoutingClusters {
		c := &v2.Cluster{
			Name:                 v.ClusterName,
			LbPolicy:             v2.Cluster_ROUND_ROBIN,
			ConnectTimeout:       *flags.ClusterConnectTimeout,
			ClusterDiscoveryType: &v2.Cluster_Type{v2.Cluster_LOGICAL_DNS},
			LoadAssignment:       createLoadAssignment(v.Hostname, v.Port),
			TlsContext: &auth.UpstreamTlsContext{
				Sni: v.Hostname,
			},
		}
		brClusters = append(brClusters, c)
		glog.Infof("Add backend routing cluster configuration for %v: %v", v.ClusterName, c)
	}
	return brClusters, nil
}
