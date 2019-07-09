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
	"strings"
	"time"

	"cloudesf.googlesource.com/gcpproxy/src/go/flags"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	sc "cloudesf.googlesource.com/gcpproxy/src/go/configinfo"
	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
	cdspb "github.com/envoyproxy/data-plane-api/api/cds"
	certpb "github.com/envoyproxy/data-plane-api/api/cert"
	protocolpb "github.com/envoyproxy/data-plane-api/api/protocol"
)

// MakeClusters provides dynamic cluster settings for Envoy
func MakeClusters(serviceInfo *sc.ServiceInfo) ([]*cdspb.Cluster, error) {
	var clusters []*cdspb.Cluster
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

func makeMetadataCluster(serviceInfo *sc.ServiceInfo) (*cdspb.Cluster, error) {
	scheme, hostname, port, _, err := ut.ParseURI(*flags.MetadataURL)
	if err != nil {
		return nil, err
	}

	connectTimeoutProto := ptypes.DurationProto(*flags.ClusterConnectTimeout)
	c := &cdspb.Cluster{
		Name:           ut.MetadataServerClusterName,
		LbPolicy:       cdspb.Cluster_ROUND_ROBIN,
		ConnectTimeout: connectTimeoutProto,
		ClusterDiscoveryType: &cdspb.Cluster_Type{
			Type: cdspb.Cluster_STRICT_DNS,
		},
		LoadAssignment: ut.CreateLoadAssignment(hostname, port),
	}

	if scheme == "https" {
		c.TlsContext = &certpb.UpstreamTlsContext{
			Sni: hostname,
		}
	}

	return c, nil
}

func makeJwtProviderClusters(serviceInfo *sc.ServiceInfo) ([]*cdspb.Cluster, error) {
	var providerClusters []*cdspb.Cluster
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

		connectTimeoutProto := ptypes.DurationProto(*flags.ClusterConnectTimeout)
		c := &cdspb.Cluster{
			Name:           provider.GetIssuer(),
			LbPolicy:       cdspb.Cluster_ROUND_ROBIN,
			ConnectTimeout: connectTimeoutProto,
			// Note: It may not be V4.
			DnsLookupFamily:      cdspb.Cluster_V4_ONLY,
			ClusterDiscoveryType: &cdspb.Cluster_Type{cdspb.Cluster_LOGICAL_DNS},
			LoadAssignment:       ut.CreateLoadAssignment(hostname, port),
		}
		if scheme == "https" {
			c.TlsContext = &certpb.UpstreamTlsContext{
				Sni: hostname,
			}
		}
		providerClusters = append(providerClusters, c)

		glog.Infof("Add provider cluster configuration for %v: %v", provider.JwksUri, c)
	}
	return providerClusters, nil
}

func makeBackendCluster(serviceInfo *sc.ServiceInfo) (*cdspb.Cluster, error) {
	connectTimeoutProto := ptypes.DurationProto(*flags.ClusterConnectTimeout)
	c := &cdspb.Cluster{
		Name:                 serviceInfo.ApiName,
		LbPolicy:             cdspb.Cluster_ROUND_ROBIN,
		ConnectTimeout:       connectTimeoutProto,
		ClusterDiscoveryType: &cdspb.Cluster_Type{cdspb.Cluster_STRICT_DNS},
		LoadAssignment:       ut.CreateLoadAssignment(*flags.ClusterAddress, uint32(*flags.ClusterPort)),
	}
	// gRPC and HTTP/2 need this configuration.
	if serviceInfo.BackendProtocol != ut.HTTP1 {
		c.Http2ProtocolOptions = &protocolpb.Http2ProtocolOptions{}
	}
	glog.Infof("Backend cluster configuration for service %s: %v", serviceInfo.Name, c)
	return c, nil
}

func makeServiceControlCluster(serviceInfo *sc.ServiceInfo) (*cdspb.Cluster, error) {
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

	connectTimeoutProto := ptypes.DurationProto(5 * time.Second)
	serviceInfo.ServiceControlURI = scheme + "://" + hostname + "/v1/services/"
	c := &cdspb.Cluster{
		Name:                 ut.ServiceControlClusterName,
		LbPolicy:             cdspb.Cluster_ROUND_ROBIN,
		ConnectTimeout:       connectTimeoutProto,
		DnsLookupFamily:      cdspb.Cluster_V4_ONLY,
		ClusterDiscoveryType: &cdspb.Cluster_Type{cdspb.Cluster_LOGICAL_DNS},
		LoadAssignment:       ut.CreateLoadAssignment(hostname, port),
	}

	if scheme == "https" {
		c.TlsContext = &certpb.UpstreamTlsContext{
			Sni: hostname,
		}
	}
	glog.Infof("adding cluster Configuration for uri: %s: %v", uri, c)
	return c, nil
}

func makeBackendRoutingClusters(serviceInfo *sc.ServiceInfo) ([]*cdspb.Cluster, error) {
	var brClusters []*cdspb.Cluster

	connectTimeoutProto := ptypes.DurationProto(*flags.ClusterConnectTimeout)
	for _, v := range serviceInfo.BackendRoutingClusters {
		c := &cdspb.Cluster{
			Name:                 v.ClusterName,
			LbPolicy:             cdspb.Cluster_ROUND_ROBIN,
			ConnectTimeout:       connectTimeoutProto,
			ClusterDiscoveryType: &cdspb.Cluster_Type{cdspb.Cluster_LOGICAL_DNS},
			LoadAssignment:       ut.CreateLoadAssignment(v.Hostname, v.Port),
			TlsContext: &certpb.UpstreamTlsContext{
				Sni: v.Hostname,
			},
		}

		if strings.HasSuffix(v.Hostname, ".run.app") {
			// The IPv6 DNS lookup Cloud Run does not resolve properly
			c.DnsLookupFamily = cdspb.Cluster_V4_ONLY
		}

		brClusters = append(brClusters, c)
		glog.Infof("Add backend routing cluster configuration for %v: %v", v.ClusterName, c)
	}
	return brClusters, nil
}
