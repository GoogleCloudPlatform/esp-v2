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
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/api-proxy/src/go/util"
)

func doRequest(action, url string) (int, string, error) {
	reqq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, "", err
	}
	resp, err := http.DefaultClient.Do(reqq)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, "", err
	}
	respStr := string(bodyBytes)
	return resp.StatusCode, respStr, nil
}

func TestMockMetadata(t *testing.T) {
	s := NewMockMetadata(map[string]string{"/foo_key": "foo_val", "/foo?bar": "foo_bar_val"})

	testdata := []struct {
		desc         string
		url          string
		wantedResp   string
		wantedStatus int
		wantReqCnt   int
	}{
		{
			desc:       "Success, simple case",
			url:        "/foo_key",
			wantedResp: "foo_val",
			wantReqCnt: 1,
		},
		{
			desc:       "Success, test GetReqCnt",
			url:        "/foo_key",
			wantedResp: "foo_val",
			wantReqCnt: 2,
		},
		{
			desc:         "Fail, query nonexist metadata",
			url:          "/nonexist_key",
			wantedStatus: http.StatusNotFound,
		},
		{
			desc:       "Success, query hard-code metadata",
			url:        util.ConfigIDSuffix,
			wantedResp: fakeConfigID,
		},
		{
			desc:       "Success, using query url",
			url:        "/foo?bar",
			wantedResp: "foo_bar_val",
		},
	}

	for _, tc := range testdata {
		url := s.GetURL() + tc.url
		status, resp, err := doRequest("GET", url)
		if err != nil {
			t.Errorf("Test (%s): got err, %s", tc.desc, err.Error())
		}
		if tc.wantedStatus != 0 && (status == http.StatusOK || status != tc.wantedStatus) {
			t.Errorf("Test (%s): wrong response status: %v", tc.desc, status)
		} else if tc.wantedResp != "" && !strings.Contains(resp, tc.wantedResp) {
			t.Errorf("Test (%s): failed, expected response: %s, got response: %s", tc.desc, tc.wantedResp, resp)
		}
		if cnt := s.GetReqCnt(tc.url); tc.wantReqCnt != 0 && cnt != tc.wantReqCnt {
			t.Errorf("Test (%s): failed, expected request count: %v, got request count: %v", tc.desc, 0, cnt)
		}
	}
}
