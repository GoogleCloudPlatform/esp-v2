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

// Package routegen provides individual Route Generators to generate RDS config.
package routegen

import (
	"fmt"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/glog"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// RouteGenerator is an interface for objects that generate Envoy routes.
type RouteGenerator interface {
	// RouteType returns the debug name of the route generator.
	RouteType() string

	// GenRouteConfig generates all routes (ordered) for the config.
	//
	// If any FilterGenerators are provided, the per-route config for each filter
	// gen is also attached to each outputted route.
	GenRouteConfig([]filtergen.FilterGenerator) ([]*routepb.Route, error)

	// AffectedHTTPPatterns returns a list of HTTP patterns that this
	// RouteGenerator operates on.
	//
	// Useful for other RouteGenerators to wrap the patterns and output more
	// (aggregated) routes.
	AffectedHTTPPatterns() httppattern.MethodSlice
}

// RouteGeneratorOPFactory is the factory function to create an ordered slice
// of RouteGenerator from One Platform config.
type RouteGeneratorOPFactory func(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) (RouteGenerator, error)

// NoopRouteGenerator is a RouteGenerator that provides empty implementation
// for all optional methods.
type NoopRouteGenerator struct{}

func (g *NoopRouteGenerator) AffectedHTTPPatterns() httppattern.MethodSlice {
	return nil
}

// NewRouteGeneratorsFromOPConfig creates all required RouteGenerators from
// OP service config + descriptor + ESPv2 options.
func NewRouteGeneratorsFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions, factories []RouteGeneratorOPFactory) ([]RouteGenerator, error) {
	var gens []RouteGenerator
	for _, factory := range factories {
		generator, err := factory(serviceConfig, opts)
		if err != nil {
			return nil, fmt.Errorf("fail to run RouteGeneratorOPFactory: %v", err)
		}
		if generator != nil {
			gens = append(gens, generator)
		}
	}

	for i, gen := range gens {
		glog.Infof("RouteGenerator %d is %q", i, gen.RouteType())
	}
	return gens, nil
}
