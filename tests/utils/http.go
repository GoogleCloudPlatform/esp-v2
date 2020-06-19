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

package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// rpcStatus is a type, used to parse the json format of google.rpc.Status,
// https://github.com/googleapis/googleapis/blob/master/google/rpc/status.proto.
type rpcStatus struct {
	Code    uint16
	Message string
}

func (r rpcStatus) toString() string {
	return fmt.Sprintf("{\"code\":%v,\"message\":\"%s\"}", r.Code, r.Message)
}

// RpcStatusDeterministicJsonFormat converts the unordered json format of
// rpcStatus to an ordered one.
func RpcStatusDeterministicJsonFormat(jsonBytes []byte) string {
	var jsonErr rpcStatus
	_ = json.Unmarshal(jsonBytes, &jsonErr)
	return jsonErr.toString()
}

// DoWithHeaders performs a GET/POST/PUT/DELETE/PATCH request to a specified url
// with given headers and message(if provided)
func DoWithHeaders(url, method, message string, headers map[string]string) (http.Header, []byte, error) {
	var request *http.Request
	var err error
	if method == "DELETE" || method == "GET" {
		request, err = http.NewRequest(method, url, nil)
	} else {
		msg := map[string]string{
			"message": message,
		}
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(msg); err != nil {
			return nil, nil, err
		}
		request, err = http.NewRequest(method, url, &buf)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("create request error: %v", err)
	}

	if message != "" {
		request.Header.Set("Content-Type", "application/json")
	}

	for k, v := range headers {
		request.Header.Set(k, v)
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}

	resp, err := client.Do(request)
	if err != nil {
		return nil, nil, fmt.Errorf("http %s error: %v", method, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("http got error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("http response status is not 200 OK: %s, %s", resp.Status, RpcStatusDeterministicJsonFormat(bodyBytes))
	}
	return resp.Header, bodyBytes, err
}
