// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0 //
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configmanager

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configmanager/testdata"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/metadata"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"

	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discoverypb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	servicecontrolpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
)

func TestFetchListeners(t *testing.T) {
	var fakeConfig, fakeScReport, fakeRollouts safeData

	testData := []struct {
		desc              string
		enableTracing     bool
		enableDebug       bool
		BackendAddress    string
		fakeServiceConfig string
		wantedListeners   string
	}{
		{
			desc:              "Success for grpc backend with transcoding",
			BackendAddress:    "grpc://127.0.0.1:80",
			fakeServiceConfig: testdata.FakeServiceConfigForGrpcWithTranscoding,
			wantedListeners:   testdata.WantedListsenerForGrpcWithTranscoding,
		},
		{
			desc:              "Success for grpc backend, with Jwt filter, with audiences, no Http Rules",
			BackendAddress:    "grpc://127.0.0.1:80",
			fakeServiceConfig: testdata.FakeServiceConfigForGrpcWithJwtFilterWithAuds,
			wantedListeners:   testdata.WantedListsenerForGrpcWithJwtFilterWithAuds,
		},
		{
			desc:              "Success for gRPC backend, with Jwt filter, without audiences",
			BackendAddress:    "grpc://127.0.0.1:80",
			fakeServiceConfig: testdata.FakeServiceConfigForGrpcWithJwtFilterWithoutAuds,
			wantedListeners:   testdata.WantedListsenerForGrpcWithJwtFilterWithoutAuds,
		},
		{
			desc:              "Success for gRPC backend, with Jwt filter, with multi requirements, matching with regex",
			BackendAddress:    "grpc://127.0.0.1:80",
			fakeServiceConfig: testdata.FakeServiceConfigForGrpcWithJwtFilterWithMultiReqs,
			wantedListeners:   testdata.WantedListenerForGrpcWithJwtFilterWithMultiReqs,
		},
		{
			desc:              "Success for gRPC backend with Service Control",
			BackendAddress:    "grpc://127.0.0.1:80",
			fakeServiceConfig: testdata.FakeServiceConfigForGrpcWithServiceControl,
			wantedListeners:   testdata.WantedListenerForGrpcWithServiceControl,
		},
		{
			desc:              "Success for http backend, with Jwt filter, with audiences",
			BackendAddress:    "http://127.0.0.1:80",
			fakeServiceConfig: testdata.FakeServiceConfigForHttp,
			wantedListeners:   testdata.WantedListenerForHttp,
		},
		{
			desc:              "Success for backend that allow CORS, with tracing and debug enabled",
			enableTracing:     true,
			enableDebug:       true,
			BackendAddress:    "http://127.0.0.1:80",
			fakeServiceConfig: testdata.FakeServiceConfigAllowCorsTracingDebug,
			wantedListeners:   testdata.WantedListenersAllowCorsTracingDebug,
		},
	}

	for i, tc := range testData {
		// Overrides fakeConfig for the test case.

		if err := genProtoBinary(tc.fakeServiceConfig, new(confpb.Service), &fakeConfig); err != nil {
			t.Fatalf("generate fake service config failed: %v", err)
		}

		opts := options.DefaultConfigGeneratorOptions()
		opts.BackendAddress = tc.BackendAddress
		opts.DisableTracing = !tc.enableTracing
		opts.SuppressEnvoyHeaders = !tc.enableDebug

		setFlags(testdata.TestFetchListenersProjectName, testdata.TestFetchListenersConfigID, util.FixedRolloutStrategy, "100ms", "")

		runTest(t, &fakeScReport, &fakeRollouts, &fakeConfig, opts, func(configManager *ConfigManager) {
			ctx := context.Background()
			// First request, VersionId should be empty.
			req := discoverypb.DiscoveryRequest{
				Node: &corepb.Node{
					Id: opts.Node,
				},
				TypeUrl: resource.ListenerType,
			}
			respInterface, err := configManager.cache.Fetch(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			resp := respInterface.(cache.Response)

			marshaler := &jsonpb.Marshaler{
				AnyResolver: util.Resolver,
			}
			gotListeners, err := marshaler.MarshalToString(resp.Resources[0])
			if err != nil {
				t.Fatal(err)
			}

			if resp.Version != testdata.TestFetchListenersConfigID {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got version: %v, want: %v", i, tc.desc, resp.Version, testdata.TestFetchListenersConfigID)
			}
			if !proto.Equal(&resp.Request, &req) {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got request: %v, want: %v", i, tc.desc, resp.Request, req)
			}

			if err := util.JsonEqual(tc.wantedListeners, gotListeners); err != nil {
				t.Errorf("Test Desc(%d): %s, snapshot cache fetch got unexpected Listeners, %v", i, tc.desc, err)
			}
		})
	}
}

func TestFixedModeDynamicRouting(t *testing.T) {
	testData := []struct {
		desc              string
		serviceConfigPath string
		wantedClusters    []string
		wantedListener    string
	}{
		{
			desc:              "Success for http with dynamic routing with fixed config",
			serviceConfigPath: platform.GetFilePath(platform.FixedDrServiceConfig),
			wantedClusters:    testdata.FakeWantedClustersForDynamicRouting,
			wantedListener:    testdata.FakeWantedListenerForDynamicRouting,
		},
	}

	marshaler := &jsonpb.Marshaler{}
	for i, tc := range testData {
		opts := options.DefaultConfigGeneratorOptions()
		opts.DisableTracing = true

		_ = flag.Set("service_json_path", tc.serviceConfigPath)

		manager, err := NewConfigManager(nil, opts)
		if err != nil {
			t.Fatal("fail to initialize Config Manager: ", err)
		}
		ctx := context.Background()
		// First request, VersionId should be empty.
		reqForClusters := discoverypb.DiscoveryRequest{
			Node: &corepb.Node{
				Id: opts.Node,
			},
			TypeUrl: resource.ClusterType,
		}

		respInterface, err := manager.cache.Fetch(ctx, reqForClusters)
		if err != nil {
			t.Fatal(err)
		}
		respForClusters := respInterface.(cache.Response)

		if !proto.Equal(&respForClusters.Request, &reqForClusters) {
			t.Errorf("Test Desc(%d): %s, snapshot cache fetch got request: %v, want: %v", i, tc.desc, respForClusters.Request, reqForClusters)
			continue
		}

		sortedClusters := sortResources(respForClusters)

		if len(sortedClusters) != len(tc.wantedClusters) {
			t.Errorf("Test Desc(%d): %s, snapshot cache fetch got clusters: %v, want: %v", i, tc.desc, sortedClusters, tc.wantedClusters)
			continue
		}

		for idx, want := range tc.wantedClusters {
			gotCluster, err := marshaler.MarshalToString(sortedClusters[idx])
			if err != nil {
				t.Error(err)
				continue
			}
			if err := util.JsonEqual(want, gotCluster); err != nil {
				t.Errorf("Test Desc(%d): %s, idx %d snapshot cache fetch got Cluster: \n%v", i, tc.desc, idx, err)
				continue
			}
		}

		reqForListener := discoverypb.DiscoveryRequest{
			Node: &corepb.Node{
				Id: opts.Node,
			},
			TypeUrl: resource.ListenerType,
		}

		respInterface, err = manager.cache.Fetch(ctx, reqForListener)
		if err != nil {
			t.Error(err)
			continue
		}
		respForListener := respInterface.(cache.Response)

		if respForListener.Version != testdata.TestFetchListenersConfigID {
			t.Errorf("Test Desc(%d): %s, snapshot cache fetch got version: %v, want: %v", i, tc.desc, respForListener.Version, testdata.TestFetchListenersConfigID)
			continue
		}
		if !proto.Equal(&respForListener.Request, &reqForListener) {
			t.Errorf("Test Desc(%d): %s, snapshot cache fetch got request: %v, want: %v", i, tc.desc, respForListener.Request, reqForListener)
			continue
		}

		gotListener, err := marshaler.MarshalToString(respForListener.Resources[0])
		if err != nil {
			t.Error(err)
			continue
		}
		if err := util.JsonEqual(tc.wantedListener, gotListener); err != nil {
			t.Errorf("Test Desc(%d): %s, snapshot cache fetch Listener,\n\t %v", i, tc.desc, err)
		}
	}
}

func TestServiceConfigAutoUpdate(t *testing.T) {
	var fakeConfig, fakeScReport, fakeRollouts safeData

	var oldConfigID, oldRolloutID, newConfigID, newRolloutID string
	oldConfigID = "2018-12-05r0"
	oldRolloutID = oldConfigID
	newConfigID = "2018-12-05r1"
	newRolloutID = newConfigID

	testProjectName := "bookstore.endpoints.project123.cloud.goog"
	testEndpointName := "endpoints.examples.bookstore.Bookstore"
	testConfigID := "2017-05-01r0"

	tc := struct {
		desc                  string
		fakeOldScReport       string
		fakeNewScReport       string
		fakeOldServiceRollout string
		fakeNewServiceRollout string
		fakeOldServiceConfig  string
		fakeNewServiceConfig  string
		BackendAddress        string
	}{
		desc: "Success for service config auto update",
		fakeOldScReport: fmt.Sprintf(`{
                "serviceConfigId": "%s",
                "serviceRolloutId": "%s"
            }`, oldRolloutID, oldConfigID),
		fakeNewScReport: fmt.Sprintf(`{
                "serviceConfigId": "%s",
                "serviceRolloutId": "%s"
            }`, newRolloutID, newConfigID),
		fakeOldServiceRollout: fmt.Sprintf(`{
            "rollouts": [
                {
                  "rolloutId": "%s",
                  "createTime": "2018-12-05T19:07:18.438Z",
                  "createdBy": "mocktest@google.com",
                  "status": "SUCCESS",
                  "trafficPercentStrategy": {
                    "percentages": {
                      "%s": 100
                    }
                  },
                  "serviceName": "%s"
                }
              ]
            }`, oldRolloutID, oldConfigID, testProjectName),
		fakeNewServiceRollout: fmt.Sprintf(`{
            "rollouts": [
                {
                  "rolloutId": "%s",
                  "createTime": "2018-12-05T19:07:18.438Z",
                  "createdBy": "mocktest@google.com",
                  "status": "SUCCESS",
                  "trafficPercentStrategy": {
                    "percentages": {
                      "%s": 40,
                      "%s": 60
                    }
                  },
                  "serviceName": "%s"
                },
                {
                  "rolloutId": "%s",
                  "createTime": "2018-12-05T19:07:18.438Z",
                  "createdBy": "mocktest@google.com",
                  "status": "SUCCESS",
                  "trafficPercentStrategy": {
                    "percentages": {
                      "%s": 100
                    }
                  },
                  "serviceName": "%s"
                }
              ]
            }`, newRolloutID, oldConfigID, newConfigID, testProjectName,
			oldRolloutID, oldConfigID, testProjectName),
		fakeOldServiceConfig: fmt.Sprintf(`{
                "name": "%s",
                "title": "Endpoints Example",
                "documentation": {
                "summary": "A simple Google Cloud Endpoints API example."
                },
                "apis":[
                    {
                        "name":"%s",
                        "methods":[
                            {
                                "name": "Simplegetcors"
                            }
                        ]
                    }
                ],
                "id": "%s"
            }`, testProjectName, testEndpointName, oldConfigID),
		fakeNewServiceConfig: fmt.Sprintf(`{
                "name": "%s",
                "title": "Endpoints Example",
                "documentation": {
                "summary": "A simple Google Cloud Endpoints API example."
                },
                "apis":[
                    {
                        "name":"%s",
                        "methods":[
                            {
                                "name": "Simplegetcors"
                            }
                        ]
                    }
                ],
                "id": "%s"
            }`, testProjectName, testEndpointName, newConfigID),
		BackendAddress: "grpc://127.0.0.1:80",
	}

	// Overrides fakeConfig with fakeOldServiceConfig for the test case.
	var err error

	if err = genProtoBinary(tc.fakeOldScReport, new(servicecontrolpb.ReportResponse), &fakeScReport); err != nil {
		t.Fatalf("generate fake service control report failed: %v", err)
	}

	if err = genProtoBinary(tc.fakeOldServiceRollout, new(smpb.ListServiceRolloutsResponse), &fakeRollouts); err != nil {
		t.Fatalf("generate fake service rollout failed: %v", err)
	}

	if err = genProtoBinary(tc.fakeOldServiceConfig, new(confpb.Service), &fakeConfig); err != nil {
		t.Fatalf("generate fake service config failed: %v", err)
	}

	opts := options.DefaultConfigGeneratorOptions()
	opts.BackendAddress = tc.BackendAddress

	setFlags(testProjectName, testConfigID, util.ManagedRolloutStrategy, "100ms", "")

	runTest(t, &fakeScReport, &fakeRollouts, &fakeConfig, opts, func(configManager *ConfigManager) {
		var respInterface cache.ResponseIface
		var resp cache.Response
		var err error
		ctx := context.Background()
		req := discoverypb.DiscoveryRequest{
			Node: &corepb.Node{
				Id: opts.Node,
			},
			TypeUrl: resource.ListenerType,
		}
		respInterface, err = configManager.cache.Fetch(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		resp = respInterface.(cache.Response)

		if resp.Version != oldConfigID {
			t.Errorf("Test Desc: %s, snapshot cache fetch got version: %v, want: %v", tc.desc, resp.Version, oldConfigID)
		}
		if !proto.Equal(&resp.Request, &req) {
			t.Errorf("Test Desc: %s, snapshot cache fetch got request: %v, want: %v", tc.desc, resp.Request, req)
		}

		if err = genProtoBinary(tc.fakeNewScReport, new(servicecontrolpb.ReportResponse), &fakeScReport); err != nil {
			t.Fatalf("generate fake service control report failed: %v", err)
		}

		if err = genProtoBinary(tc.fakeNewServiceRollout, new(smpb.ListServiceRolloutsResponse), &fakeRollouts); err != nil {
			t.Fatalf("generate fake service rollout failed: %v", err)
		}

		if err = genProtoBinary(tc.fakeNewServiceConfig, new(confpb.Service), &fakeConfig); err != nil {
			t.Fatalf("generate fake service config failed: %v", err)
		}

		time.Sleep(*checkNewRolloutInterval + time.Second)

		respInterface, err = configManager.cache.Fetch(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		resp = respInterface.(cache.Response)

		if resp.Version != newConfigID || configManager.curConfigId() != newConfigID {
			t.Errorf("Test Desc: %s, snapshot cache fetch got version: %v, want: %v", tc.desc, resp.Version, newConfigID)
		}

		if !proto.Equal(&resp.Request, &req) {
			t.Errorf("Test Desc: %s, snapshot cache fetch got request: %v, want: %v", tc.desc, resp.Request, req)
		}
	})
}

func runTest(t *testing.T, fakeScReport, fakeRollouts, fakeConfig *safeData, opts options.ConfigGeneratorOptions, f func(configManager *ConfigManager)) {
	fakeToken := `{"access_token": "ya29.new", "expires_in":3599, "token_type":"Bearer"}`
	mockServiceControl := initMockServer(t, fakeScReport)
	defer mockServiceControl.Close()
	util.FetchRolloutIdURL = func(serviceControlUrl, serviceName string) string {
		return mockServiceControl.URL
	}

	mockRollout := initMockServer(t, fakeRollouts)
	defer mockRollout.Close()
	util.FetchRolloutsURL = func(serviceManagementUrl, serviceName string) string {
		return mockRollout.URL
	}

	mockConfig := initMockServer(t, fakeConfig)
	defer mockConfig.Close()
	util.FetchConfigURL = func(serviceManagementUrl, serviceName, configId string) string {
		return mockConfig.URL
	}

	mockMetadataServer := util.InitMockServerFromPathResp(map[string]string{
		util.AccessTokenSuffix: fakeToken,
	})
	defer mockMetadataServer.Close()

	metadataFetcher := metadata.NewMockMetadataFetcher(mockMetadataServer.URL, time.Now())

	opts.RootCertsPath = platform.GetFilePath(platform.TestRootCaCerts)
	manager, err := NewConfigManager(metadataFetcher, opts)
	if err != nil {
		t.Fatal("fail to initialize Config Manager: ", err)
	}

	f(manager)
}

type safeData struct {
	mutex sync.Mutex
	data  []byte
}

func (s *safeData) write(src []byte) {
	s.mutex.Lock()
	s.data = make([]byte, len(src))
	copy(s.data, src)
	s.mutex.Unlock()
}

func (s *safeData) read() []byte {
	s.mutex.Lock()
	ret := make([]byte, len(s.data))
	copy(ret, s.data)
	s.mutex.Unlock()
	return ret
}

func initMockServer(t *testing.T, config *safeData) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(config.read())
		if err != nil {
			t.Fatal("fail to write config: ", err)
		}
	}))
}

