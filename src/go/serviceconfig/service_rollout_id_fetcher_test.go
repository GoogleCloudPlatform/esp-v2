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

func initServiceControlForTestNewRolloutId(t *testing.T, serviceRolloutId *string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, err := genFakeReport(*serviceRolloutId)
		if err != nil {
			t.Fatalf("fail to generate servicecontrol report response: %v", err)
		}
		_, _ = w.Write(resp)
	}))
}

func genFakeReport(serviceRolloutId string) ([]byte, error) {
	reportResp := new(scpb.ReportResponse)
	reportResp.ServiceRolloutId = serviceRolloutId
	return proto.Marshal(reportResp)
}

func TestServiceRolloutIdFetcherNewRolloutId(t *testing.T) {
	serviceRolloutId := "service-config-id"
	serviceControlServer := initServiceControlForTestNewRolloutId(t, &serviceRolloutId)
	accessToken := func() (string, time.Duration, error) { return "token", time.Duration(60), nil }

	cif := NewServiceRolloutIdFetcher("service-name", "service-control-url", http.Client{}, accessToken)

	util.FetchRolloutIdURL = func(serviceControlUrl, serviceName string) string {
		return serviceControlServer.URL
	}

	_test := func() {
		getRolloutId, err := cif.fetchNewRolloutId()
		if err != nil {
			t.Fatalf("fail to get new service config id %v", err)
		}
		if getRolloutId != serviceRolloutId {
			t.Fatalf("expect service config id: %s, get service config id: %s", serviceRolloutId, getRolloutId)
		}
	}
	_test()

	// Update the service config id.
	serviceRolloutId = "new-service-config-id"

	_test()
}
