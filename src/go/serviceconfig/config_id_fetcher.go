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

package serviceconfig

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/proto"
	scpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
)

type ConfigIdFetcher struct {
	serviceName string
	client      *http.Client
	accessToken func() (string, time.Duration, error)
}

func NewConfigIdFetcher(serviceName string, client *http.Client, accessToken func() (string, time.Duration, error)) (*ConfigIdFetcher, error) {
	return &ConfigIdFetcher{
		serviceName: serviceName,
		client:      client,
		accessToken: accessToken,
	}, nil
}

func (cif *ConfigIdFetcher) latestConfigId() (string, error) {
	token, _, err := cif.accessToken()
	if err != nil {
		return "", fmt.Errorf("fail to get access token: %v", err)
	}
	reportResponse, err := cif.callServiceControl(util.FetchConfigIdURL(cif.serviceName), token)
	if err != nil {
		return "", err
	}
	return reportResponse.ServiceConfigId, nil
}

func (scf *ConfigIdFetcher) callWithAccessToken(path, token string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", path, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-protobuf")
	resp, err := scf.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("http call to %s returns not 200 OK: %v", path, resp.Status)
	}
	return resp, nil
}

func (cif *ConfigIdFetcher) callServiceControl(path, token string) (*scpb.ReportResponse, error) {
	var err error
	var resp *http.Response
	if resp, err = cif.callWithAccessToken(path, token); err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fail to read response body: %s", err)
	}
	defer resp.Body.Close()
	reportResponse := new(scpb.ReportResponse)
	if err := proto.Unmarshal(body, reportResponse); err != nil {
		return nil, fmt.Errorf("fail to unmarshal ListServiceRolloutsResponse: %s", err)
	}
	fmt.Printf("%v\n%v", reportResponse, err)
	return reportResponse, nil
}
