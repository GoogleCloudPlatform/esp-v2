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
	"context"
	"flag"
	"fmt"

	basepb "github.com/envoyproxy/data-plane-api/api/base"
	discoverypb "github.com/envoyproxy/data-plane-api/api/discovery"
	ldspb "github.com/envoyproxy/data-plane-api/api/lds"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/grpc"
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

	client := ldspb.NewListenerDiscoveryServiceClient(conn)
	ctx := context.Background()

	req := &discoverypb.DiscoveryRequest{
		Node: &basepb.Node{
			Id: "api_proxy",
		},
	}
	resp := &discoverypb.DiscoveryResponse{}
	if resp, err = client.FetchListeners(ctx, req); err != nil {
		glog.Exitf("discovery: %v", err)
	}

	fmt.Println("Version Info: ", resp.GetVersionInfo())
	fmt.Println("Type Url: ", resp.GetTypeUrl())

	for _, r := range resp.GetResources() {
		listener := &ldspb.Listener{}
		if err := proto.Unmarshal(r.GetValue(), listener); err != nil {
			glog.Exitf("Unmarshal got error: %v", err)
		}
		marshaler := &jsonpb.Marshaler{}
		var jsonStr string
		if jsonStr, err = marshaler.MarshalToString(listener); err != nil {
			glog.Exitf("fail to unmarshal listener: %v", err)
		}
		glog.Infof(jsonStr)
	}
	glog.Flush()
	// All fine.
	return
}
