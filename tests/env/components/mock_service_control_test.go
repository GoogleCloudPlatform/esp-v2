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
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	scpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
)

func TestMockServiceControl(t *testing.T) {
	s := NewMockServiceCtrl("mmm", "test-rollout-id")
	s.Setup()

	url := s.GetURL() + "/v1/services/mmm:check"

	req := &scpb.CheckRequest{
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

	rr, err := s.GetRequests(1)
	if err != nil {
		t.Errorf("GetRequests failed with: %v", err)
	}
	if len(rr) != 1 {
		t.Errorf("Wrong number: %d", len(rr))
	}
	if rr[0].ReqType != CHECK_REQUEST {
		t.Errorf("Wrong type: %v", rr[0].ReqType)
	}
	req1 := &scpb.CheckRequest{}
	err = proto.Unmarshal(rr[0].ReqBody, req1)
	if err != nil {
		t.Errorf("failed to parse body into CheckRequest.")
	}
	if !proto.Equal(req1, req) {
		t.Errorf("Wrong request data")
	}

	// try to read it again
	s.SetGetRequestsTimeout(500 * time.Millisecond)
	rr, err = s.GetRequests(1)
	if err == nil {
		t.Errorf("Expected timeout error")
	}
}

func TestMockServiceControlCheckError(t *testing.T) {
	testdata := []struct {
		name              string
		checkResponse     *scpb.CheckResponse
		wantCheckResponse *scpb.CheckResponse
	}{
		{
			name: "mmm",
			wantCheckResponse: &scpb.CheckResponse{
				CheckInfo: &scpb.CheckResponse_CheckInfo{
					ConsumerInfo: &scpb.CheckResponse_ConsumerInfo{
						ProjectNumber: 123456,
					},
				},
			},
		},
		{
			name: "mmm",
			checkResponse: &scpb.CheckResponse{
				CheckInfo: &scpb.CheckResponse_CheckInfo{
					ConsumerInfo: &scpb.CheckResponse_ConsumerInfo{
						ProjectNumber: 123456,
					},
				},
				CheckErrors: []*scpb.CheckError{
					{
						Code: scpb.CheckError_API_KEY_INVALID,
					},
				},
			},
			wantCheckResponse: &scpb.CheckResponse{
				CheckInfo: &scpb.CheckResponse_CheckInfo{
					ConsumerInfo: &scpb.CheckResponse_ConsumerInfo{
						ProjectNumber: 123456,
					},
				},
				CheckErrors: []*scpb.CheckError{
					{
						Code: scpb.CheckError_API_KEY_INVALID,
					},
				},
			},
		},
	}

	for _, tc := range testdata {
		s := NewMockServiceCtrl(tc.name, "test-rollout-id")
		s.Setup()
		if tc.checkResponse != nil {
			s.SetCheckResponse(tc.checkResponse)
		}

		url := s.GetURL() + "/v1/services/mmm:check"
		req := &scpb.CheckRequest{
			ServiceName: tc.name,
		}
		req_body, _ := proto.Marshal(req)
		reqq, _ := http.NewRequest("POST", url, bytes.NewReader(req_body))
		resp, err := http.DefaultClient.Do(reqq)
		if err != nil {
			t.Errorf("Failed in request: %v", err)
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("Failed in reading response: %v", err)
		}

		parsedResp := &scpb.CheckResponse{}
		err = proto.Unmarshal(body, parsedResp)
		if err != nil {
			t.Errorf("Failed to parse body into CheckResponse.")
		}

		if !proto.Equal(parsedResp, tc.wantCheckResponse) {
			t.Errorf("Wrong response data, want: %v, get: %v.", parsedResp, tc.wantCheckResponse)
		}
	}
}

func TestMockServiceControlReportStatus(t *testing.T) {
	testdata := []struct {
		name                 string
		reportStatusCode     int
		wantReportStatusCode int
	}{
		{
			name:                 "mmm",
			wantReportStatusCode: 200,
		},
		{
			name:                 "mmm",
			reportStatusCode:     http.StatusInternalServerError,
			wantReportStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range testdata {
		s := NewMockServiceCtrl(tc.name, "test-rollout-id")
		s.Setup()
		if tc.reportStatusCode != 0 {
			s.SetReportResponseStatus(tc.reportStatusCode)
		}

		url := s.GetURL() + "/v1/services/mmm:report"
		req := &scpb.CheckRequest{
			ServiceName: tc.name,
		}
		req_body, _ := proto.Marshal(req)
		reqq, _ := http.NewRequest("POST", url, bytes.NewReader(req_body))
		resp, err := http.DefaultClient.Do(reqq)
		if err != nil {
			t.Errorf("Failed in request: %v", err)
		}
		defer resp.Body.Close()
		if tc.wantReportStatusCode != resp.StatusCode {
			t.Errorf("Failed in status code, expected: %v, go: %v", tc.wantReportStatusCode, resp.StatusCode)
		}
	}
}
