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
	"github.com/GoogleCloudPlatform/esp-v2/src/go/serviceconfig"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discoverypb "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	servicecontrolpb "google.golang.org/genproto/googleapis/api/servicecontrol/v1"
	smpb "google.golang.org/genproto/googleapis/api/servicemanagement/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
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

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			// Overrides fakeConfig for the test case.
			if err := genProtoBinary(tc.fakeServiceConfig, new(confpb.Service), &fakeConfig); err != nil {
				t.Fatalf("generate fake service config failed: %v", err)
			}

			opts := options.DefaultConfigGeneratorOptions()
			opts.BackendAddress = tc.BackendAddress
			opts.DisableTracing = !tc.enableTracing
			opts.TracingProjectId = "fake-project-id"
			opts.SuppressEnvoyHeaders = !tc.enableDebug

			setFlags(testdata.TestFetchListenersProjectName, testdata.TestFetchListenersConfigID, util.FixedRolloutStrategy, "100ms", "")

			runTest(t, &fakeScReport, &fakeRollouts, &fakeConfig, opts, func(configManager *ConfigManager, err error) {
				if err != nil {
					t.Fatal(err)
				}

				req, resp, gotListeners, err := getListeners(configManager, opts)
				if err != nil {
					t.Fatal(err)
				}

				version, err := resp.GetVersion()
				if err != nil {
					t.Fatal(err)
				}
				if version != testdata.TestFetchListenersConfigID {
					t.Fatalf("snapshot cache fetch got version: %v, want: %v", version, testdata.TestFetchListenersConfigID)
				}
				if !proto.Equal(resp.GetRequest(), req) {
					t.Fatalf("snapshot cache fetch got request: %v, want: %v", resp.GetRequest(), req)
				}

				if err := util.JsonEqual(tc.wantedListeners, gotListeners); err != nil {
					t.Fatalf("snapshot cache fetch got unexpected Listeners, %v", err)
				}
			})
		})
	}
}

func TestRetryCallServiceManagement(t *testing.T) {
	var fakeConfig, fakeScReport, fakeRollouts safeData
	fakeServiceConfig := testdata.FakeServiceConfigForGrpcWithTranscoding
	wantedListeners := testdata.WantedListsenerForGrpcWithTranscoding
	if err := genProtoBinary(fakeServiceConfig, new(confpb.Service), &fakeConfig); err != nil {
		t.Fatalf("generate fake service config failed: %v", err)
	}

	opts := options.DefaultConfigGeneratorOptions()
	opts.BackendAddress = "grpc://127.0.0.1:80"
	opts.TracingProjectId = "fake-project-id"
	opts.DisableTracing = true

	var originalInitMockServer = initMockServer
	defer func() { initMockServer = originalInitMockServer }()

	// The mock server will reject the first 3 requests with 429 and it has a
	//silent interval, during which it will reject incoming requests with 500.
	initMockServer = func(t *testing.T, config *safeData) *httptest.Server {
		rejectWith429Times := 3
		rejectCnt := 0
		var lastCallTime time.Time
		silentInterval := time.Millisecond * 150
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rejectCnt < rejectWith429Times {
				rejectCnt += 1
				w.WriteHeader(http.StatusTooManyRequests)
				lastCallTime = time.Now()
				return
			}

			if !lastCallTime.IsZero() && lastCallTime.Add(silentInterval).After(time.Now()) {
				w.WriteHeader(http.StatusInternalServerError)
				rejectCnt += 1
				lastCallTime = time.Now()
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write(config.read())
			if err != nil {
				t.Fatal("fail to write config: ", err)
			}
			lastCallTime = time.Now()
		}))
	}

	var testData = []struct {
		desc          string
		retryNum      int
		retryInterval int
		retryConfigs  map[int]util.RetryConfig
		wantedError   string
	}{
		{
			desc: "fail, retryInterval is too short",
			retryConfigs: map[int]util.RetryConfig{
				http.StatusTooManyRequests: {
					RetryNum:      3,
					RetryInterval: time.Millisecond * 100,
				},
			},
			wantedError: "fail to fetch and apply the startup service config",
		},
		{
			desc: "fail, insufficient retryNum",
			retryConfigs: map[int]util.RetryConfig{
				http.StatusTooManyRequests: {
					RetryNum:      2,
					RetryInterval: time.Millisecond * 200,
				},
			},
			wantedError: "fail to fetch and apply the startup service config",
		},
		{
			desc: "Success, sufficient retryNum and long enough retryInterval",
			retryConfigs: map[int]util.RetryConfig{
				http.StatusTooManyRequests: {
					RetryNum:      3,
					RetryInterval: time.Millisecond * 200,
				},
			},
		},
	}
	for _, tc := range testData {
		serviceconfig.SmRetryConfigs = tc.retryConfigs

		setFlags(testdata.TestFetchListenersProjectName, testdata.TestFetchListenersConfigID, util.FixedRolloutStrategy, "100ms", "")

		runTest(t, &fakeScReport, &fakeRollouts, &fakeConfig, opts, func(configManager *ConfigManager, err error) {
			if tc.wantedError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantedError) {
					t.Errorf("test(%s) expected error: %v, got error: %v", tc.desc, tc.wantedError, err)
				}
			} else if err != nil {
				t.Errorf("test(%s) expected error: %v, got error: %v", tc.desc, tc.wantedError, err)
			}

			_, _, gotListeners, err := getListeners(configManager, opts)

			if tc.wantedError == "" {
				if err != nil {
					t.Errorf("test(%s) fail to get listener config from configmanager, error: %v", tc.desc, err)
				} else if err := util.JsonEqual(wantedListeners, gotListeners); err != nil {
					t.Errorf("Test Desc: %s, snapshot cache fetch got unexpected Listeners, %v", tc.desc, err)
				}
			}
		})
	}
}

