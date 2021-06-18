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
				"admin_port":         "0",
				"tracing_project_id": "test_project",
			},
			wantConfig: `
{
   "admin":{
      
   },
   "dynamicResources":{
      "adsConfig":{
         "apiType":"GRPC",
         "grpcServices":[
            {
               "envoyGrpc":{
                  "clusterName":"@espv2-ads-cluster"
               }
            }
         ],
         "transportApiVersion":"V3"
      },
      "cdsConfig":{
         "ads":{
            
         },
         "resourceApiVersion":"V3"
      },
      "ldsConfig":{
         "ads":{
            
         },
         "resourceApiVersion":"V3"
      }
   },
   "layeredRuntime":{
      "layers":[
         {
            "name": "static-runtime",
            "staticLayer": {
              "envoy.reloadable_features.preserve_downstream_scheme": false,
              "re2.max_program_size.error_level":1000
            }
         }
      ]
   },
   "node":{
      "cluster":"ESPv2_cluster",
      "id":"ESPv2"
   },
   "staticResources":{
      "clusters":[
         {
            "connectTimeout":"10s",
            "http2ProtocolOptions":{
               
            },
            "loadAssignment":{
               "clusterName":"@espv2-ads-cluster",
               "endpoints":[
                  {
                     "lbEndpoints":[
                        {
                           "endpoint":{
                              "address":{
                                 "pipe":{
                                    "path":"@espv2-ads-cluster"
                                 }
                              }
                           }
                        }
                     ]
                  }
               ]
            },
            "name":"@espv2-ads-cluster",
            "type":"STATIC"
         }
      ]
   }
}
`,
		},
		{
			desc: "bootstrap with options",
			args: map[string]string{
				// TODO(nareddyt): Remove flag from bootstrap binary in follow-up PR
				"disable_tracing": "true",
				"admin_port":      "8001",
				"node":            "test-node",
			},
			wantConfig: `
{
   "admin":{
      "accessLogPath":"/dev/null",
      "address":{
         "socketAddress":{
            "address":"0.0.0.0",
            "portValue":8001
         }
      }
   },
   "dynamicResources":{
      "adsConfig":{
         "apiType":"GRPC",
         "grpcServices":[
            {
               "envoyGrpc":{
                  "clusterName":"@espv2-ads-cluster"
               }
            }
         ],
         "transportApiVersion":"V3"
      },
      "cdsConfig":{
         "ads":{
            
         },
         "resourceApiVersion":"V3"
      },
      "ldsConfig":{
         "ads":{
            
         },
         "resourceApiVersion":"V3"
      }
   },
   "layeredRuntime":{
      "layers":[
         {
            "name": "static-runtime",
            "staticLayer": {
              "envoy.reloadable_features.preserve_downstream_scheme": false,
              "re2.max_program_size.error_level":1000
            }
         }
      ]
   },
   "node":{
      "cluster":"test-node_cluster",
      "id":"test-node"
   },
   "staticResources":{
      "clusters":[
         {
            "connectTimeout":"10s",
            "http2ProtocolOptions":{
               
            },
            "loadAssignment":{
               "clusterName":"@espv2-ads-cluster",
               "endpoints":[
                  {
                     "lbEndpoints":[
                        {
                           "endpoint":{
                              "address":{
                                 "pipe":{
                                    "path":"@espv2-ads-cluster"
                                 }
                              }
                           }
                        }
                     ]
                  }
               ]
            },
            "name":"@espv2-ads-cluster",
            "type":"STATIC"
         }
      ]
   }
}
`,
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
