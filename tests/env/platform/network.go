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

package platform

// Returns the loopback address in a form that can be used with URLs.
func GetLoopbackAddress() string {
	return "127.0.0.1"
}

// Returns the loopback IPv6 address. Need to embrace it with brackets to be used with URLs, e.g. [::1].
func GetLoopbackIPv6Address() string {
	return "::1"
}

// Returns the loopback hostname. Not safe to use in URLs.
func GetLoopbackHost() string {
	return "127.0.0.1"
}

// Returns the meta-address that resolves to any address.
func GetAnyAddress() string {
	return "0.0.0.0"
}

// Returns the DNS family that should be resolved.
func GetDnsFamily() string {
	return "v4only"
}

// Returns the IP protocol these addresses are for.
func GetIpProtocol() string {
	return "ipv4"
}

// Returns the network protocol, accepted by `net.Listen`.
func GetNetworkProtocol() string {
	return "tcp4"
}

// Returns localhost. Encouraged not to use this, as it doesn't provide guarantees on ipv4 vs ipv6 addresses.
func GetLocalhost() string {
	return "localhost"
}