func getListeners(configManager *ConfigManager, opts options.ConfigGeneratorOptions) (*cache.Request, cache.Response, string, error) {
	if configManager == nil {
		return nil, nil, "", fmt.Errorf("configmanager is empty")
	}

	ctx := context.Background()
	// First request, VersionId should be empty.
	req := &discoverypb.DiscoveryRequest{
		Node: &corepb.Node{
			Id: opts.Node,
		},
		TypeUrl: resource.ListenerType,
	}
	respInterface, err := configManager.cache.Fetch(ctx, req)
	if err != nil {
		return nil, nil, "", err
	}
	resp, err := respInterface.GetDiscoveryResponse()
	if err != nil {
		return nil, nil, "", err
	}

	gotListeners, err := util.ProtoToJson(resp.Resources[0])
	return req, respInterface, gotListeners, err
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
		reqForClusters := &discoverypb.DiscoveryRequest{
			Node: &corepb.Node{
				Id: opts.Node,
			},
			TypeUrl: resource.ClusterType,
		}

		respInterface, err := manager.cache.Fetch(ctx, reqForClusters)
		if err != nil {
			t.Fatal(err)
		}

		if !proto.Equal(respInterface.GetRequest(), reqForClusters) {
			t.Errorf("Test Desc(%d): %s, snapshot cache fetch got request: %v, want: %v", i, tc.desc, respInterface.GetRequest(), reqForClusters)
			continue
		}

		resp, err := respInterface.GetDiscoveryResponse()
		if err != nil {
			t.Fatal(err)
		}
		sortedClusters := sortClusters(resp.Resources)

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

		reqForListener := &discoverypb.DiscoveryRequest{
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
		version, err := respInterface.GetVersion()
		if err != nil {
			t.Fatal(err)
			continue
		}
		resp, err = respInterface.GetDiscoveryResponse()
		if err != nil {
			t.Fatal(err)
			continue
		}

		if version != testdata.TestFetchListenersConfigID {
			t.Errorf("Test Desc(%d): %s, snapshot cache fetch got version: %v, want: %v", i, tc.desc, version, testdata.TestFetchListenersConfigID)
			continue
		}
		if !proto.Equal(respInterface.GetRequest(), reqForListener) {
			t.Errorf("Test Desc(%d): %s, snapshot cache fetch got request: %v, want: %v", i, tc.desc, respInterface.GetRequest(), reqForListener)
			continue
		}

		gotListener, err := marshaler.MarshalToString(resp.Resources[0])
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
	opts.DisableTracing = true

	setFlags(testProjectName, testConfigID, util.ManagedRolloutStrategy, "100ms", "")

	runTest(t, &fakeScReport, &fakeRollouts, &fakeConfig, opts, func(configManager *ConfigManager, err error) {
		if err != nil {
			t.Fatal(err)
		}
		var respInterface cache.Response
		ctx := context.Background()
		req := &discoverypb.DiscoveryRequest{
			Node: &corepb.Node{
				Id: opts.Node,
			},
			TypeUrl: resource.ListenerType,
		}
		respInterface, err = configManager.cache.Fetch(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		version, err := respInterface.GetVersion()
		if err != nil {
			t.Fatal(err)
		}

		if version != oldConfigID {
			t.Errorf("Test Desc: %s, snapshot cache fetch got version: %v, want: %v", tc.desc, version, oldConfigID)
		}
		if !proto.Equal(respInterface.GetRequest(), req) {
			t.Errorf("Test Desc: %s, snapshot cache fetch got request: %v, want: %v", tc.desc, respInterface.GetRequest(), req)
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
		version, err = respInterface.GetVersion()
		if err != nil {
			t.Fatal(err)
		}

		if version != newConfigID || configManager.curConfigId() != newConfigID {
			t.Errorf("Test Desc: %s, snapshot cache fetch got version: %v, want: %v", tc.desc, version, newConfigID)
		}

		if !proto.Equal(respInterface.GetRequest(), req) {
			t.Errorf("Test Desc: %s, snapshot cache fetch got request: %v, want: %v", tc.desc, respInterface.GetRequest(), req)
		}
	})
}

func runTest(t *testing.T, fakeScReport, fakeRollouts, fakeConfig *safeData, opts options.ConfigGeneratorOptions, f func(configManager *ConfigManager, err error)) {
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
		util.AccessTokenPath: fakeToken,
	})
	defer mockMetadataServer.Close()

	metadataFetcher := metadata.NewMockMetadataFetcher(mockMetadataServer.URL, time.Now())

	opts.SslSidestreamClientRootCertsPath = platform.GetFilePath(platform.TestRootCaCerts)

	f(NewConfigManager(metadataFetcher, opts))
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

var initMockServer = func(t *testing.T, config *safeData) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(config.read())
		if err != nil {
			t.Fatal("fail to write config: ", err)
		}
	}))
}

func getClusterName(a *anypb.Any) string {
	c := &clusterpb.Cluster{}
	ptypes.UnmarshalAny(a, c)
	return c.GetName()
}

func sortClusters(s []*anypb.Any) []*anypb.Any {
	sort.Slice(s, func(i, j int) bool {
		return getClusterName(s[i]) < getClusterName(s[j])
	})
	return s
}

func unmarshalJsonTestToPbMessage(input string, output proto.Message) error {
	switch t := output.(type) {
	case *confpb.Service:
		if err := protojson.Unmarshal([]byte(input), output.(*confpb.Service)); err != nil {
			return fmt.Errorf("fail to unmarshal %T: %v", t, err)
		}
	case *smpb.ListServiceRolloutsResponse:
		if err := protojson.Unmarshal([]byte(input), output.(*smpb.ListServiceRolloutsResponse)); err != nil {
			return fmt.Errorf("fail to unmarshal %T: %v", t, err)
		}
	case *servicecontrolpb.ReportResponse:
		if err := protojson.Unmarshal([]byte(input), output.(*servicecontrolpb.ReportResponse)); err != nil {
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
