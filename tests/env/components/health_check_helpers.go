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

package components

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"

	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	// Max number of retries for a health check.
	healthCheckRetries = 10

	// Time to wait between each linear retry.
	healthCheckRetryBackoff = 1 * time.Second

	// Timeout for a request, in case server does not respond.
	healthCheckDeadline = time.Duration(1 * time.Second)
)

// Options to configure retries, timeouts, etc. for the retry helpers.
type HealthCheckOptions struct {
	HealthCheckRetries      int
	HealthCheckRetryBackoff time.Duration
	HealthCheckDeadline     time.Duration
}

// Constructs a HealthCheckOptions with the default values.
func NewHealthCheckOptions() *HealthCheckOptions {
	return &HealthCheckOptions{
		HealthCheckRetries:      healthCheckRetries,
		HealthCheckRetryBackoff: healthCheckRetryBackoff,
		HealthCheckDeadline:     healthCheckDeadline,
	}
}

// Very simple function to retry the given function using linear backoff.
// Retries until the retryFn returns a nil error, or until the max attempts are reached.
// Will wait the specified duration between retries (linear). Does not account for jitter.
// Returns the error that the last retry returns (or nil if successful).
func withRetry(attempts int, waitTime time.Duration, retryFn func() error) error {
	var err error

	for i := 0; i < attempts; i++ {

		// Return success if function does not error
		err = retryFn()
		if err == nil {
			return nil
		}

		// Had error, sleep
		glog.Infof("Health check got error %v, sleeping %v before retrying", err, waitTime.String())
		time.Sleep(waitTime)
	}

	// Give up on retries, return error
	return err
}

// BasicGrpcConnectionCheck performs a basic connectivity check to the underlying server using the standard gRPC connectivity semantics.
// https://github.com/grpc/grpc/blob/master/doc/connectivity-semantics-and-api.md
// This can be used if the server does not support the gRPC health checking protocol, but is generally less accurate.
func GrpcConnectionCheck(addr string, opts *HealthCheckOptions) error {
	// Connect to gRPC backend
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	err = withRetry(opts.HealthCheckRetries, opts.HealthCheckRetryBackoff, func() error {
		// Ensure we have connected to the gRPC server
		state := conn.GetState()
		if state != connectivity.Ready {
			return fmt.Errorf("health check response was not healthy: %v", state)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// GrpcHealthCheck checks the service running on the specified port using the standard gRPC health checking protocol.
// https://github.com/grpc/grpc/blob/master/doc/health-checking.md
// Does not support authentication. Supports retries and timeouts.
// Recommended to use over BasicGrpcConnectionCheck is the underlying server supports the protocol.
func GrpcHealthCheck(addr string, opts *HealthCheckOptions) error {
	// Connect to gRPC backend
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	// Create client
	client := healthpb.NewHealthClient(conn)

	// Create the health check request
	req := &healthpb.HealthCheckRequest{
		Service: "", // Default convention: Empty string represents overall server status
	}

	err = withRetry(opts.HealthCheckRetries, opts.HealthCheckRetryBackoff, func() error {
		// Create the deadline
		ctx, cancel := context.WithTimeout(context.Background(), opts.HealthCheckDeadline)
		defer cancel()

		// Make the actual request
		resp, err := client.Check(ctx, req)
		if err != nil {
			return err
		}

		// Ensure service is healthy
		if resp.Status != healthpb.HealthCheckResponse_SERVING {
			return fmt.Errorf("health check response was not healthy: %v", resp.Status)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// HttpHealthCheck checks the service running at the specified port and path for a 200 OK response.
// Does not support authentication. Supports retries and timeouts.
func HttpHealthCheck(addr string, endpoint string, opts *HealthCheckOptions) error {

	// Server address
	u, err := url.Parse(addr)
	if err != nil {
		return err
	}

	// Health check path
	u.Path = endpoint

	err = withRetry(opts.HealthCheckRetries, opts.HealthCheckRetryBackoff, func() error {
		// Create a client with an explicit deadline
		client := http.Client{
			Timeout: opts.HealthCheckDeadline,
		}

		// Make the request
		resp, err := client.Get(u.String())
		if err != nil {
			return err
		}

		// Ensure service is healthy
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("health check response was not healthy: %v", resp.StatusCode)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
