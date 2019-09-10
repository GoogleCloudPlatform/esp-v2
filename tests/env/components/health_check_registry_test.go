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
	"fmt"
	"testing"

	"github.com/golang/glog"
)

type FakeHealthChecker struct {
	name string
	resp error
}

func (fhc FakeHealthChecker) String() string {
	return fhc.name
}

func (fhc FakeHealthChecker) CheckHealth() error {
	return fhc.resp
}

func TestHealthRegistry(t *testing.T) {

	testData := []struct {
		desc      string
		responses []error
		wantErr   bool
	}{
		{
			desc: "All 2 health checks pass, no error returned",
			responses: []error{
				nil,
				nil,
			},
			wantErr: false,
		},
		{
			desc: "Only 1 health check passes, 1 error returned",
			responses: []error{
				nil,
				fmt.Errorf("fake error"),
			},
			wantErr: true,
		},
		{
			desc: "No health checks pass, 1 error returned",
			responses: []error{
				fmt.Errorf("fake error 1"),
				fmt.Errorf("fake error 2"),
			},
			wantErr: true,
		},
	}

	for i, tc := range testData {
		glog.Infof("Running test (%s)", tc.desc)

		// Setup the registry
		hr := NewHealthRegistry()

		// Register fake health checkers
		for _, resp := range tc.responses {
			hc := &FakeHealthChecker{
				name: fmt.Sprintf("Fake Health Checker %v", i),
				resp: resp,
			}
			hr.RegisterHealthChecker(hc)
		}

		// Run the health checks
		err := hr.RunAllHealthChecks()

		// Check for errors
		if tc.wantErr && err == nil {
			t.Errorf("Test (%s): failed, expect error, but got none", tc.desc)
		} else if !tc.wantErr && err != nil {
			t.Errorf("Test (%s): failed, expect success, but got err: %v", tc.desc, err)
		}

	}

}
