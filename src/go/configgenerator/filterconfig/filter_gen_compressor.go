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

package filterconfig

import (
	"fmt"

	ci "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	brpb "github.com/envoyproxy/go-control-plane/envoy/extensions/compression/brotli/compressor/v3"
	gzippb "github.com/envoyproxy/go-control-plane/envoy/extensions/compression/gzip/compressor/v3"
	comppb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/compressor/v3"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
)

type compressorType int

const (
	gzipCompressor compressorType = iota
	brotliCompressor
)

func getCompressorConfig(c compressorType) (proto.Message, string, error) {
	switch c {
	case gzipCompressor:
		return &gzippb.Gzip{}, util.EnvoyGzipCompressor, nil
	case brotliCompressor:
		return &brpb.Brotli{}, util.EnvoyBrotliCompressor, nil
	}
	return nil, "", fmt.Errorf("unknown compressor type: %v", c)
}

func createComprssorFilter(c compressorType) (*hcmpb.HttpFilter, []*ci.MethodInfo, error) {
	cfg, name, err := getCompressorConfig(c)
	if err != nil {
		return nil, nil, err
	}
	ca, err := ptypes.MarshalAny(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("error marshaling %s Compressor config to Any: %v", name, err)
	}
	cmp := &comppb.Compressor{
		CompressorLibrary: &corepb.TypedExtensionConfig{
			Name:        name,
			TypedConfig: ca,
		},
	}
	a, err := ptypes.MarshalAny(cmp)
	if err != nil {
		return nil, nil, fmt.Errorf("error marshaling Compressor filter config to Any: %v", err)
	}
	return &hcmpb.HttpFilter{
		Name:       util.EnvoyCompressorFilter,
		ConfigType: &hcmpb.HttpFilter_TypedConfig{a},
	}, nil, nil

}

var gzipCompressorGenFunc = func(sc *ci.ServiceInfo) (*hcmpb.HttpFilter, []*ci.MethodInfo, error) {
	return createComprssorFilter(gzipCompressor)
}

var brotliCompressorGenFunc = func(sc *ci.ServiceInfo) (*hcmpb.HttpFilter, []*ci.MethodInfo, error) {
	return createComprssorFilter(brotliCompressor)
}
