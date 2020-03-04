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

	scpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/service_control"
	v2pb "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
)

func TestAddGCPAttributes(t *testing.T) {
	tests := []struct {
		name             string
		opts             FetchConfigOptions
		metadataResponds map[string]string
		want             *scpb.FilterConfig
	}{
		{
			name: "success with default platform",
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
			opts: FetchConfigOptions{
				OverridePlatform: "override",
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
			tc.opts.MetadataURL = ts.URL

			got := &scpb.FilterConfig{}
			if err := addGCPAttributes(got, tc.opts); err != nil {
				t.Fatalf("addGCPAttributes(%v,%v) returned error %v", got, tc.opts, err)
			}
			if diff := cmp.Diff(tc.want, got, cmp.Comparer(proto.Equal)); diff != "" {
				t.Errorf("addGCPAttributes returned unexpected result: (-want/+got): %s", diff)
			}
		})
	}
}

func TestReplaceListenerPort(t *testing.T) {
	testCases := []struct {
		name                   string
		listener, wantListener *v2pb.Listener
		opts                   FetchConfigOptions
		wantError              bool
	}{
		{
			name: "successful replace",
			opts: FetchConfigOptions{
				ReplacePort: 1234,
				WantPort:    5678,
			},
			listener: &v2pb.Listener{
				Address: &corepb.Address{
					Address: &corepb.Address_SocketAddress{
						SocketAddress: &corepb.SocketAddress{
							PortSpecifier: &corepb.SocketAddress_PortValue{
								PortValue: 1234,
							},
						},
					},
				},
			},
			wantListener: &v2pb.Listener{
				Address: &corepb.Address{
					Address: &corepb.Address_SocketAddress{
						SocketAddress: &corepb.SocketAddress{
							PortSpecifier: &corepb.SocketAddress_PortValue{
								PortValue: 5678,
							},
						},
					},
				},
			},
		},
		{
			name: "unset WantPort does not replace port",
			opts: FetchConfigOptions{
				ReplacePort: 1234,
			},
			listener: &v2pb.Listener{
				Address: &corepb.Address{
					Address: &corepb.Address_SocketAddress{
						SocketAddress: &corepb.SocketAddress{
							PortSpecifier: &corepb.SocketAddress_PortValue{
								PortValue: 1234,
							},
						},
					},
				},
			},
			wantListener: &v2pb.Listener{
				Address: &corepb.Address{
					Address: &corepb.Address_SocketAddress{
						SocketAddress: &corepb.SocketAddress{
							PortSpecifier: &corepb.SocketAddress_PortValue{
								PortValue: 1234,
							},
						},
					},
				},
			},
		},
		{
			name: "Port != ReplacePort should return an error",
			opts: FetchConfigOptions{
				ReplacePort: 1234,
				WantPort:    5678,
			},
			wantError: true,
			listener: &v2pb.Listener{
				Address: &corepb.Address{
					Address: &corepb.Address_SocketAddress{
						SocketAddress: &corepb.SocketAddress{
							PortSpecifier: &corepb.SocketAddress_PortValue{
								PortValue: 9999,
							},
						},
					},
				},
			},
		},
		{
			name: "Invalid config should return an error",
			opts: FetchConfigOptions{
				ReplacePort: 1234,
				WantPort:    5678,
			},
			wantError: true,
			listener: &v2pb.Listener{
				Address: &corepb.Address{
					Address: &corepb.Address_Pipe{
						Pipe: &corepb.Pipe{},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		err := replaceListenerPort(tc.listener, tc.opts)
		if (err != nil) != tc.wantError {
			t.Errorf("%s: replaceListenerPort(%v,%v) returned error %v, want err!=nil to be %v", tc.name, tc.listener, tc.opts, err, tc.wantError)
		}

		if err == nil {
			if diff := cmp.Diff(tc.wantListener, tc.listener, cmp.Comparer(proto.Equal)); diff != "" {
				t.Errorf("%s: replaceListenerPort returned unexpected result: (-want/+got): %s", tc.name, diff)
			}
		}
	}
}

func TestTransformConfigBytes(t *testing.T) {
	doListenerCalled := false
	doServiceControlCalled := false
	doListenerTransform = func(_ *v2pb.Listener, _ FetchConfigOptions) error {
		doListenerCalled = true
		return nil
	}
	doServiceControlTransform = func(_ *scpb.FilterConfig, _ FetchConfigOptions) error {
		doServiceControlCalled = true
		return nil
	}

	_, err := transformConfigBytes(validConfigInput, FetchConfigOptions{})
	if err != nil {
		t.Fatalf("transformConfigBytes() returned %v, want nil", err)
	}

	if !doListenerCalled {
		t.Errorf("doListenerTransform was not called")
	}
	if !doServiceControlCalled {
		t.Errorf("doServiceControlTransform was not called")
	}
}

var validConfigInput = []byte(`{
	"static_resources": {
    "listeners": [
      {
        "address": {
          "socket_address": {
            "port_value": 1234
          }
        },
        "filter_chains": [
          {
            "filters": [
              {
                "name": "envoy.http_connection_manager",
                "typed_config": {
                  "@type": "type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager",
                  "http_filters": [
                    {
                      "name": "envoy.filters.http.service_control",
                      "typed_config": {
												"@type": "type.googleapis.com/google.api.envoy.http.service_control.FilterConfig"
                      }
                    }
                  ]
                }
              }
            ]
          }
        ]
      }
    ]
  }
}`)
