// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package health check provides a library to run a new gRPC server that implements the gRPC Health Check Protocol.
// This is intended to be used by our tests.
package healthcheckendpoint

import (
	"fmt"
	"net"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"

	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// Server represents the gRPC health checkable server.
// The address the server is listening on can be extracted from this struct.
type Server struct {
	Lis          net.Listener
	grpcServer   *grpc.Server
	healthServer *health.Server
}

// NewServer creates a new server but does not start it.
// This sets up the listening address and the initial health.
func NewServer() (*Server, error) {

	// Setup health server
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)

	// Setup gRPC server
	grpcServer := grpc.NewServer()

	// Create a new listener, allowing it to choose the port
	lis, err := net.Listen(platform.GetNetworkProtocol(), fmt.Sprintf("%v:", platform.GetLoopbackAddress()))
	if err != nil {
		return nil, fmt.Errorf("server failed to listen: %v", err)
	}

	// Register gRPC health services
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	// Return server
	return &Server{
		Lis:          lis,
		grpcServer:   grpcServer,
		healthServer: healthServer,
	}, nil
}

// StartServer starts the gRPC server with two durations.
// StartTime is the amount of time it takes to actually start serving the endpoints.
// HealthyTime is the amount of time it takes for the service to be considered healthy.
//
// Example: startTime = 2 sec, healthyTime = 3 sec
//
//	Tick 0: Server not running
//	Tick 1: Server not running
//	Tick 2: Server running but unhealthy
//	Tick 3: Server running and healthy
func (s *Server) StartServer(startTime time.Duration, healthyTime time.Duration) {

	// Start a timer in the background to set the healthy status
	timer := time.NewTimer(healthyTime)
	go func() {
		<-timer.C
		s.healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	}()

	// Wait before starting the server
	time.Sleep(startTime)

	// Start server
	fmt.Printf("Health check server is running at %s .......\n", s.Lis.Addr())
	if err := s.grpcServer.Serve(s.Lis); err != nil {
		glog.Errorf("server fail to serve: %v", err)
	}
}

func (s *Server) StopServer() {
	s.grpcServer.Stop()
}
