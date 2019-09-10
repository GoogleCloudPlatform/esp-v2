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
	"sync"

	"github.com/golang/glog"
)

// All registered HealthCheckers must implement this interface
type HealthChecker interface {
	fmt.Stringer
	CheckHealth() error
}

// Registry of Health Checkers
type HealthRegistry struct {
	checkers map[string]HealthChecker
	errors   chan error
	wg       sync.WaitGroup
}

func NewHealthRegistry() *HealthRegistry {
	return &HealthRegistry{
		checkers: make(map[string]HealthChecker),
	}
}

func (hr *HealthRegistry) RegisterHealthChecker(checker HealthChecker) {
	hr.checkers[checker.String()] = checker
}

func (hr *HealthRegistry) DeregisterHealthChecker(checker HealthChecker) {
	delete(hr.checkers, checker.String())
}

// Runs all registered health checks in parallel. Will return error if any health check fails.
func (hr *HealthRegistry) RunAllHealthChecks() error {

	// Reset internal variables (allows this function to be called multiple times)
	hr.errors = make(chan error, len(hr.checkers))

	for _, checker := range hr.checkers {

		// Run this health checker in parallel
		glog.Infof("Running health check in background: %v", checker)
		hr.wg.Add(1)

		// HealthChecker is an explicit argument to prevent shared memory across goroutines
		// https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		go func(hc HealthChecker) {

			defer hr.wg.Done()

			if err := hc.CheckHealth(); err != nil {
				hr.errors <- fmt.Errorf("health check %v failed with err: %v", hc, err)
			}

		}(checker)

	}

	// Wait for all health checks to finish and close the channel
	hr.wg.Wait()
	close(hr.errors)

	// Check for any errors. Print all of them and return a generic error message if any of them failed
	var lastErr error
	for err := range hr.errors {
		glog.Error(err)
		lastErr = err
	}

	if lastErr != nil {
		return fmt.Errorf("1 or more health checks failed, view test logs for all failures. Last known failure: %v", lastErr)
	}

	glog.Infof("All health checks passed")
	return nil
}
