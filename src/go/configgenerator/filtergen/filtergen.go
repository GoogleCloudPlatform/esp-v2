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

// Package filtergen provides individual Filter Generators to generate an
// xDS filter config.
package filtergen

import (
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/proto"
)

// FilterGenerator is an interface for objects that generate Envoy filters.
type FilterGenerator interface {

	// FilterName returns the debug name of the filter.
	FilterName() string

	// GenFilterConfig generates the filter config.
	//
	// Return type is the filter's config proto.
	//
	// Return (nil, nil) if the filter has no listener-level config, but may
	// have per-route configurations.
	GenFilterConfig() (proto.Message, error)

	// GenPerRouteConfig generates the per-route config for the given selector and HTTP route (HTTP pattern).
	//
	// Return type is the filter's per-route config proto.
	//
	// This method is called on all routes. Return (nil, nil) to indicate the
	// filter does NOT require a per-route config for the given route.
	GenPerRouteConfig(string, *httppattern.Pattern) (proto.Message, error)

	// GenPerHostConfig generates the per-host config in RDS for the given virtual
	// host name. Useful in case a filter needs to apply config for the entire
	// virtual host.
	//
	// Return type is the filter's per-host config proto.
	//
	// This method is called on all virtual hosts. Return (nil, nil) to indicate the
	// filter does NOT require a per-host config for the given virtual host.
	GenPerHostConfig(string) (proto.Message, error)
}

// FilterGeneratorOPFactory is the factory function to create an ordered slice
// of FilterGenerator from One Platform config.
//
// The majority of factories will only return 1 FilterGenerator, but they should
// be encapsulated by a slice for generalization.
type FilterGeneratorOPFactory func(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) ([]FilterGenerator, error)

// NoopFilterGenerator is a FilterGenerator that provides empty implementation
// for all optional methods.
type NoopFilterGenerator struct{}

func (g *NoopFilterGenerator) GenPerRouteConfig(string, *httppattern.Pattern) (proto.Message, error) {
	return nil, nil
}

func (g *NoopFilterGenerator) GenPerHostConfig(string) (proto.Message, error) {
	return nil, nil
}
