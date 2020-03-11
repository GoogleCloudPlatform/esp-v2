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

package bootstrap

import (
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"

	corepb "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	bootstrappb "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
)

// CreateAdmin outputs Admin struct for bootstrap config
func CreateAdmin(opts options.CommonOptions) *bootstrappb.Admin {

	if opts.AdminPort == 0 {
		return &bootstrappb.Admin{}
	}

	return &bootstrappb.Admin{
		AccessLogPath: "/dev/null",
		Address: &corepb.Address{
			Address: &corepb.Address_SocketAddress{
				SocketAddress: &corepb.SocketAddress{
					Address: opts.AdminAddress,
					PortSpecifier: &corepb.SocketAddress_PortValue{
						PortValue: uint32(opts.AdminPort),
					},
				},
			},
		},
	}
}
