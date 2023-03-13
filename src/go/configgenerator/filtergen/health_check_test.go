package filtergen

import (
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/jsonpb"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	apipb "google.golang.org/genproto/protobuf/api"
)

func TestHealthCheckFilter(t *testing.T) {
	testdata := []struct {
		desc                   string
		BackendAddress         string
		healthz                string
		healthCheckGrpcBackend bool
		wantHealthCheckFilter  string
	}{
		{
			desc:           "Success, generate health check filter for gRPC",
			BackendAddress: "grpc://127.0.0.1:80",
			healthz:        "healthz",
			wantHealthCheckFilter: `{
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
		{
			desc:                   "Success, generate health check filter for gRPC with health check",
			BackendAddress:         "grpc://127.0.0.1:80",
			healthz:                "healthz",
			healthCheckGrpcBackend: true,
			wantHealthCheckFilter: `{
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
		{
			desc:           "Success, generate health check filter for http",
			BackendAddress: "http://127.0.0.1:80",
			healthz:        "/",
			wantHealthCheckFilter: `{
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
	}

	fakeServiceConfig := &confpb.Service{
		Name: testProjectName,
		Apis: []*apipb.Api{
			{
				Name: "endpoints.examples.bookstore.Bookstore",
				Methods: []*apipb.Method{
					{
						Name: "CreateShelf",
					},
				},
			},
		},
	}

	for _, tc := range testdata {
		t.Run(tc.desc, func(t *testing.T) {
			opts := options.DefaultConfigGeneratorOptions()
			opts.BackendAddress = tc.BackendAddress
			opts.Healthz = tc.healthz
			opts.HealthCheckGrpcBackend = tc.healthCheckGrpcBackend
			fakeServiceInfo, err := configinfo.NewServiceInfoFromServiceConfig(fakeServiceConfig, testConfigID, opts)
			if err != nil {
				t.Fatal(err)
			}

			gen := &HealthCheckGenerator{}
			filter, _, err := gen.GenFilterConfig(fakeServiceInfo)
			if err != nil {
				t.Fatal(err)
			}

			marshaler := &jsonpb.Marshaler{}
			gotFilter, err := marshaler.MarshalToString(filter)
			if err != nil {
				t.Fatal(err)
			}

			if err := util.JsonEqual(tc.wantHealthCheckFilter, gotFilter); err != nil {
				t.Errorf("GenFilterConfig has JSON diff\n%v", err)
			}
		})
	}
}
