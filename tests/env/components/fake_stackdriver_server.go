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

package components

import (
	"context"
	"fmt"
	"net"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	cloudtracepb "google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
)

// FakeTraceServer implements all of the cloud trace v3 RPCs (see cloudtracepb.TraceServiceServer)
type FakeTraceServer struct {
	RcvSpan chan *cloudtracepb.Span
	server  *grpc.Server
}

func (s *FakeTraceServer) BatchWriteSpans(ctx context.Context, req *cloudtracepb.BatchWriteSpansRequest) (*empty.Empty, error) {
	for _, span := range req.Spans {
		s.RcvSpan <- span
	}
	return &empty.Empty{}, nil
}

func (s *FakeTraceServer) CreateSpan(ctx context.Context, span *cloudtracepb.Span) (*cloudtracepb.Span, error) {
	return span, nil
}

func (s *FakeTraceServer) StopAndWait() {
	glog.Infof("Stopping Stackdriver trace server")
	s.server.Stop()
}

func NewFakeStackdriver(port uint16) *FakeTraceServer {

	grpcServer := grpc.NewServer()
	fsds := &FakeTraceServer{
		RcvSpan: make(chan *cloudtracepb.Span, 10),
		server:  grpcServer,
	}
	cloudtracepb.RegisterTraceServiceServer(grpcServer, fsds)

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		glog.Infof("Stackdriver trace server listening on port %v\n", port)
		if err != nil {
			glog.Fatalf("failed to listen: %v", err)
		}
		err = grpcServer.Serve(lis)
		if err != nil {
			glog.Fatalf("fake stackdriver server terminated abnormally: %v", err)
		}
	}()

	return fsds
}
