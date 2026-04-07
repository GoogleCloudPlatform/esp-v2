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
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configmanager"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configmanager/flags"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/metadata"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/tokengenerator"
	"github.com/golang/glog"
	"google.golang.org/grpc"

	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	xds "github.com/envoyproxy/go-control-plane/pkg/server/v3"
)

func main() {
	flag.Parse()
	opts := flags.EnvoyConfigOptionsFromFlags()

	// Create context that allows cancellation.
	// Allows shutting down downstream servers gracefully.
	ctx, cancel := context.WithCancel(context.Background())

	var mf *metadata.MetadataFetcher
	if !opts.NonGCP {
		glog.Info("running on GCP, initializing metadata fetcher")
		mf = metadata.NewMetadataFetcher(opts.CommonOptions)
	}

	m, err := configmanager.NewConfigManager(mf, opts)
	if err != nil {
		glog.Exitf("fail to initialize config manager: %v", err)
	}
	server := xds.NewServer(ctx, m.Cache(), nil)
	grpcServer := grpc.NewServer()
	lis, err := net.Listen("unix", opts.AdsNamedPipe)
	if err != nil {
		glog.Exitf("Server failed to listen: %v", err)
	}

	// Register Envoy discovery services.
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)

	glog.Infof("config manager server is running at %s .......\n", lis.Addr())

	// Handle signals gracefully
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signalChan
		glog.Warningf("Server got signal %v, stopping", sig)
		cancel()
		grpcServer.Stop()
	}()

	if opts.ServiceAccountKey != "" {
		// Setup token agent server
		r := tokengenerator.MakeTokenAgentHandler(opts.ServiceAccountKey)
		go func() {
			err := http.ListenAndServe(fmt.Sprintf(":%v", opts.TokenAgentPort), r)

			if err != nil {
				glog.Errorf("token agent fail to serve: %v", err)
			}

		}()

	}

	if err := grpcServer.Serve(lis); err != nil {
		glog.Exitf("Server fail to serve: %v", err)
	}
}
