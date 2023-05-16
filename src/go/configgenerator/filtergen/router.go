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
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	routerpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/proto"
)

const (
	// RouterFilterName is the Envoy filter name for debug logging.
	RouterFilterName = "envoy.filters.http.router"
)

type RouterGenerator struct {
	SuppressEnvoyHeaders bool
	StartChildSpan       bool
}

// NewRouterFilterGensFromOPConfig creates a RouterGenerator from
// OP service config + descriptor + ESPv2 options. It is a FilterGeneratorOPFactory.
func NewRouterFilterGensFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) ([]FilterGenerator, error) {
	return []FilterGenerator{
		&RouterGenerator{
			SuppressEnvoyHeaders: opts.SuppressEnvoyHeaders,
			StartChildSpan:       !opts.DisableTracing,
		},
	}, nil
}

func (g *RouterGenerator) FilterName() string {
	return RouterFilterName
}

func (g *RouterGenerator) GenFilterConfig() (proto.Message, error) {
	return &routerpb.Router{
		SuppressEnvoyHeaders: g.SuppressEnvoyHeaders,
		StartChildSpan:       g.StartChildSpan,
	}, nil
}

func (g *RouterGenerator) GenPerRouteConfig(selector string, httpRule *httppattern.Pattern) (proto.Message, error) {
	return nil, nil
}
