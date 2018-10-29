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
  "net/http"
  "net/http/httptest"
  "testing"
  "reflect"

    //googleapi "google.golang.org/api/googleapi"
)

const (
  serviceName = "bookstore.test.appspot.com"

  fakeConfig = `{ "name": "bookstore.test.appspot.com", "title": "Bookstore","id": "2017-05-01r0"}`
)

func TestFetchRollouts(t *testing.T) {
  runTest(t, func(env *testEnv) {
    err := env.configManager.Init("2017-05-01r0")
    if err != nil {
      t.Errorf("Init() got error: %v, want nil", err)
    }
    expectedRolloutInfo := rolloutInfo{
      configs: map[string]string{
        "2017-05-01r0": fakeConfig,
      },
    }
    if !reflect.DeepEqual(*env.configManager.rolloutInfo, expectedRolloutInfo) {
      t.Errorf("Init() got config: %v, want: %v", *env.configManager.rolloutInfo, expectedRolloutInfo)
    }
  })
}

// Test Environment setup.

type testEnv struct {
  configManager *ConfigManager
}

func runTest(t *testing.T, f func(*testEnv)) {
mockConfig := initMockConfigServer()
  defer mockConfig.Close()
  fetchConfigURL = mockConfig.URL

  mockMetadata := initMockMetadataServer()
  defer mockMetadata.Close()
  serviceAccountTokenURL = mockMetadata.URL

  manager, err := NewConfigManager(serviceName)
  if err != nil {
    t.Fatal("fail to initialize ConfigManager")
  }
  env := &testEnv{
    configManager: manager,
  }
  f(env)
}

func initMockConfigServer() *httptest.Server {
  return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(fakeConfig))
  }))
}