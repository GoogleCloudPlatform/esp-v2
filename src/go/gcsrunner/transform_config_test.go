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

package gcsrunner

import (
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"

	scpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v7/http/service_control"
)

func TestAddGCPAttributes(t *testing.T) {
	tests := []struct {
		name             string
		in               *scpb.FilterConfig
		metadataResponds map[string]string
		want             *scpb.FilterConfig
	}{
		{
			name: "success with default platform",
			in:   &scpb.FilterConfig{},
			metadataResponds: map[string]string{
				util.ProjectIDSuffix: "project",
				util.ZoneSuffix:      "projectzs/project/zone",
			},
			want: &scpb.FilterConfig{
				GcpAttributes: &scpb.GcpAttributes{
					ProjectId: "project",
					Zone:      "zone",
					Platform:  util.GCE,
				},
			},
		},
		{
			name: "success with platform override",
			in: &scpb.FilterConfig{
				GcpAttributes: &scpb.GcpAttributes{
					Platform: "override",
				},
			},
			metadataResponds: map[string]string{
				util.ProjectIDSuffix: "project",
				util.ZoneSuffix:      "projectzs/project/zone",
			},
			want: &scpb.FilterConfig{
				GcpAttributes: &scpb.GcpAttributes{
					ProjectId: "project",
					Zone:      "zone",
					Platform:  "override",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := util.InitMockServerFromPathResp(tc.metadataResponds)
			defer ts.Close()
			opts := FetchConfigOptions{
				MetadataURL: ts.URL,
			}
			if err := addGCPAttributes(tc.in, opts); err != nil {
				t.Fatalf("addGCPAttributes(%v,%v) returned error %v", tc.in, opts, err)
			}
			if diff := cmp.Diff(tc.want, tc.in, cmp.Comparer(proto.Equal)); diff != "" {
				t.Errorf("addGCPAttributes returned unexpected result: (-want/+got): %s", diff)
			}
		})
	}
}

func TestTransformConfigBytes(t *testing.T) {
	testCases := []struct {
		name      string
		config    []byte
		wantError bool
	}{
		{
			name:   "Valid config",
			config: validConfigInput,
		},
		{
			name:      "Invalid config missing ingress_listener",
			config:    invalidConfigInput,
			wantError: true,
		},
	}

	for _, tc := range testCases {
		doServiceControlCalled := false
		doServiceControlTransform = func(_ *scpb.FilterConfig, _ FetchConfigOptions) error {
			doServiceControlCalled = true
			return nil
		}
		opts := FetchConfigOptions{}
		_, err := transformConfigBytes(tc.config, opts)
		if tc.wantError {
			if err == nil {
				t.Fatal("transformConfigBytes() returned nil, want an error", err)
			}
		} else {
			if err != nil {
				t.Fatalf("transformConfigBytes() returned %v, want nil", err)
			}
			if !doServiceControlCalled {
				t.Errorf("doServiceControlTransform was not called")
			}
		}
	}
}

var validConfigInput = []byte(`{
"static_resources": {
    "listeners": [
      {
      "name": "ingress_listener",
        "address": {
          "socket_address": {
            "port_value": 1111
          }
        },
        "filter_chains": [
          {
            "filters": [
              {
                "name": "envoy.filters.network.http_connection_manager",
                "typed_config": {
                  "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
                  "http_filters": [
                    {
                      "name": "com.google.espv2.filters.http.service_control",
                      "typed_config": {
                        "@type": "type.googleapis.com/espv2.api.envoy.v7.http.service_control.FilterConfig"
                      }
                    }
                  ]
                }
              }
            ]
          }
        ]
      },
      {
        "name": "loopback_listener",
        "address": {
          "socket_address": {
            "port_value": 2222
          }
        }
      }
    ]
  }
}`)

var invalidConfigInput = []byte(`{
"static_resources": {
    "listeners": [
      {
        "name": "loopback_listener",
        "address": {
          "socket_address": {
            "port_value": 2222
          }
        }
      }
    ]
  }
}`)
