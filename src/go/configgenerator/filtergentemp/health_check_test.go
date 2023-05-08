// Copyright 2023 Google LLC
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

package filtergentemp_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergentemp"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestNewHealthCheckFilterGensFromOPConfig_GenConfig(t *testing.T) {
	testdata := []SuccessOPTestCase{
		{
			Desc: "Success, generate health check filter for standard /healthz path",
			OptsIn: options.ConfigGeneratorOptions{
				Healthz: "/healthz",
			},
			WantFilterConfigs: []string{
				`{
        "name": "envoy.filters.http.health_check",
        "typedConfig": {
          "@type":"type.googleapis.com/envoy.extensions.filters.http.health_check.v3.HealthCheck",
          "passThroughMode":false,
          "headers": [
            {
              "stringMatch":{"exact":"/healthz"},
              "name":":path"
            }
          ]
        }
      }`,
			},
		},
		{
			Desc: "Success, generate health check filter where / prefix is automatically added",
			OptsIn: options.ConfigGeneratorOptions{
				Healthz: "healthz",
			},
			WantFilterConfigs: []string{
				`{
        "name": "envoy.filters.http.health_check",
        "typedConfig": {
          "@type":"type.googleapis.com/envoy.extensions.filters.http.health_check.v3.HealthCheck",
          "passThroughMode":false,
          "headers": [
            {
              "stringMatch":{"exact":"/healthz"},
              "name":":path"
            }
          ]
        }
      }`,
			},
		},
		{
			Desc: "Success, generate health check filter for gRPC with health check",
			OptsIn: options.ConfigGeneratorOptions{
				Healthz:                "healthz",
				HealthCheckGrpcBackend: true,
			},
			ServiceConfigIn: &confpb.Service{
				Name: "bookstore.endpoints.project123.cloud.goog",
			},
			WantFilterConfigs: []string{
				`{
        "name": "envoy.filters.http.health_check",
        "typedConfig": {
          "@type":"type.googleapis.com/envoy.extensions.filters.http.health_check.v3.HealthCheck",
          "passThroughMode":false,
          "headers": [
            {
              "stringMatch":{"exact":"/healthz"},
              "name":":path"
            }
          ],
          "clusterMinHealthyPercentages": {
              "backend-cluster-bookstore.endpoints.project123.cloud.goog_local": { "value": 100.0 }
          }
        }
      }`,
			},
		},
		{
			Desc: "Success, generate health check filter for root level",
			OptsIn: options.ConfigGeneratorOptions{
				Healthz: "/",
			},
			WantFilterConfigs: []string{
				`{
        "name": "envoy.filters.http.health_check",
        "typedConfig": {
          "@type":"type.googleapis.com/envoy.extensions.filters.http.health_check.v3.HealthCheck",
          "passThroughMode":false,
          "headers": [
            {
              "stringMatch":{"exact":"/"},
              "name":":path"
            }
          ]
        }
      }`,
			},
		},
	}

	for _, tc := range testdata {
		tc.RunTest(t, filtergentemp.NewHealthCheckFilterGensFromOPConfig)
	}
}
