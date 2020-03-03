// Copyright 2019 Google LLC
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

package ads

import (
	"flag"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/bootstrap/ads/flags"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
)

func TestCreateBootstrapConfig(t *testing.T) {

	testData := []struct {
		desc       string
		args       map[string]string
		wantConfig string
	}{
		{
			desc: "bootstrap with default options",
			args: map[string]string{
				"tracing_project_id": "test_project",
			},
			wantConfig: `{
			  "node": {
          "id": "ESPv2",
          "cluster": "ESPv2_cluster"
        },
        "staticResources": {
          "clusters": [
            {
              "name": "ads_cluster",
              "type": "STRICT_DNS",
              "connectTimeout": "10s",
              "loadAssignment": {
                "clusterName": "127.0.0.1",
                "endpoints": [
                  {
                    "lbEndpoints": [
                      {
                        "endpoint": {
                          "address": {
                            "socketAddress": {
                              "address": "127.0.0.1",
                              "portValue": 8790
                            }
                          }
                        }
                      }
                    ]
                  }
                ]
              },
              "http2ProtocolOptions": {
              }
            }
          ]
        },
        "dynamicResources": {
          "ldsConfig": {
            "ads": {
            }
          },
          "cdsConfig": {
            "ads": {
            }
          },
          "adsConfig": {
            "apiType": "GRPC",
            "grpcServices": [
              {
                "envoyGrpc": {
                  "clusterName": "ads_cluster"
                }
              }
            ]
          }
        },
        "tracing": {
          "http": {
            "name": "envoy.tracers.opencensus",
            "typedConfig": {
              "@type": "type.googleapis.com/envoy.config.trace.v2.OpenCensusConfig",
              "traceConfig": {
                "probabilitySampler": {
                  "samplingProbability": 0.001
                },
                "maxNumberOfAttributes": "32",
                "maxNumberOfAnnotations": "32",
                "maxNumberOfMessageEvents": "128",
                "maxNumberOfLinks": "128"
              },
              "stackdriverExporterEnabled": true,
              "stackdriverProjectId": "test_project"
            }
          }
        },
        "admin": {}
      }`,
		},
		{
			desc: "bootstrap with options",
			args: map[string]string{
				"disable_tracing": "true",
				"enable_admin":    "true",
				"node":            "test-node",
			},
			wantConfig: `{
			  "node": {
          "id": "test-node",
          "cluster": "test-node_cluster"
        },
        "staticResources": {
          "clusters": [
            {
              "name": "ads_cluster",
              "type": "STRICT_DNS",
              "connectTimeout": "10s",
              "loadAssignment": {
                "clusterName": "127.0.0.1",
                "endpoints": [
                  {
                    "lbEndpoints": [
                      {
                        "endpoint": {
                          "address": {
                            "socketAddress": {
                              "address": "127.0.0.1",
                              "portValue": 8790
                            }
                          }
                        }
                      }
                    ]
                  }
                ]
              },
              "http2ProtocolOptions": {
              }
            }
          ]
        },
        "dynamicResources": {
          "ldsConfig": {
            "ads": {
            }
          },
          "cdsConfig": {
            "ads": {
            }
          },
          "adsConfig": {
            "apiType": "GRPC",
            "grpcServices": [
              {
                "envoyGrpc": {
                  "clusterName": "ads_cluster"
                }
              }
            ]
          }
        },
        "admin": {
          "accessLogPath": "/dev/null",
          "address": {
            "socketAddress": {
              "address": "0.0.0.0",
              "portValue": 8001
            }
          }
        }
      }`,
		},
	}

	for _, tc := range testData {
		for key, value := range tc.args {
			flag.Set(key, value)
		}
		opts := flags.DefaultBootstrapperOptionsFromFlags()
		bootstrapStr, err := CreateBootstrapConfig(opts)
		if err != nil {
			t.Fatalf("failed to create bootstrap config, error: %v", err)
		}
		if err := util.JsonEqual(tc.wantConfig, bootstrapStr); err != nil {
			t.Errorf("Test (%s) failed:\n %v", tc.desc, err)
		}
	}
}
