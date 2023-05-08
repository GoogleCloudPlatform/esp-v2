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

package filtergentemp

import (
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	corspb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/cors/v3"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/proto"
)

const (
	// CORSFilterName is the Envoy filter name for debug logging.
	CORSFilterName = "envoy.filters.http.cors"
)

type CORSGenerator struct{}

// NewCORSFilterGensFromOPConfig creates a CORSGenerator from
// OP service config + descriptor + ESPv2 options. It is a FilterGeneratorOPFactory.
func NewCORSFilterGensFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions, params FactoryParams) ([]FilterGenerator, error) {
	if opts.CorsPreset != "basic" && opts.CorsPreset != "cors_with_regex" {
		return nil, nil
	}

	return []FilterGenerator{
		&CORSGenerator{},
	}, nil
}

func (g *CORSGenerator) FilterName() string {
	return CORSFilterName
}

func (g *CORSGenerator) GenFilterConfig() (proto.Message, error) {
	return &corspb.Cors{}, nil
}

func (g *CORSGenerator) GenPerRouteConfig(selector string, httpRule *httppattern.Pattern) (proto.Message, error) {
	return nil, nil
}
