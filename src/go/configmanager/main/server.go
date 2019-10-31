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

package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/GoogleCloudPlatform/api-proxy/src/go/configmanager"
	"github.com/GoogleCloudPlatform/api-proxy/src/go/configmanager/flags"
	"github.com/GoogleCloudPlatform/api-proxy/src/go/metadata"
	"github.com/golang/glog"
	"google.golang.org/grpc"

	v2grpc "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
)

var (
	DiscoveryPort = flag.Int("discovery_port", 8790, "Port that configmanager should serve ADS")
)

func main() {
	flag.Parse()
	opts := flags.EnvoyConfigOptionsFromFlags()

	var mf *metadata.MetadataFetcher
	if !opts.NonGCP {
		glog.Info("running on GCP, initializing metadata fetcher")
		mf = metadata.NewMetadataFetcher(opts.CommonOptions)
	}

	m, err := configmanager.NewConfigManager(mf, opts)
	if err != nil {
		glog.Exitf("fail to initialize config manager: %v", err)
	}
	server := xds.NewServer(m.Cache(), nil)
	grpcServer := grpc.NewServer()
	lis, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *DiscoveryPort))
	if err != nil {
		glog.Exitf("Server failed to listen: %v", err)
	}

	// Register Envoy discovery services.
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	v2grpc.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	v2grpc.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	v2grpc.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	v2grpc.RegisterListenerDiscoveryServiceServer(grpcServer, server)
	fmt.Printf("config manager server is running at %s .......\n", lis.Addr())

	// Handle signals gracefully
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signalChan
		glog.Warningf("Server got signal %v, stopping", sig)
		grpcServer.Stop()
	}()

	if err := grpcServer.Serve(lis); err != nil {
		glog.Exitf("Server fail to serve: %v", err)
	}
}
