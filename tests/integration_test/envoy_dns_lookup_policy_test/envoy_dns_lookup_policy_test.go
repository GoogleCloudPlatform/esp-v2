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

/*
The following comments will be removed once the implementation is done.

GET endpoint /ipversion
Tests:
	*"auto"/"v6preferred":
		1. dns resolver contains both IPv4 and IPv6 addresses => "IPv6"
		2. dns resolver contains IPv4 address only => "IPv4"
	*"v4only":
		1. dns resolver contains both IPv4 and IPv6 addresses => "IPv4"
		2. dns resolver contains IPv6 address only => error
	*"v6only":
		1. dns resolver contains both IPv4 and IPv6 addresses => "IPv6"
		2. dns resolver contains IPv4 address only => error
	*"v4preferred":
		1. dns resolver contains both IPv4 and IPv6 addresses => "IPv4"
		2. dns resolver contains IPv6 address only => "IPv6"
	*"all" - concurrently trying to get addresses, first-get-first-used
		???
*/

package envoy_dns_lookup_policy_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
)

const (
	domainName = "envoy-dns-lookup-policy-test-backend"
	v4Response = "IPv4"
	v6Response = "IPv6"
)

func toFqdnWithRoot(dns string) string {
	return dns + "."
}

func TestEnvoyDnsLookupPolicy(t *testing.T) {
	t.Parallel()

	testCase := []struct {
		desc            string
		dnsLookupPolicy string
		domainAddresses []string
		isIPv6Backend   bool
		wantResp        string
		wantError       string
	}{
		// test cases for dns lookup policy 'auto'
		{
			desc:            "dns resolver contains both IPv4 and IPv6 and dns lookup policy is 'auto'",
			dnsLookupPolicy: "auto",
			domainAddresses: []string{platform.GetLoopbackAddress(), platform.GetLoopbackIPv6Address()},
			isIPv6Backend:   true,
			wantResp:        v6Response,
		},
		{
			desc:            "dns resolver contains IPv4 only and dns lookup policy is 'auto'",
			dnsLookupPolicy: "auto",
			domainAddresses: []string{platform.GetLoopbackAddress()},
			isIPv6Backend:   false,
			wantResp:        v4Response,
		},
		// test cases for dns lookup policy 'v4only'
		{
			desc:            "dns resolver contains both IPv4 and IPv6 and dns lookup policy is 'v4only'",
			dnsLookupPolicy: "v4only",
			domainAddresses: []string{platform.GetLoopbackAddress(), platform.GetLoopbackIPv6Address()},
			isIPv6Backend:   false,
			wantResp:        v4Response,
		},
		{
			desc:            "dns resolver contains IPv6 only and dns lookup policy is 'v4only'",
			dnsLookupPolicy: "v4only",
			domainAddresses: []string{platform.GetLoopbackIPv6Address()},
			isIPv6Backend:   true,
			wantError:       `503 Service Unavailable, {"message":"no healthy upstream","code":503}`,
		},
		// test cases for dns lookup policy 'v6only'
		{
			desc:            "dns resolver contains both IPv4 and IPv6 and dns lookup policy is 'v6only'",
			dnsLookupPolicy: "v6only",
			domainAddresses: []string{platform.GetLoopbackAddress(), platform.GetLoopbackIPv6Address()},
			isIPv6Backend:   true,
			wantResp:        v6Response,
		},
		{
			desc:            "dns resolver contains IPv4 only and dns lookup policy is 'v6only'",
			dnsLookupPolicy: "v6only",
			domainAddresses: []string{platform.GetLoopbackAddress()},
			isIPv6Backend:   false,
			wantError:       `503 Service Unavailable, {"message":"no healthy upstream","code":503}`,
		},
		// test cases for dns lookup policy 'v4preferred'
		{
			desc:            "dns resolver contains both IPv4 and IPv6 and dns lookup policy is 'v4preferred'",
			dnsLookupPolicy: "v4preferred",
			domainAddresses: []string{platform.GetLoopbackAddress(), platform.GetLoopbackIPv6Address()},
			isIPv6Backend:   false,
			wantResp:        v4Response,
		},
		{
			desc:            "dns resolver contains IPv6 only and dns lookup policy is 'v4preferred'",
			dnsLookupPolicy: "v4preferred",
			domainAddresses: []string{platform.GetLoopbackIPv6Address()},
			isIPv6Backend:   true,
			wantResp:        v6Response,
		},
	}

	for _, tc := range testCase {
		func() {
			s := env.NewTestEnv(platform.TestEnvoyDnsLookupPolicy, platform.EchoSidecar)

			// Spin up dns resolver
			dnsRecords := map[string][]string{
				toFqdnWithRoot(domainName): tc.domainAddresses,
			}
			dnsResolver := components.NewDnsResolver(s.Ports().DnsResolverPort, dnsRecords)
			defer dnsResolver.Shutdown()
			go func() {
				if err := dnsResolver.ListenAndServe(); err != nil {
					t.Fatalf("Failed to set udp listener %s\n", err.Error())
				}
			}()
			// Check dns resolver's health.
			dnsResolverAddress := fmt.Sprintf("%v:%v", platform.GetLoopbackAddress(), s.Ports().DnsResolverPort)
			if err := components.CheckDnsResolverHealth(dnsResolverAddress, domainName, tc.domainAddresses[0]); err != nil {
				t.Fatalf("DNS Resolver is not healthy: %v", err)
			}

			// Set up the whole test framework, one echo backend will be spun up within the framework
			// The echo backend in the framework should be serving on IPv6 only when one IPv6 backend is expected to be up.
			s.SetUseIPv6Address(tc.isIPv6Backend)
			s.SetBackendAddress(fmt.Sprintf("http://%s:%v", domainName, s.Ports().BackendServerPort))
			args := []string{"" +
				"--service_config_id=test-config-id",
				"--rollout_strategy=fixed",
				"--healthz=/healthz",
				"--dns_resolver_addresses=" + dnsResolverAddress,
				"--backend_dns_lookup_family=" + tc.dnsLookupPolicy,
			}
			defer s.TearDown(t)
			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			// Make request and check response
			url := fmt.Sprintf("http://%v:%v/ipversion?key=api-key", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
			resp, err := client.DoGet(url)
			if err != nil {
				if tc.wantError == "" {
					t.Errorf("Test(%v): got unexpected error: %s", tc.desc, err)
				} else if strings.Contains(err.Error(), tc.wantError) {
					t.Errorf("Test(%v): got unexpected error, expect: %s, get: %s", tc.desc, tc.wantError, err.Error())
				}
				return
			}

			if string(resp) != tc.wantResp {
				t.Errorf("Test(%v): expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
			}
		}()
	}
}
