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

package configinfo

import (
	"fmt"
	"strings"
	"time"

	scpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/service_control"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

const (
	discoveryAPIPrefix = "google.discovery"
)

// MethodInfo contains all information about this method.
type MethodInfo struct {
	ShortName              string
	ApiName                string
	ApiVersion             string
	HttpRule               []*httppattern.Pattern
	BackendInfo            *backendInfo
	HttpBackendInfo        *backendInfo
	AllowUnregisteredCalls bool
	// Method that is generated by ESPv2.
	IsGenerated        bool
	SkipServiceControl bool
	RequireAuth        bool
	ApiKeyLocations    []*scpb.ApiKeyLocation
	MetricCosts        []*scpb.MetricCost
	// All non-unary gRPC methods are considered streaming.
	IsStreaming bool

	// The request type name (not the entire type URL).
	RequestTypeName string

	// The auto-generated cors methods, used to replace snakeName with jsonName in their
	// url templates in config time.
	GeneratedCorsMethod *MethodInfo
}

// backendInfo stores information from Backend rule for backend rerouting.
type backendInfo struct {
	ClusterName     string
	Path            string
	Hostname        string
	TranslationType confpb.BackendRule_PathTranslation
	Port            uint32

	// Audience to use when creating a JWT for backend auth.
	// If empty, backend auth should be disabled for the method.
	JwtAudience string

	// Response timeout for the backend.
	Deadline    time.Duration
	IdleTimeout time.Duration

	// Retry setting on the backend.
	RetryOns             string
	RetryNum             uint
	RetriableStatusCodes []uint32
	PerTryTimeout        time.Duration
}

type SnakeToJsonSegments = map[string]string

func (m *MethodInfo) Operation() string {
	return m.ApiName + "." + m.ShortName
}

func (m *MethodInfo) GRPCPath() string {
	return fmt.Sprintf("/%s/%s", m.ApiName, m.ShortName)
}

func (m *MethodInfo) IsGRPCPath(path string) bool {
	return strings.HasPrefix(path, m.GRPCPath())
}

func IsDiscoveryAPI(operationName string) bool {
	return strings.HasPrefix(operationName, discoveryAPIPrefix)
}
