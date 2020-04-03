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

package components

import (
	"context"
	"fmt"
	"net"

	"github.com/golang/glog"
	"google.golang.org/grpc"

	emptypb "github.com/golang/protobuf/ptypes/empty"
	cloudtracegrpc "google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
	cloudtracepb "google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
)

// FakeTraceServer implements the cloud trace v2 RPCs (see cloudtracegrpc.TraceServiceServer)
type FakeTraceServer struct {
	cloudtracegrpc.TraceServiceServer

	RcvSpan chan *cloudtracepb.Span
	server  *grpc.Server
}

func (s *FakeTraceServer) BatchWriteSpans(ctx context.Context, req *cloudtracepb.BatchWriteSpansRequest) (*emptypb.Empty, error) {
	for _, span := range req.Spans {
		s.RcvSpan <- span
	}
	return &emptypb.Empty{}, nil
}

func (s *FakeTraceServer) CreateSpan(ctx context.Context, span *cloudtracepb.Span) (*cloudtracepb.Span, error) {
	return span, nil
}

func (s *FakeTraceServer) StopAndWait() {
	glog.Infof("Stopping Stackdriver trace server")
	close(s.RcvSpan)
	s.server.Stop()
}

func NewFakeStackdriver() *FakeTraceServer {

	grpcServer := grpc.NewServer()
	fsds := &FakeTraceServer{
		RcvSpan: make(chan *cloudtracepb.Span, 10),
		server:  grpcServer,
	}
	cloudtracegrpc.RegisterTraceServiceServer(grpcServer, fsds)

	return fsds
}

func (s *FakeTraceServer) StartStackdriverServer(port uint16) {
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		glog.Infof("Stackdriver trace server listening on port %v\n", port)
		if err != nil {
			glog.Fatalf("failed to listen: %v", err)
		}
		err = s.server.Serve(lis)
		if err != nil {
			glog.Fatalf("fake stackdriver server terminated abnormally: %v", err)
		}
	}()
}
