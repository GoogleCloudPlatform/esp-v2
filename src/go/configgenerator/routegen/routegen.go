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
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// RouteGenerator is an interface for objects that generate Envoy routes.
type RouteGenerator interface {
	// GenRouteConfig generates all routes (ordered) for the config.
	GenRouteConfig() ([]*routepb.Route, error)
}

// RouteGeneratorOPFactory is the factory function to create an ordered slice
// of RouteGenerator from One Platform config.
type RouteGeneratorOPFactory func(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) ([]RouteGenerator, error)
