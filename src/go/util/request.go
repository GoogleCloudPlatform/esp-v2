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

	"github.com/golang/protobuf/proto"
)

func callWithAccessToken(client *http.Client, path, method, token string) ([]byte, error) {
	req, _ := http.NewRequest(method, path, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-protobuf")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http call to %s %s returns not 200 OK: %v", method, path, resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fail to read response body: %s", err)
	}

	return body, nil
}

func CallGooglelapis(client *http.Client, path, method string, getTokenFunc GetAccessTokenFunc, output proto.Message) error {
	token, _, err := getTokenFunc()
	if err != nil {
		return fmt.Errorf("fail to get access token: %v", err)
	}

	respBytes, err := callWithAccessToken(client, path, method, token)
	if err != nil {
		return err
	}

	err = UnmarshalBytesToPbMessage(respBytes, output)
	if err != nil {
		return err
	}

	return nil
}
