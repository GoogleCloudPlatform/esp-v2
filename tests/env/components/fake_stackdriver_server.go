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
	"strings"
	"time"

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
	for i, span := range req.Spans {
		glog.Infof("Fake stackdriver server received span %v with name: %v", i, span.DisplayName.Value)
		s.RcvSpan <- span
	}
	return &emptypb.Empty{}, nil
}

func (s *FakeTraceServer) CreateSpan(ctx context.Context, span *cloudtracepb.Span) (*cloudtracepb.Span, error) {
	glog.Infof("Fake stackdriver server created span with name: %v", span.DisplayName.Value)
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
		RcvSpan: make(chan *cloudtracepb.Span, 20),
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

func (s *FakeTraceServer) RetrieveSpanNames() ([]string, error) {
	names := make([]string, 0)
	for true {

		select {
		case span := <-s.RcvSpan:

			// Check attributes
			if len(span.Attributes.AttributeMap) == 0 {
				return nil, fmt.Errorf("expected span %s to have more than 0 attributes attached to it", span.DisplayName.Value)
			}

			// Check for project id
			if !strings.Contains(span.Name, FakeProjectID) {
				return nil, fmt.Errorf("expected span %s to have the project id in its name, but got name: %s", span.DisplayName.Value, span.Name)
			}

			names = append(names, span.DisplayName.Value)

		case <-time.After(5 * time.Second):
			// No more spans received by the server.
			glog.Infof("got spans: %+q", names)
			return names, nil
		}
	}

	return nil, fmt.Errorf("did not expect fake stackdriver server to close channel")
}

func (s *FakeTraceServer) RetrieveSpanCount() (int, error) {
	names, err := s.RetrieveSpanNames()
	if err != nil {
		return 0, err
	}
	glog.Infof("got %v spans", len(names))
	return len(names), nil
}

// When the test is over, there should be no more spans left.
func (s *FakeTraceServer) VerifyInvariants() error {
	glog.Infof("Verifying trace invariants")

	gotSpansNum, err := s.RetrieveSpanCount()
	if err != nil {
		return err
	}

	if gotSpansNum != 0 {
		return fmt.Errorf("at the end of the test, there were (%v) spans unaccounted for", gotSpansNum)
	}

	return nil
}
