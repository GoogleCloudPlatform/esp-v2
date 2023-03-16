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

package configgenerator

import (
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen"
	ci "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	anypb "github.com/golang/protobuf/ptypes/any"
)

// FilterGenerator is an interface for objects that generate Envoy filters.
type FilterGenerator interface {

	// FilterName returns the name of the filter.
	FilterName() string

	// IsEnabled returns true if the filter config should be generated.
	// If false, none of the generation methods will be called.
	IsEnabled() bool

	// GenFilterConfig generates the filter config.
	//
	// Return (nil, nil) if the filter has no listener-level config, but may
	// have per-route configurations.
	GenFilterConfig(*ci.ServiceInfo) (*hcmpb.HttpFilter, error)

	// GenPerRouteConfig generates the per-route config for the given HTTP route (HTTP pattern).
	// The MethodInfo that contains the route is also provided.
	//
	// This method is called on all routes. Return (nil, nil) to indicate the
	// filter does NOT require a per-route config for the given route.
	GenPerRouteConfig(*ci.MethodInfo, *httppattern.Pattern) (*anypb.Any, error)
}

// MakeFilterGenerators provide of a slice of FilterGenerator in sequence.
func MakeFilterGenerators(serviceInfo *ci.ServiceInfo) ([]FilterGenerator, error) {
	return []FilterGenerator{
		filtergen.NewCORSGenerator(serviceInfo),

		// Health check filter is behind Path Matcher filter, since Service Control
		// filter needs to get the corresponding rule for health check in order to skip Report
		filtergen.NewHealthCheckGenerator(serviceInfo),
		filtergen.NewCompressorGenerator(serviceInfo, filtergen.GzipCompressor),
		filtergen.NewCompressorGenerator(serviceInfo, filtergen.BrotliCompressor),
		filtergen.NewJwtAuthnGenerator(serviceInfo),
		filtergen.NewServiceControlGenerator(serviceInfo),

		// grpc-web filter should be before grpc transcoder filter.
		// It converts content-type application/grpc-web to application/grpc and
		// grpc transcoder will bypass requests with application/grpc content type.
		// Otherwise grpc transcoder will try to transcode a grpc-web request which
		// will fail.
		filtergen.NewGRPCWebGenerator(serviceInfo),
		filtergen.NewGRPCTranscoderGenerator(serviceInfo),

		filtergen.NewBackendAuthGenerator(serviceInfo),
		filtergen.NewPathRewriteGenerator(serviceInfo),
		filtergen.NewGRPCMetadataScrubberGenerator(serviceInfo),

		// Add Envoy Router filter so requests are routed upstream.
		// Router filter should be the last.
		&filtergen.RouterGenerator{},
	}, nil
}
