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
	hspb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/header_sanitizer"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	"google.golang.org/protobuf/proto"
)

const (
	// HeaderSanitizerFilterName is the Envoy filter name for debug logging.
	HeaderSanitizerFilterName = "com.google.espv2.filters.http.header_sanitizer"
)

type HeaderSanitizerGenerator struct{}

func (g *HeaderSanitizerGenerator) FilterName() string {
	return HeaderSanitizerFilterName
}

func (g *HeaderSanitizerGenerator) IsEnabled() bool {
	return true
}

func (g *HeaderSanitizerGenerator) GenFilterConfig(serviceInfo *ci.ServiceInfo) (proto.Message, error) {
	return &hspb.FilterConfig{}, nil
}

func (g *HeaderSanitizerGenerator) GenPerRouteConfig(method *ci.MethodInfo, httpRule *httppattern.Pattern) (proto.Message, error) {
	return nil, nil
}
