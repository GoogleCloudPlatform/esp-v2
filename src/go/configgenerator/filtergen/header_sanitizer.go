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
	ci "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	hspb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v11/http/header_sanitizer"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/golang/protobuf/ptypes"
	anypb "github.com/golang/protobuf/ptypes/any"
)

type HeaderSanitizerGenerator struct{}

func (g *HeaderSanitizerGenerator) FilterName() string {
	return util.HeaderSanitizerScrubber
}

func (g *HeaderSanitizerGenerator) IsEnabled() bool {
	return true
}

func (g *HeaderSanitizerGenerator) GenFilterConfig(serviceInfo *ci.ServiceInfo) (*hcmpb.HttpFilter, error) {
	a, err := ptypes.MarshalAny(&hspb.FilterConfig{})
	if err != nil {
		return nil, err
	}
	return &hcmpb.HttpFilter{
		Name:       g.FilterName(),
		ConfigType: &hcmpb.HttpFilter_TypedConfig{TypedConfig: a},
	}, nil
}

func (g *HeaderSanitizerGenerator) GenPerRouteConfig(method *ci.MethodInfo, httpRule *httppattern.Pattern) (*anypb.Any, error) {
	return nil, nil
}