func sortResources(response cache.Response) []types.Resource {
	// configManager.cache may change the order
	// sort them before comparing results.
	sortedResources := response.Resources
	sort.Slice(sortedResources, func(i, j int) bool {
		return cache.GetResourceName(sortedResources[i]) < cache.GetResourceName(sortedResources[j])
	})
	return sortedResources
}

func unmarshalJsonTestToPbMessage(input string, output proto.Message) error {
	unmarshaler := &jsonpb.Unmarshaler{
		AnyResolver: util.Resolver,
	}

	switch t := output.(type) {
	case *confpb.Service:
		if err := unmarshaler.Unmarshal(strings.NewReader(input), output.(*confpb.Service)); err != nil {
			return fmt.Errorf("fail to unmarshal %T: %v", t, err)
		}
	case *smpb.ListServiceRolloutsResponse:
		if err := unmarshaler.Unmarshal(strings.NewReader(input), output.(*smpb.ListServiceRolloutsResponse)); err != nil {
			return fmt.Errorf("fail to unmarshal %T: %v", t, err)
		}
	case *servicecontrolpb.ReportResponse:
		if err := unmarshaler.Unmarshal(strings.NewReader(input), output.(*servicecontrolpb.ReportResponse)); err != nil {
			return fmt.Errorf("fail to unmarshal %T: %v", t, err)
		}
		return nil
	default:
		return fmt.Errorf("not support unmarshalling %T", t)
	}
	return nil
}

func genProtoBinary(input string, msg proto.Message, dest *safeData) error {
	if err := unmarshalJsonTestToPbMessage(input, msg); err != nil {
		return err
	}

	protoBytesArray, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	dest.write(protoBytesArray)

	return nil
}

func setFlags(service, serviceConfigId, rolloutStrategy, checkRolloutInterval, serviceJsonPath string) {
	_ = flag.Set("service", service)
	_ = flag.Set("service_config_id", serviceConfigId)
	_ = flag.Set("rollout_strategy", rolloutStrategy)
	_ = flag.Set("check_rollout_interval", checkRolloutInterval)
	_ = flag.Set("service_json_path", serviceJsonPath)
}
