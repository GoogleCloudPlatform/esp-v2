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

	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/grpc"

	v2pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
)

var (
	port = flag.Int("port", 8790, "LDS port")
)

func main() {
	flag.Parse()
	var opts []grpc.DialOption

	opts = append(opts, grpc.WithInsecure())
	addr := fmt.Sprintf("localhost:%d", *port)
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		glog.Exitf("failed to connect to server: %v", err)
	}

	client := discoverygrpc.NewAggregatedDiscoveryServiceClient(conn)
	ctx := context.Background()
	stream, err := client.StreamAggregatedResources(ctx)
	if err != nil {
		glog.Exitf("StreamAggregatedResources: %v", err)
	}

	req := &v2pb.DiscoveryRequest{
		TypeUrl: "type.googleapis.com/envoy.api.v2.Listener",
		Node: &corepb.Node{
			Id: "api_proxy",
		},
	}
	if err := stream.Send(req); err != nil {
		glog.Exitf("SendMsg: %v", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		glog.Exitf("Failed to call Config Manager, got error:\n%s", resp)
	}


	marshaler := &jsonpb.Marshaler{}
	var jsonStr string
	if jsonStr, err = marshaler.MarshalToString(resp); err != nil {
		glog.Exitf("fail to unmarshal listener: %v", err)
	}
	glog.Infof("Received response from Config Manager:\n%s", jsonStr)
	// All fine.
	return
}
