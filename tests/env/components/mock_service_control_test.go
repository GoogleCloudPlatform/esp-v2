// Copyright 2018 Google Cloud Platform Proxy Authors
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
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	sc "github.com/google/go-genproto/googleapis/api/servicecontrol/v1"
)

func TestMockServiceControl(t *testing.T) {
	s := NewMockServiceCtrl("mmm")

	url := s.GetURL() + "/v1/services/mmm:check"

	req := &sc.CheckRequest{
		ServiceName: "mmm",
	}
	req_body, _ := proto.Marshal(req)

	reqq, _ := http.NewRequest("POST", url, bytes.NewReader(req_body))
	resp, err := http.DefaultClient.Do(reqq)
	if err != nil {
		t.Errorf("Failed in request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("Wrong response status: %v", resp.StatusCode)
	}

	rr, err := s.GetRequests(1, 3*time.Second)
	if err != nil {
		t.Errorf("GetRequests failed with: %v", err)
	}
	if len(rr) != 1 {
		t.Errorf("Wrong number: %d", len(rr))
	}
	if rr[0].ReqType != CHECK_REQUEST {
		t.Errorf("Wrong type: %v", rr[0].ReqType)
	}
	req1 := &sc.CheckRequest{}
	err = proto.Unmarshal(rr[0].ReqBody, req1)
	if err != nil {
		t.Errorf("failed to parse body into CheckRequest.")
	}
	if !proto.Equal(req1, req) {
		t.Errorf("Wrong request data")
	}

	// try to read it again
	rr, err = s.GetRequests(1, 1*time.Second)
	if err == nil {
		t.Errorf("Expected timeout error")
	}
}
