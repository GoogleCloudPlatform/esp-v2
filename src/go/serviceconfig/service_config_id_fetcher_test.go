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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/proto"
	scpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
)

func initServiceControlForTestNewConfigId(t *testing.T, serviceConfigId *string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, err := genFakeReport(*serviceConfigId)
		if err != nil {
			t.Fatalf("fail to generate servicecontrol report response: %v", err)
		}
		_, _ = w.Write(resp)
	}))
}

func genFakeReport(serviceConfigId string) ([]byte, error) {
	reportResp := new(scpb.ReportResponse)
	reportResp.ServiceConfigId = serviceConfigId
	return proto.Marshal(reportResp)
}

func TestServiceConfigIdFetcherNewConfigId(t *testing.T) {
	serviceConfigId := "service-config-id"
	serviceControlServer := initServiceControlForTestNewConfigId(t, &serviceConfigId)
	accessToken := func() (string, time.Duration, error) { return "token", time.Duration(60), nil }

	cif := NewServiceConfigIdFetcher("service-name", "service-control-url", http.Client{}, accessToken)

	util.FetchConfigIdURL = func(serviceControlUrl, serviceName string) string {
		return serviceControlServer.URL
	}

	_test := func() {
		getConfigId, err := cif.fetchNewConfigId()
		if err != nil {
			t.Fatalf("fail to get new service config id %v", err)
		}
		if getConfigId != serviceConfigId {
			t.Fatalf("expect service config id: %s, get service config id: %s", serviceConfigId, getConfigId)
		}
	}
	_test()

	// Update the service config id.
	serviceConfigId = "new-service-config-id"

	_test()
}
