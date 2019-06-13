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
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	boot "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
)

// CreateAdmin outputs Admin struct for bootstrap config
func CreateAdmin() *boot.Admin {
	return &boot.Admin{
		AccessLogPath: "/dev/null",
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: 8001,
					},
				},
			},
		},
	}
}
