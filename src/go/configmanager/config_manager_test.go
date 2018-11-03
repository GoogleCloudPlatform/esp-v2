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

package configmanager

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/jsonpb"

	"cloudesf.googlesource.com/gcpproxy/src/go/proto/google/api"
	gp "cloudesf.googlesource.com/gcpproxy/src/go/proto/google/protobuf"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
)

const (
	testServiceName = "bookstore.test.appspot.com"
	testConfigID    = "2017-05-01r0"
	fakeNodeID      = "id"
)

var (
	fakeConfig = &api.Service{
		Name:  testServiceName,
		Title: "Bookstore",
		Id:    testConfigID,
		Apis: []*gp.Api{
			{
				Name: testServiceName + ".v1.BookstoreService",
			},
		},
	}
)

func TestFetchRollouts(t *testing.T) {
	runTest(t, func(env *testEnv) {
		ctx := context.Background()
		// First request, VersionId should be empty.
		req := v2.DiscoveryRequest{
			Node: &core.Node{
				Id: node,
			},
			TypeUrl: cache.ListenerType,
		}

		resp, err := env.configManager.cache.Fetch(ctx, req)
		if err != nil {
			t.Fatal(err)
		}

		marshaler := &jsonpb.Marshaler{}
		gotListeners, err := marshaler.MarshalToString(resp.Resources[0])

		wantedListeners := `{"address":{"socketAddress":{"address":"0.0.0.0","portValue":8080}},"filterChains":[{"filters":[{"name":"envoy.http_connection_manager","config":{"http_filters":[{"config":{"proto_descriptor":"","services":["bookstore.test.appspot.com.v1.BookstoreService"]},"name":"envoy.grpc_json_transcoder"}],"rds":{"config_source":{"ads":{}}},"stat_prefix":"ingress_http"}}]}]}`
		if resp.Version != testConfigID {
			t.Errorf("snapshot cache fetch got version: %v, want: %v", resp.Version, testConfigID)
		}
		if !reflect.DeepEqual(resp.Request, req) {
			t.Errorf("snapshot cache fetch got request: %v, want: %v", resp.Request, req)
		}
		if gotListeners != wantedListeners {
			t.Errorf("snapshot cache fetch got Listeners: %s, want: %s", gotListeners, wantedListeners)
		}
	})
}

// Test Environment setup.

type testEnv struct {
	configManager *ConfigManager
}

func runTest(t *testing.T, f func(*testEnv)) {
	mockConfig := initMockConfigServer(t)
	defer mockConfig.Close()
	fetchConfigURL = mockConfig.URL

	mockMetadata := initMockMetadataServer()
	defer mockMetadata.Close()
	serviceAccountTokenURL = mockMetadata.URL

	manager, err := NewConfigManager(testServiceName, testConfigID)
	if err != nil {
		t.Fatal("fail to initialize ConfigManager")
	}

	env := &testEnv{
		configManager: manager,
	}
	f(env)
}

func initMockConfigServer(t *testing.T) *httptest.Server {
	body, err := json.Marshal(fakeConfig)
	if err != nil {
		t.Fatal("json.Marshal failed")
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
}

type mock struct{}

func (mock) ID(*core.Node) string {
	return fakeNodeID
}
