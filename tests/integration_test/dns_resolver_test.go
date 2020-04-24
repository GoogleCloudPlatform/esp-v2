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

package integration_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/endpoints/echo/client"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env"
	comp "github.com/GoogleCloudPlatform/esp-v2/tests/env/components"
	"github.com/GoogleCloudPlatform/esp-v2/tests/env/platform"
)

func fullDns(dns string) string {
	return dns + "."
}

func TestDnsResolver(t *testing.T) {
	t.Parallel()
	s := env.NewTestEnv(comp.TestDnsResolver, platform.EchoSidecar)

	// Setup dns resolver.
	backendHost := "dns-resolver-test-backend"
	dnsRecords := map[string]string{
		fullDns(backendHost): "127.0.0.1",
	}
	dnsResolver := comp.NewDnsResolver(s.Ports().DnsResolverPort, dnsRecords)
	go func() {
		if err := dnsResolver.ListenAndServe(); err != nil {
			t.Fatalf("Failed to set udp listener %s\n", err.Error())
		}
	}()

	// Check dns resolver's health.
	dnsResolverAddress := fmt.Sprintf("127.0.0.1:%v", s.Ports().DnsResolverPort)
	if err := comp.CheckDnsResolverHealth(dnsResolverAddress, backendHost, dnsRecords[fullDns(backendHost)]); err != nil {
		t.Fatalf("DNS Resolver is not healthy: %v", err)
	}

	// Setup the whole test framework.
	s.SetBackendAddress(fmt.Sprintf("http://%s:%v", backendHost, s.Ports().BackendServerPort))
	args := []string{"--service_config_id=test-config-id",
		"--rollout_strategy=fixed", "--healthz=/healthz", "--dns_resolver_address=" + dnsResolverAddress}

	defer s.TearDown()
	if err := s.Setup(args); err != nil {
		t.Fatalf("fail to setup test env, %v", err)
	}

	// Do the test.
	wantResp := `{"message":"hello"}`
	url := fmt.Sprintf("http://localhost:%v/echo?key=api-key", s.Ports().ListenerPort)
	resp, err := client.DoPost(url, "hello")
	if err != nil {
		t.Fatalf("got unexpected error: %s", err)
		return
	}

	if !strings.Contains(string(resp), wantResp) {
		t.Errorf("expected: %s, got: %s", wantResp, string(resp))
	}
}
