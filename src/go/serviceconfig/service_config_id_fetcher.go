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
// See the License for the specfic language governing permissions and
// limitations under the License.

package serviceconfig

import (
	"fmt"
	"net/http"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/proto"
	scpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
)

type ServiceConfigIdFetcher struct {
	serviceName       string
	serviceControlUrl string
	client            http.Client
	accessToken       func() (string, time.Duration, error)
}

func NewServiceConfigIdFetcher(serviceName, serviceControlUrl string, client http.Client,
	accessToken GetAccessTokenFunc) *ServiceConfigIdFetcher {
	return &ServiceConfigIdFetcher{
		serviceName:       serviceName,
		serviceControlUrl: serviceControlUrl,
		client:            client,
		accessToken:       accessToken,
	}

}

func (c *ServiceConfigIdFetcher) fetchNewConfigId() (string, error) {
	token, _, err := c.accessToken()
	if err != nil {
		return "", fmt.Errorf("fail to get access token: %v", err)
	}

	reportResponse, err := c.callServiceControl(util.FetchConfigIdURL(c.serviceControlUrl, c.serviceName), token)
	if err != nil {
		return "", err
	}

	return reportResponse.ServiceConfigId, nil
}

func (c *ServiceConfigIdFetcher) callServiceControl(path, token string) (*scpb.ReportResponse, error) {
	respBytes, err := util.CallWithAccessToken(c.client, util.POST, path, token)
	if err != nil {
		return nil, err
	}

	reportResponse := new(scpb.ReportResponse)
	if err := proto.Unmarshal(respBytes, reportResponse); err != nil {
		return nil, fmt.Errorf("fail to unmarshal ReportResponse: %s", err)
	}

	return reportResponse, nil
}
