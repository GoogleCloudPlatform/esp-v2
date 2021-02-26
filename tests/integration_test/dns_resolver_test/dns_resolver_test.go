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

package dns_resolver_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
)

func toFqdnWithRoot(dns string) string {
	return dns + "."
}

func TestDnsResolver(t *testing.T) {
	t.Parallel()

	testCase := []struct {
		desc            string
		backendHost     string
		isResolveFailed bool
		wantResp        string
		wantError       string
	}{
		{
			desc:        "resolve PQDN domain name successfully",
			backendHost: "dns-resolver-test-backend",
			wantResp:    `{"message":"hello"}`,
		},
		{
			desc:        "resolve FQDN domain name successfully",
			backendHost: "dns-resolver-test-backend.example.com",
			wantResp:    `{"message":"hello"}`,
		},
		{
			desc:        "resolve workstation FQDN domain name successfully",
			backendHost: "dns-resolver-test-backend.corp.google.com",
			wantResp:    `{"message":"hello"}`,
		},
		{
			desc:        "resolve k8s FQDN domain name successfully",
			backendHost: "dns-resolver-test-backend.test-pods.svc.cluster.local",
			wantResp:    `{"message":"hello"}`,
		},
		{
			desc:            "resolve domain name fails because record not exist in resolver",
			backendHost:     "dns-resolver-test-backend",
			isResolveFailed: true,
			wantError:       `503 Service Unavailable, {"message":"no healthy upstream","code":503}`,
		},
	}

	for _, tc := range testCase {
		func() {
			s := env.NewTestEnv(platform.TestDnsResolver, platform.EchoSidecar)

			// Setup dns resolver.
			dnsRecords := map[string]string{
				toFqdnWithRoot(tc.backendHost): platform.GetLoopbackAddress(),
			}
			dnsResolver := comp.NewDnsResolver(s.Ports().DnsResolverPort, dnsRecords)
			defer dnsResolver.Shutdown()
			go func() {
				if err := dnsResolver.ListenAndServe(); err != nil {
					t.Fatalf("Failed to set udp listener %s\n", err.Error())
				}
			}()

			// Check dns resolver's health.
			dnsResolverAddress := fmt.Sprintf("%v:%v", platform.GetLoopbackAddress(), s.Ports().DnsResolverPort)
			if err := comp.CheckDnsResolverHealth(dnsResolverAddress, tc.backendHost, platform.GetLoopbackAddress()); err != nil {
				t.Fatalf("DNS Resolver is not healthy: %v", err)
			}

			// If testing failure case, remove records after dns health check passes.
			if tc.isResolveFailed {
				delete(dnsRecords, toFqdnWithRoot(tc.backendHost))
				dnsRecords[toFqdnWithRoot("invalid."+tc.backendHost)] = platform.GetLoopbackAddress()
			}

			// Setup the whole test framework.
			s.SetBackendAddress(fmt.Sprintf("http://%s:%v", tc.backendHost, s.Ports().BackendServerPort))
			args := []string{"" +
				"--service_config_id=test-config-id",
				"--rollout_strategy=fixed",
				"--healthz=/healthz",
				"--dns_resolver_addresses=" + dnsResolverAddress,
			}

			defer s.TearDown(t)
			if err := s.Setup(args); err != nil {
				t.Fatalf("fail to setup test env, %v", err)
			}

			url := fmt.Sprintf("http://%v:%v/echo?key=api-key", platform.GetLoopbackAddress(), s.Ports().ListenerPort)
			resp, err := client.DoPost(url, "hello")
			if err != nil {
				if tc.wantError == "" {
					t.Errorf("Test(%v): got unexpected error: %s", tc.desc, err)
				} else if strings.Contains(err.Error(), tc.wantError) {
					t.Errorf("Test(%v): got unexpected error, expect: %s, get: %s", tc.desc, tc.wantError, err.Error())
				}
				return
			}

			if !strings.Contains(string(resp), tc.wantResp) {
				t.Errorf("Test(%v): expected: %s, got: %s", tc.desc, tc.wantResp, string(resp))
			}
		}()
	}

}
