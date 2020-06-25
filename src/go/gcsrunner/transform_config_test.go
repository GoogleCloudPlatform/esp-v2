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
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"

	scpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v6/http/service_control"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerpb "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
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

func TestReplaceListenerPort(t *testing.T) {
	testCases := []struct {
		name                   string
		listener, wantListener *listenerpb.Listener
		wantPort               uint32
		wantError              bool
	}{
		{
			name:     "successful replace",
			wantPort: 5678,
			listener: &listenerpb.Listener{
				Name: util.IngressListenerName,
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
			wantListener: &listenerpb.Listener{
				Name: util.IngressListenerName,
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
			listener: &listenerpb.Listener{
				Name: util.IngressListenerName,
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
			wantListener: &listenerpb.Listener{
				Name: util.IngressListenerName,
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
			name:      "Invalid config should return an error",
			wantPort:  5678,
			wantError: true,
			listener: &listenerpb.Listener{
				Name: util.IngressListenerName,
				Address: &corepb.Address{
					Address: &corepb.Address_Pipe{
						Pipe: &corepb.Pipe{},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		err := replaceListenerPort(tc.listener, tc.wantPort)
		if (err != nil) != tc.wantError {
			t.Errorf("%s: replaceListenerPort(%v,%v) returned error %v, want err!=nil to be %v", tc.name, tc.listener, tc.wantPort, err, tc.wantError)
		}

		if err == nil {
			if diff := cmp.Diff(tc.wantListener, tc.listener, cmp.Comparer(proto.Equal)); diff != "" {
				t.Errorf("%s: replaceListenerPort returned unexpected result: (-want/+got): %s", tc.name, diff)
			}
		}
	}
}

func TestTransformConfigBytes(t *testing.T) {
	opts := FetchConfigOptions{
		WantPort:     1234,
		LoopbackPort: 5678,
	}

	testCases := []struct {
		name            string
		config          []byte
		requireLoopback bool
	}{
		{
			name:            "Valid config",
			config:          validConfigInput,
			requireLoopback: true,
		},
		{
			name:            "Valid config with old name (http_listener)",
			config:          validConfigInputHTTPListener,
			requireLoopback: true,
		},
		{
			name:            "Valid config with old name (https_listener)",
			config:          validConfigInputHTTPSListener,
			requireLoopback: true,
		},
		{
			name:   "Valid config without Loopback",
			config: validConfigInputWithoutLoopback,
		},
	}

	for _, tc := range testCases {
		doListenerCalledOnIngress := false
		doListenerCalledOnLoopback := false
		doServiceControlCalled := false
		doListenerTransform = func(_ *listenerpb.Listener, port uint32) error {
			switch port {
			case opts.WantPort:
				doListenerCalledOnIngress = true
				return nil
			case opts.LoopbackPort:
				doListenerCalledOnLoopback = true
				return nil
			default:
				return fmt.Errorf("wrong parameters: doListenerTransform(%d), want %d or %d", port, opts.WantPort, opts.LoopbackPort)
			}
		}
		doServiceControlTransform = func(_ *scpb.FilterConfig, _ FetchConfigOptions) error {
			doServiceControlCalled = true
			return nil
		}
		_, err := transformConfigBytes(tc.config, opts)
		if err != nil {
			t.Fatalf("transformConfigBytes() returned %v, want nil", err)
		}
		if !doListenerCalledOnIngress {
			t.Errorf("doListenerTransform was not called for ingress_listener")
		}
		if !doListenerCalledOnLoopback && tc.requireLoopback {
			t.Errorf("doListenerTransform was not called for loopback_listener")
		}
		if !doServiceControlCalled {
			t.Errorf("doServiceControlTransform was not called")
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
                        "@type": "type.googleapis.com/espv2.api.envoy.v6.http.service_control.FilterConfig"
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

var validConfigInputWithoutLoopback = []byte(`{
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
                        "@type": "type.googleapis.com/espv2.api.envoy.v6.http.service_control.FilterConfig"
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

// This config has a listener name which old configs will be using.
var validConfigInputHTTPListener = []byte(`{
"static_resources": {
    "listeners": [
      {
      "name": "http_listener",
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
                        "@type": "type.googleapis.com/espv2.api.envoy.v6.http.service_control.FilterConfig"
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

// This config has a listener name which old configs will be using.
var validConfigInputHTTPSListener = []byte(`{
"static_resources": {
    "listeners": [
      {
      "name": "https_listener",
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
                        "@type": "type.googleapis.com/espv2.api.envoy.v6.http.service_control.FilterConfig"
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
