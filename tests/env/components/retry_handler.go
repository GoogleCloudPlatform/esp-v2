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
	"net/http"
	"sync"
	"time"

	"github.com/golang/glog"
)

// Handles responding with errors for a specified number of retries.
// Has the ability to debounce multiple requests made within a time interval.
type RetryHandler struct {
	numWantFails int
	curNumFails  int
	lastFailTime time.Time
	mtx          sync.Mutex
}

func NewRetryHandler(wantNumFails int) *RetryHandler {
	return &RetryHandler{
		numWantFails: wantNumFails,
	}
}

// Returns true if this request is handled by a retry.
func (rh *RetryHandler) handleRetry(w http.ResponseWriter) bool {
	rh.mtx.Lock()

	if rh.curNumFails < rh.numWantFails {
		// Fail the first `n` times.
		w.WriteHeader(http.StatusForbidden)

		// Debounce multiple calls at the same time.
		diff := time.Now().Sub(rh.lastFailTime)
		if diff.Seconds() >= 2 {
			glog.Infof("Incrementing RetryHandler failures from %v", rh.curNumFails)
			rh.curNumFails++
			rh.lastFailTime = time.Now()
		}

		rh.mtx.Unlock()
		return true
	}

	rh.mtx.Unlock()
	return false
}

// Useful if ConfigManager is the first request, don't fail it in our integration test env.
// TODO(b/149525888): Remove this so we can validate ConfigManager doesn't have this hard dependency.
func (rh *RetryHandler) handleRetryExceptFirst(w http.ResponseWriter) bool {
	if rh.curNumFails == 0 {
		rh.curNumFails++
		return false
	}

	return rh.handleRetry(w)
}
