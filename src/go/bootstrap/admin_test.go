// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bootstrap

import (
	"testing"

	"github.com/GoogleCloudPlatform/api-proxy/src/go/options"
	"github.com/golang/protobuf/proto"

	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	bootstrappb "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
)

func TestCreateAdmin(t *testing.T) {
	testData := []struct {
		desc        string
		enableAdmin bool
		want        *bootstrappb.Admin
	}{
		{
			desc:        "Admin interface is disabled",
			enableAdmin: false,
			want:        &bootstrappb.Admin{},
		},
		{
			desc:        "Admin interface is enabled, created with default values",
			enableAdmin: true,
			want: &bootstrappb.Admin{
				AccessLogPath: "/dev/null",
				Address: &corepb.Address{
					Address: &corepb.Address_SocketAddress{
						SocketAddress: &corepb.SocketAddress{
							Address: "0.0.0.0",
							PortSpecifier: &corepb.SocketAddress_PortValue{
								PortValue: 8001,
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testData {

		opts := options.DefaultCommonOptions()
		opts.EnableAdmin = tc.enableAdmin

		got := CreateAdmin(opts)

		if !proto.Equal(got, tc.want) {
			t.Errorf("Test (%s): failed, got: %v, want: %v", tc.desc, got, tc.want)
		}

	}
}
