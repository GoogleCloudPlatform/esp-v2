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
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
)

type RetryConfig struct {
	RetryNum      int
	RetryInterval time.Duration
}

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

// CallGoogleapisMu guards the access to CallGoogleapis. This is used in the test to fake CallGoogleapis.
var CallGoogleapisMu sync.RWMutex

// Method to call servicecontrol for latest service rolloutId and servicecontrol for service rollout and service config.
var CallGoogleapis = func(client *http.Client, path, method string, getTokenFunc GetAccessTokenFunc, retryConfigs map[int]RetryConfig, output proto.Message) error {
	token, _, err := getTokenFunc()
	if err != nil {
		return fmt.Errorf("fail to get access token: %v", err)
	}

	var respBytes []byte
	var statusCode int

	callStatusCnts := map[int]int{}

	for {
		respBytes, statusCode, err = callWithAccessToken(client, path, method, token)
		if retryConfigs == nil {
			break
		} else if retryConfig, ok := retryConfigs[statusCode]; !ok {
			break
		} else if retryConfig.RetryNum <= callStatusCnts[statusCode] {
			break
		} else {
			callStatusCnts[statusCode] += 1
			glog.Warningf("after %v failures on status %v, retrying http call %s with %v remaining chances", callStatusCnts[statusCode], statusCode, path, retryConfig.RetryNum-callStatusCnts[statusCode])

			time.Sleep(retryConfig.RetryInterval)
		}
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
