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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/golang/protobuf/proto"

	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func initServiceManagementForTestServbiceConfigFetcherFetchConfig(t *testing.T, serviceConfig *confpb.Service, serviceName string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if wantPath := fmt.Sprintf("/v1/services/%s/configs/%s", serviceName, serviceConfig.Id); r.URL.Path != wantPath {
			t.Fatalf("want path: %s, get path: %s", wantPath, r.URL.Path)
		}
		resp, err := genFakeServiceConfig(serviceConfig)
		if err != nil {
			t.Fatalf("fail to generate servicecontrol report response: %v", err)
		}
		_, _ = w.Write(resp)
	}))
}

func genFakeServiceConfig(serviceConfig *confpb.Service) ([]byte, error) {
	return proto.Marshal(serviceConfig)
}

func TestServiceConfigFetcherFetchConfig(t *testing.T) {
	serviceConfigId := "test-config-id"
	serviceName := "service-name"
	serviceConfig := &confpb.Service{
		Name: "foo",
		Id:   serviceConfigId,
	}

	serviceManagementServer := initServiceManagementForTestServbiceConfigFetcherFetchConfig(t, serviceConfig, serviceName)
	opts := options.DefaultConfigGeneratorOptions()
	opts.ServiceManagementURL = serviceManagementServer.URL

	scf, err := NewServiceConfigFetcher(&opts, serviceName, func() (string, time.Duration, error) { return "access-token", time.Duration(60), nil })
	if err != nil {
		t.Fatal(err)
	}

	scf.newConfigId = func() (string, error) { return serviceConfigId, nil }

	_test := func(configId string, wantServiceConfig *confpb.Service, wantError string) {
		getConfig, err := scf.FetchConfig(configId)
		if err != nil {
			if wantError == "" {
				t.Fatalf("fail to fetch config %v", err)
			}

			if err.Error() != wantError {
				t.Fatalf("want error: %s, get error: %v", wantError, err)
			}
			return
		}

		if getConfig == nil {
			if wantServiceConfig != nil {
				t.Fatalf("want service config: %v, get service config: nil", wantServiceConfig)
			}
			return
		}

		if !proto.Equal(getConfig, serviceConfig) {
			t.Fatalf("want service config: %v, get service config: %v", serviceConfig, getConfig)
		}
	}

	// No config id specified.
	_test("", serviceConfig, "")

	// Config id specified.
	serviceConfigId = "test-config-id-1"
	serviceConfig.Id = serviceConfigId
	_test(serviceConfigId, serviceConfig, "")

	// No service config really fetched when service config id is same.
	_test(serviceConfigId, nil, "")

	// Error caused by failing to get new config id.
	serviceConfigId = "test-config-id-2"
	serviceConfig.Id = serviceConfigId
	scf.newConfigId = func() (string, error) { return "", fmt.Errorf("newConfigIdError") }
	_test("", nil, "error occurred when checking new service config id: newConfigIdError")

	// Error caused by failing to get access token.
	serviceConfigId = "test-config-id-3"
	serviceConfig.Id = serviceConfigId
	scf.newConfigId = func() (string, error) { return serviceConfigId, nil }
	scf.accessToken = func() (string, time.Duration, error) { return "", time.Duration(0), fmt.Errorf("accessTokenError") }
	_test("", nil, "fail to get access token: accessTokenError")
}

func TestServiceConfigFetcher_SetFetchConfigTimer(t *testing.T) {
	serviceConfigId := "test-config-id"
	serviceName := "service-name"
	serviceConfig := &confpb.Service{
		Name: "foo",
		Id:   serviceConfigId,
	}

	serviceManagementServer := initServiceManagementForTestServbiceConfigFetcherFetchConfig(t, serviceConfig, serviceName)
	opts := options.DefaultConfigGeneratorOptions()
	opts.ServiceManagementURL = serviceManagementServer.URL

	scf, err := NewServiceConfigFetcher(&opts, serviceName, func() (string, time.Duration, error) { return "access-token", time.Duration(60), nil })
	if err != nil {
		t.Fatal(err)
	}

	scf.newConfigId = func() (string, error) { return serviceConfigId, nil }
	cnt := 0
	scf.SetFetchConfigTimer(time.Millisecond*100, func(getService *confpb.Service) {
		cnt += 1
		if !proto.Equal(getService, serviceConfig) {
			t.Fatalf("want service %v, get service %v", serviceConfig, getService)
		}

		// Update service config so fetchConfig will do real fetching.
		serviceConfigId = fmt.Sprintf("test-config-id-%v", cnt)
		serviceConfig.Id = serviceConfigId
	})
	wantCnt := 10
	time.Sleep(time.Millisecond * time.Duration(100*wantCnt))

	// grace buffer
	if cnt < wantCnt-2 {
		t.Fatalf("want callback called by %v times, get %v times", wantCnt, cnt)
	}
}
