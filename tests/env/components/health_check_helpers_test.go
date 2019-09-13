// Copyright 2019 Google Cloud Platform Proxy Authors
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
	"testing"
	"time"

	"github.com/golang/glog"

	endpoint "cloudesf.googlesource.com/gcpproxy/tests/endpoints/health_check"
)

func TestGrpcHealthCheck(t *testing.T) {
	testData := []struct {
		desc        string
		startTime   time.Duration
		healthyTime time.Duration
		wantErr     bool
		startServer bool
	}{
		{
			desc:        "server is healthy on startup, health checks pass",
			healthyTime: 0 * time.Second,
		},
		{
			desc:      "server takes time to start, health checks pass eventually",
			startTime: 2 * time.Second, // Health checks will still be retrying after this duration
		},
		{
			desc:        "server is running but takes time to be healthy, health checks pass eventually",
			healthyTime: 2 * time.Second, // Health checks will still be retrying after this duration
		},
		{
			desc:      "server never starts, health checks fail",
			startTime: 10 * time.Second, // Health checks do not retry for this long
			wantErr:   true,
		},
		{
			desc:        "server is running but unhealthy, health checks fail",
			healthyTime: 10 * time.Second, // Health checks do not retry for this long
			wantErr:     true,
		},
	}

	for _, tc := range testData {
		glog.Infof("Running test (%s)", tc.desc)

		// Setup the server
		healthServer, err := endpoint.NewServer()
		if err != nil {
			t.Errorf("Test (%s): failed, on setup got err: %v", tc.desc, err)
			continue
		}

		func() {
			// Start the server and defer stopping till end of func
			defer healthServer.StopServer()
			go healthServer.StartServer(tc.startTime, tc.healthyTime)

			// Extract the address the server is listening on
			addr := healthServer.Lis.Addr()

			// Do the health check
			opts := NewHealthCheckOptions()
			err = GrpcHealthCheck(addr.String(), opts)

			// Check for errors
			if tc.wantErr && err == nil {
				t.Errorf("Test (%s): failed, expect error, but got none", tc.desc)
			} else if !tc.wantErr && err != nil {
				t.Errorf("Test (%s): failed, expect success, but got err: %v", tc.desc, err)
			}
		}()
	}
}
