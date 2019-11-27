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
	commonpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/common"
	scpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/http/service_control"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// methodInfo contains all information about this method.
type methodInfo struct {
	ShortName              string
	HttpRule               []*commonpb.Pattern
	BackendInfo            *backendInfo
	AllowUnregisteredCalls bool
	IsGeneratedOption      bool
	SkipServiceControl     bool
	APIKeyLocations        []*scpb.APIKeyLocation
	MetricCosts            []*scpb.MetricCost
}

// backendInfo stores information from Backend rule for backend rerouting.
type backendInfo struct {
	ClusterName     string
	Uri             string
	Hostname        string
	TranslationType confpb.BackendRule_PathTranslation
	JwtAudience     string
}
