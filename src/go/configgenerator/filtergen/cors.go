// Copyright 2023 Google LLC
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

package filtergen

import (
	ci "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	corspb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/cors/v3"
	"github.com/golang/protobuf/proto"
)

const (
	// CORSFilterName is the Envoy filter name for debug logging.
	CORSFilterName = "envoy.filters.http.cors"
)

type CORSGenerator struct {
	// skipFilter indicates if this filter is disabled based on options and config.
	skipFilter bool
}

// NewCORSGenerator creates the CORSGenerator with cached config.
func NewCORSGenerator(serviceInfo *ci.ServiceInfo) *CORSGenerator {
	return &CORSGenerator{
		skipFilter: serviceInfo.Options.CorsPreset != "basic" && serviceInfo.Options.CorsPreset != "cors_with_regex",
	}
}

func (g *CORSGenerator) FilterName() string {
	return CORSFilterName
}

func (g *CORSGenerator) IsEnabled() bool {
	return !g.skipFilter
}

func (g *CORSGenerator) GenFilterConfig(serviceInfo *ci.ServiceInfo) (proto.Message, error) {
	return &corspb.Cors{}, nil
}

func (g *CORSGenerator) GenPerRouteConfig(method *ci.MethodInfo, httpRule *httppattern.Pattern) (proto.Message, error) {
	return nil, nil
}
