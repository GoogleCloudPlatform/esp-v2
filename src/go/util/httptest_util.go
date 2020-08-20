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

package util

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"time"
)

type MockServer struct {
	s             *httptest.Server
	sleepDuration time.Duration
	resp          string
}

func InitMockServer(response string) *MockServer {
	m := &MockServer{
		resp: response,
	}
	m.s = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(m.sleepDuration)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(m.resp))
	}))
	return m
}

func (m *MockServer) SetResp(response string) {
	m.resp = response
}

func (m *MockServer) GetURL() string {
	return m.s.URL
}

func (m *MockServer) Close() {
	m.s.Close()
}

func (m *MockServer) SetSleepTime(sleepDuration time.Duration) {
	m.sleepDuration = sleepDuration
}
func InitMockServerFromPathResp(pathResp map[string]string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Root is used to tell if the sever is healthy or not.
		if r.URL.Path == "" || r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if resp, ok := pathResp[r.URL.Path]; ok {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(resp))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
}

// JsonEqual compares two JSON strings after normalizing them.
// Should be used for test only.
func JsonEqual(want, got string) error {
	var err error
	if got, err = normalizeJson(got); err != nil {
		return err
	}
	if want, err = normalizeJson(want); err != nil {
		return err
	}
	if !strings.EqualFold(want, got) {
		return fmt.Errorf("\n  got: %s \n want: %s", got, want)
	}
	return nil
}

// JsonContains should be used for test only. It can check whether partial
// target json string belong to the source json string.
// JsonContains will remove regex(`(\t|\n|\s)`) inside target so it shouldn't
// contains these chars in its content.
func JsonContains(source, target string) error {
	normalizedSource, err := normalizeJson(source)
	if err != nil {
		return err
	}

	var re = regexp.MustCompile(`(\t|\n|\s)`)
	normalizedTarget := re.ReplaceAllString(target, "")

	if !strings.Contains(normalizedSource, normalizedTarget) {
		return fmt.Errorf("source doesn't contain the target,\nsource: %s,\ntarget: %s\n", normalizedSource, normalizedTarget)
	}
	return nil
}

// normalizeJson returns normalized JSON string.
func normalizeJson(input string) (string, error) {
	var jsonObject map[string]interface{}
	if err := json.Unmarshal([]byte(input), &jsonObject); err != nil {
		return "", err
	}
	outputString, err := json.Marshal(jsonObject)
	return string(outputString), err
}
