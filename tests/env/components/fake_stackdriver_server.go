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
	"time"

	"github.com/golang/glog"
	"google.golang.org/grpc"

	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

// FakeTraceServer implements OTLP trace service (see coltracepb.TraceServiceServer)
type FakeTraceServer struct {
	coltracepb.TraceServiceServer

	RcvSpan chan *tracepb.Span
	server  *grpc.Server
}

func (s *FakeTraceServer) Export(ctx context.Context, req *coltracepb.ExportTraceServiceRequest) (*coltracepb.ExportTraceServiceResponse, error) {
	for _, rs := range req.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				glog.Infof("Fake stackdriver server received span with name: %v", span.Name)
				s.RcvSpan <- span
			}
		}
	}
	return &coltracepb.ExportTraceServiceResponse{}, nil
}

func (s *FakeTraceServer) StopAndWait() {
	glog.Infof("Stopping Stackdriver trace server")
	close(s.RcvSpan)
	s.server.Stop()
}

func NewFakeStackdriver() *FakeTraceServer {

	grpcServer := grpc.NewServer()
	fsds := &FakeTraceServer{
		RcvSpan: make(chan *tracepb.Span, 100),
		server:  grpcServer,
	}
	coltracepb.RegisterTraceServiceServer(grpcServer, fsds)

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

func (s *FakeTraceServer) RetrieveSpanNames() ([]string, error) {
	names := make([]string, 0)
	for true {

		select {
		case span := <-s.RcvSpan:
			names = append(names, span.Name)

		case <-time.After(1 * time.Second):
			// No more spans received by the server.
			glog.Infof("got spans: %+q", names)
			return names, nil
		}
	}

	return nil, fmt.Errorf("did not expect fake stackdriver server to close channel")
}

// When the test is over, there should be no more spans left.
func (s *FakeTraceServer) VerifyInvariants() error {
	glog.Infof("Verifying trace invariants")

	gotSpans, err := s.RetrieveSpanNames()
	if err != nil {
		return err
	}

	if len(gotSpans) != 0 {
		return fmt.Errorf("at the end of the test, there were (%v) spans unaccounted for", len(gotSpans))
	}

	return nil
}
