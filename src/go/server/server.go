// Copyright 2018 Google Cloud Platform Proxy Authors
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

package main

import (
	"flag"
	"fmt"
	"net"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
	"github.com/golang/glog"
	"google.golang.org/grpc"

	"cloudesf.googlesource.com/gcpproxy/src/go/configmanager"
)

var (
	serviceName   = flag.String("service_name", "", "endpoint service name")
	configID      = flag.String("config_id", "", "initial service config id")
	discoveryPort = flag.Int("discovery_port", 8790, "discovery service port")
)

func main() {
	flag.Parse()
	m, err := configmanager.NewConfigManager(*serviceName, *configID)
	if err != nil {
		glog.Exitf("fail to initialize config manager: %v", err)
	}
	server := xds.NewServer(m.Cache(), nil)
	grpcServer := grpc.NewServer()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *discoveryPort))
	if err != nil {
		glog.Exitf("Server failed to listen: %v", err)
	}

	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	api.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	api.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	api.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	api.RegisterListenerDiscoveryServiceServer(grpcServer, server)
	fmt.Println("config manager server is running .......")

	grpcServer.Serve(lis)
}
