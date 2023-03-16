// Copyright 2021 Google LLC
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
	"fmt"

	ci "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	brpb "github.com/envoyproxy/go-control-plane/envoy/extensions/compression/brotli/compressor/v3"
	gzippb "github.com/envoyproxy/go-control-plane/envoy/extensions/compression/gzip/compressor/v3"
	comppb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/compressor/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	anypb "github.com/golang/protobuf/ptypes/any"
)

type CompressorType int

const (
	GzipCompressor CompressorType = iota
	BrotliCompressor
)

type CompressorGenerator struct {
	compressorType CompressorType

	// skipFilter indicates if this filter is disabled based on options and config.
	skipFilter bool
}

// NewCompressorGenerator creates the CompressorGenerator with cached config.
func NewCompressorGenerator(serviceInfo *ci.ServiceInfo, compressorType CompressorType) *CompressorGenerator {
	return &CompressorGenerator{
		compressorType: compressorType,
		skipFilter:     !serviceInfo.Options.EnableResponseCompression,
	}
}

func (g *CompressorGenerator) FilterName() string {
	switch g.compressorType {
	case GzipCompressor:
		return util.EnvoyGzipCompressor
	case BrotliCompressor:
		return util.EnvoyBrotliCompressor
	}
	return ""
}

func (g *CompressorGenerator) IsEnabled() bool {
	return !g.skipFilter
}

func (g *CompressorGenerator) GenFilterConfig(serviceInfo *ci.ServiceInfo) (*hcmpb.HttpFilter, error) {
	cfg, name, err := g.getCompressorConfig()
	if err != nil {
		return nil, err
	}
	ca, err := ptypes.MarshalAny(cfg)
	if err != nil {
		return nil, fmt.Errorf("error marshaling %s Compressor config to Any: %v", name, err)
	}
	cmp := &comppb.Compressor{
		CompressorLibrary: &corepb.TypedExtensionConfig{
			Name:        name,
			TypedConfig: ca,
		},
	}
	a, err := ptypes.MarshalAny(cmp)
	if err != nil {
		return nil, fmt.Errorf("error marshaling Compressor filter config to Any: %v", err)
	}
	return &hcmpb.HttpFilter{
		Name: util.EnvoyCompressorFilter,
		ConfigType: &hcmpb.HttpFilter_TypedConfig{
			TypedConfig: a,
		},
	}, nil
}

func (g *CompressorGenerator) GenPerRouteConfig(method *ci.MethodInfo, httpRule *httppattern.Pattern) (*anypb.Any, error) {
	return nil, nil
}

func (g *CompressorGenerator) getCompressorConfig() (proto.Message, string, error) {
	switch g.compressorType {
	case GzipCompressor:
		return &gzippb.Gzip{}, util.EnvoyGzipCompressor, nil
	case BrotliCompressor:
		return &brpb.Brotli{}, util.EnvoyBrotliCompressor, nil
	}
	return nil, "", fmt.Errorf("unknown compressor type: %v", g.compressorType)
}
