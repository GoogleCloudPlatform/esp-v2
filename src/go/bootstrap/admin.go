// Copyright 2019 Google Cloud Platform Proxy Authors
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
	addresspb "github.com/envoyproxy/data-plane-api/api/address"
	bootstrappb "github.com/envoyproxy/data-plane-api/api/bootstrap"
)

// CreateAdmin outputs Admin struct for bootstrap config
func CreateAdmin(adminPort uint32) *bootstrappb.Admin {
	return &bootstrappb.Admin{
		AccessLogPath: "/dev/null",
		Address: &addresspb.Address{
			Address: &addresspb.Address_SocketAddress{
				SocketAddress: &addresspb.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &addresspb.SocketAddress_PortValue{
						PortValue: adminPort,
					},
				},
			},
		},
	}
}
