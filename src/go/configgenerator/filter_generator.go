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
	"fmt"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/golang/glog"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func GetESPv2FilterGenFactories(scParams filtergen.ServiceControlOPFactoryParams) []filtergen.FilterGeneratorOPFactory {
	return []filtergen.FilterGeneratorOPFactory{
		filtergen.NewHeaderSanitizerFilterGensFromOPConfig,
		filtergen.NewCORSFilterGensFromOPConfig,

		// Health check filter is behind Path Matcher filter, since Service Control
		// filter needs to get the corresponding rule for health check in order to skip Report
		filtergen.NewHealthCheckFilterGensFromOPConfig,
		filtergen.NewCompressorFilterGensFromOPConfig,
		filtergen.NewJwtAuthnFilterGensFromOPConfig,
		func(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) ([]filtergen.FilterGenerator, error) {
			return filtergen.NewServiceControlFilterGensFromOPConfig(serviceConfig, opts, scParams)
		},

		// grpc-web filter should be before grpc transcoder filter.
		// It converts content-type application/grpc-web to application/grpc and
		// grpc transcoder will bypass requests with application/grpc content type.
		// Otherwise grpc transcoder will try to transcode a grpc-web request which
		// will fail.
		filtergen.NewGRPCWebFilterGensFromOPConfig,
		filtergen.NewGRPCTranscoderFilterGensFromOPConfig,
		filtergen.NewBackendAuthFilterGensFromOPConfig,
		filtergen.NewPathRewriteFilterGensFromOPConfig,
		filtergen.NewGRPCMetadataScrubberFilterGensFromOPConfig,

		// Add Envoy Router filter so requests are routed upstream.
		// Router filter should be the last.
		filtergen.NewRouterFilterGensFromOPConfig,
	}
}

// NewFilterGeneratorsFromOPConfig creates all required FilterGenerators from
// OP service config + descriptor + ESPv2 options.
func NewFilterGeneratorsFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions, factories []filtergen.FilterGeneratorOPFactory) ([]filtergen.FilterGenerator, error) {
	var gens []filtergen.FilterGenerator
	for _, factory := range factories {
		generator, err := factory(serviceConfig, opts)
		if err != nil {
			return nil, fmt.Errorf("fail to run FilterGeneratorOPFactory: %v", err)
		}
		gens = append(gens, generator...)
	}

	for i, gen := range gens {
		glog.Infof("FilterGenerator %d is %q", i, gen.FilterName())
	}
	return gens, nil
}
