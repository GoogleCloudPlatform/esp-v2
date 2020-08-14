// Copyright 2020 Google LLC
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

package util

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
)

var Retries = 30
var RetryInterval = 3 * time.Second

func callWithAccessToken(client *http.Client, path, method, token string) ([]byte, int, error) {
	req, _ := http.NewRequest(method, path, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-protobuf")

	resp, err := client.Do(req)
	if err != nil {
		return nil, http.StatusOK, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("http call to %s %s returns not 200 OK: %v", method, path, resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, http.StatusOK, fmt.Errorf("fail to read response body: %s", err)
	}

	return body, http.StatusOK, nil
}

// Method to call servicecontrol for latest service rolloutId and servicecontrol for service rollout and service config.
var CallGoogleapis = func(client *http.Client, path, method string, getTokenFunc GetAccessTokenFunc, output proto.Message) error {
	token, _, err := getTokenFunc()
	if err != nil {
		return fmt.Errorf("fail to get access token: %v", err)
	}

	var respBytes []byte
	var statusCode int

	// Retry calls for code 429 Too Many Requests.
	for i := 0; i < Retries; i++ {
		respBytes, statusCode, err = callWithAccessToken(client, path, method, token)
		if statusCode != http.StatusTooManyRequests {
			break
		}

		glog.Warningf("after %v times failures because of quota limit(429 Too Many Requests), retrying http call %s with %v remaining chances", i, path, Retries-1-i)
		time.Sleep(RetryInterval)
	}

	if err != nil {
		return err
	}

	err = UnmarshalBytesToPbMessage(respBytes, output)
	if err != nil {
		return err
	}

	return nil
}
