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
	"strconv"
	"strings"
	"time"

	"cloudesf.googlesource.com/gcpproxy/src/go/flags"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/golang/glog"

	sc "cloudesf.googlesource.com/gcpproxy/src/go/configinfo"
	ut "cloudesf.googlesource.com/gcpproxy/src/go/util"
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

const (
	serviceControlClusterName = "service-control-cluster"
)

func MakeClusters(serviceInfo *sc.ServiceInfo) ([]cache.Resource, error) {
	var clusters []cache.Resource
	backendCluster, err := makeBackendCluster(serviceInfo)
	if err != nil {
		return nil, err
	}
	if backendCluster != nil {
		clusters = append(clusters, backendCluster)
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
	return clusters, nil
}

func makeBackendCluster(serviceInfo *sc.ServiceInfo) (*v2.Cluster, error) {
	c := &v2.Cluster{
		Name:                 serviceInfo.ApiName,
		LbPolicy:             v2.Cluster_ROUND_ROBIN,
		ConnectTimeout:       *flags.ClusterConnectTimeout,
		ClusterDiscoveryType: &v2.Cluster_Type{v2.Cluster_STRICT_DNS},
		Hosts: []*core.Address{
			{Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address: *flags.ClusterAddress,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: uint32(*flags.ClusterPort),
					},
				},
			},
			},
		},
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

	// Default is https
	scheme := "https"
	host := uri
	arr := strings.Split(uri, "://")
	if len(arr) == 2 {
		scheme = arr[0]
		host = arr[1]
	}

	// This is used in service_control_uri.uri in service control fitler.
	// Not path part, append /v1/services/ directly on host
	serviceInfo.ServiceControlURI = scheme + "://" + host + "/v1/services/"

	arr = strings.Split(host, ":")
	var port int
	if len(arr) == 2 {
		var err error
		port, err = strconv.Atoi(arr[1])
		if err != nil {
			return nil, fmt.Errorf("Invalid port: %s, got err: %s", arr[1], err)
		}
		host = arr[0]
	} else {
		if scheme == "http" {
			port = 80
		} else {
			port = 443
		}
	}

	c := &v2.Cluster{
		Name:                 serviceControlClusterName,
		LbPolicy:             v2.Cluster_ROUND_ROBIN,
		ConnectTimeout:       5 * time.Second,
		DnsLookupFamily:      v2.Cluster_V4_ONLY,
		ClusterDiscoveryType: &v2.Cluster_Type{v2.Cluster_LOGICAL_DNS},
		Hosts: []*core.Address{
			{Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address: host,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: uint32(port),
					},
				},
			},
			},
		},
	}

	if scheme == "https" {
		c.TlsContext = &auth.UpstreamTlsContext{
			Sni: host,
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
			Hosts: []*core.Address{
				{Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Address: v.Hostname,
						PortSpecifier: &core.SocketAddress_PortValue{
							PortValue: v.Port,
						},
					},
				},
				},
			},
			TlsContext: &auth.UpstreamTlsContext{
				Sni: v.Hostname,
			},
		}
		brClusters = append(brClusters, c)
		glog.Infof("Add backend routing cluster configuration for %v: %v", v.ClusterName, c)
	}
	return brClusters, nil
}
