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

	// GenFilterConfig generates the filter config and a list of methods that need per-route configs.
	GenFilterConfig(*ci.ServiceInfo) (*hcmpb.HttpFilter, []*ci.MethodInfo, error)

	// GenPerRouteConfig generates the per-route config for the given HTTP route.
	// This method is only called on the routes that `GenFilterConfig` returns.
	GenPerRouteConfig(*ci.MethodInfo, *httppattern.Pattern) (*anypb.Any, error)
}

// MakeFilterGenerators provide of a slice of FilterGenerator in sequence.
func MakeFilterGenerators(serviceInfo *ci.ServiceInfo) ([]FilterGenerator, error) {
	var filterGenerators []FilterGenerator

	if serviceInfo.Options.CorsPreset == "basic" || serviceInfo.Options.CorsPreset == "cors_with_regex" {
		filterGenerators = append(filterGenerators, &filtergen.CORSGenerator{})
	}

	// Add Health Check filter if needed. It must behind Path Matcher filter, since Service Control
	// filter needs to get the corresponding rule for health check cmake depend.insalls, in order to skip Report
	if serviceInfo.Options.Healthz != "" {
		filterGenerators = append(filterGenerators, &filtergen.HealthCheckGenerator{})
	}

	if serviceInfo.Options.EnableResponseCompression {
		filterGenerators = append(filterGenerators, &filtergen.CompressorGenerator{
			CompressorType: filtergen.GzipCompressor,
		})
		filterGenerators = append(filterGenerators, &filtergen.CompressorGenerator{
			CompressorType: filtergen.BrotliCompressor,
		})
	}

	// Add JWT Authn filter if needed.
	if !serviceInfo.Options.SkipJwtAuthnFilter {
		filterGenerators = append(filterGenerators, &filtergen.JwtAuthnGenerator{})
	}

	// Add Service Control filter if needed.
	if !serviceInfo.Options.SkipServiceControlFilter {
		filterGenerators = append(filterGenerators, &filtergen.ServiceControlGenerator{})
	}

	// Add gRPC Transcoder filter and gRPCWeb filter configs for gRPC backend.
	if serviceInfo.GrpcSupportRequired {
		// grpc-web filter should be before grpc transcoder filter.
		// It converts content-type application/grpc-web to application/grpc and
		// grpc transcoder will bypass requests with application/grpc content type.
		// Otherwise grpc transcoder will try to transcode a grpc-web request which
		// will fail.
		filterGenerators = append(filterGenerators, &filtergen.GRPCWebGenerator{})
		filterGenerators = append(filterGenerators, &filtergen.GRPCTranscoderGenerator{})
	}

	filterGenerators = append(filterGenerators, &filtergen.BackendAuthGenerator{})
	filterGenerators = append(filterGenerators, &filtergen.PathRewriteGenerator{})

	if serviceInfo.Options.EnableGrpcForHttp1 {
		// Add GrpcMetadataScrubber filter to retain gRPC trailers
		filterGenerators = append(filterGenerators, &filtergen.GRPCMetadataScrubberGenerator{})
	}

	// Add Envoy Router filter so requests are routed upstream.
	// Router filter should be the last.
	filterGenerators = append(filterGenerators, &filtergen.RouterGenerator{})
	return filterGenerators, nil
}
