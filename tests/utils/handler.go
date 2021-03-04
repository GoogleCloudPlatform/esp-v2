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

package utils

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang/glog"
)

type RetryServiceHandler struct {
	RequestCount  int32
	SleepTimes    int32
	SleepLengthMs int32
}

func (h *RetryServiceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.RequestCount += 1
	if h.RequestCount <= h.SleepTimes {
		time.Sleep(time.Millisecond * time.Duration(h.SleepLengthMs))
	}

	w.Write([]byte(""))
}

type ExpectHeaderHandler struct {
	RequestCount    int32
	ExpectedHeaders http.Header
	T               *testing.T
}

func (h *ExpectHeaderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.RequestCount += 1
	glog.Infof("Fake ExpectHeaderHandler got headers: %v", r.Header)

	for key, vals := range h.ExpectedHeaders {
		headerMatch := CheckHeaderExist(r.Header, key, func(gotHeaderVal string) bool {
			return strings.Contains(gotHeaderVal, strings.Join(vals, ";"))
		})

		if !headerMatch {
			h.T.Errorf("expected headers %v, got headers %v", h.ExpectedHeaders, r.Header)
		}
	}

	w.Write([]byte(""))
}
