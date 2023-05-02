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
	gmspb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/grpc_metadata_scrubber"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/proto"
)

const (
	// GrpcMetadataScrubberFilterName is the Envoy filter name for debug logging.
	GrpcMetadataScrubberFilterName = "com.google.espv2.filters.http.grpc_metadata_scrubber"
)

type GRPCMetadataScrubberGenerator struct{}

// NewGRPCMetadataScrubberFilterGensFromOPConfig creates a GRPCMetadataScrubberGenerator from
// OP service config + descriptor + ESPv2 options. It is a FilterGeneratorOPFactory.
func NewGRPCMetadataScrubberFilterGensFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions, params FactoryParams) ([]FilterGenerator, error) {
	if !opts.EnableGrpcForHttp1 {
		return nil, nil
	}

	return []FilterGenerator{
		&GRPCMetadataScrubberGenerator{},
	}, nil
}

func (g *GRPCMetadataScrubberGenerator) FilterName() string {
	return GrpcMetadataScrubberFilterName
}

func (g *GRPCMetadataScrubberGenerator) GenFilterConfig() (proto.Message, error) {
	return &gmspb.FilterConfig{}, nil
}

func (g *GRPCMetadataScrubberGenerator) GenPerRouteConfig(selector string, httpRule *httppattern.Pattern) (proto.Message, error) {
	return nil, nil
}
