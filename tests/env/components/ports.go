// Copyright 2018 Google Cloud Platform Proxy Authors
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

package components

import (
	"fmt"
	"net"
	"time"

	"github.com/golang/glog"
)

// Dynamic port allocation scheme
// To avoid port conflicts when setting up test env. Each test should use unique ports
// Each test has a unique test_name, its ports will be allocated based on that name

// All integration tests should be listed here to get their test ids
const (
	TestAsymmetricKeys uint16 = iota
	TestAuthJwksCache
	TestAuthOKCheckFail
	TestBackendAuth
	TestDifferentOriginPreflightCors
	TestDifferentOriginSimpleCors
	TestDynamicRouting
	TestGRPC
	TestGRPCErrors
	TestGRPCJwt
	TestGRPCFallback
	TestGRPCLargeRequest
	TestGRPCWeb
	TestGRPCInterops
	TestGRPCInteropMiniStress
	TestGrpcBackendPreflightCors
	TestGrpcBackendSimpleCors
	TestHttp1Basic
	TestHttp1JWT
	TestManagedServiceConfig
	TestPreflightCorsWithBasicPreset
	TestPreflightRequestWithAllowCors
	TestReportGCPAttributes
	TestServiceControlALlHTTPMethod
	TestServiceControlAPIKeyCustomLocation
	TestServiceControlAPIKeyDefaultLocation
	TestServiceControlAPIKeyRestriction
	TestServiceControlBasic
	TestServiceControlCache
	TestServiceControlCheckError
	TestServiceControlCheckNetworkFailClosed
	TestServiceControlCheckNetworkFailOpen
	TestServiceControlCheckRetry
	TestServiceControlCheckTracesWithRetry
	TestServiceControlCheckTimeout
	TestServiceControlCheckWrongServerName
	TestServiceControlCredentialId
	TestServiceControlJwtAuthFail
	TestServiceControlLogHeaders
	TestServiceControlLogJwtPayloads
	TestServiceControlNetworkFailFlagClosed
	TestServiceControlNetworkFailFlagOpen
	TestServiceControlProtocolWithGRPCBackend
	TestServiceControlProtocolWithHTTPBackend
	TestServiceControlQuota
	TestServiceControlQuotaExhausted
	TestServiceControlQuotaRetry
	TestServiceControlQuotaUnavailable
	TestServiceControlReportNetworkFail
	TestServiceControlReportResponseCode
	TestServiceControlReportRetry
	TestServiceControlRequestInDynamicRouting
	TestServiceControlRequestWithAllowCors
	TestServiceControlRequestWithoutAllowCors
	TestServiceControlSkipUsage
	TestServiceControlSkipUsageTraces
	TestSimpleCorsWithBasicPreset
	TestSimpleCorsWithRegexPreset
	TestTranscodingErrors
	TestTranscodingServiceUnavailableError
	TestTranscodingBindings
	TestUnconfiguredRequest
	// The number of total tests. has to be the last one.
	maxTestNum
)

const (
	portBase uint16 = 20000
	// Maximum number of ports used in each test.
	portNum uint16 = 6
)

// Ports stores all used ports
type Ports struct {
	BackendServerPort         uint16
	DynamicRoutingBackendPort uint16
	ListenerPort              uint16
	DiscoveryPort             uint16
	AdminPort                 uint16
	FakeStackdriverPort       uint16
}

func allocPortBase(name uint16) uint16 {
	base := portBase + name*portNum
	for i := 0; i < 10; i++ {
		if allPortFree(base, portNum) {
			return base
		}
		base += maxTestNum * portNum
	}
	glog.Warningf("test(%v) could not find free ports, continue the test...", name)
	return base
}

func allPortFree(base uint16, ports uint16) bool {
	for port := base; port < base+ports; port++ {
		if IsPortUsed(port) {
			glog.Infoln("port is used ", port)
			return false
		}
	}
	return true
}

// IsPortUsed checks if a port is used
func IsPortUsed(port uint16) bool {
	serverPort := fmt.Sprintf("localhost:%v", port)
	_, err := net.DialTimeout("tcp", serverPort, 100*time.Millisecond)
	return err == nil
}

// NewPorts allocate all ports based on test id.
func NewPorts(name uint16) *Ports {
	base := allocPortBase(name)
	glog.Infof("Ports generated for test(%v) is from %v - %v", name, base, base+3)
	return &Ports{
		BackendServerPort:         base,
		DynamicRoutingBackendPort: base + 1,
		ListenerPort:              base + 2,
		DiscoveryPort:             base + 3,
		AdminPort:                 base + 4,
		FakeStackdriverPort:       base + 5,
	}
}
