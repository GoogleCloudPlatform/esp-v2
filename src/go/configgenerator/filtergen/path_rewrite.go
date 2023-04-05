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
	ci "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	prpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/path_rewrite"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	"github.com/golang/protobuf/proto"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

const (
	// PathRewriteFilterName is the Envoy filter name for debug logging.
	PathRewriteFilterName = "com.google.espv2.filters.http.path_rewrite"
)

type PathRewriteGenerator struct {
	// skipFilter indicates if this filter is disabled based on options and config.
	skipFilter bool
}

// NewPathRewriteGenerator creates the PathRewriteGenerator with cached config.
func NewPathRewriteGenerator(serviceInfo *ci.ServiceInfo) *PathRewriteGenerator {
	g := &PathRewriteGenerator{}

	for _, method := range serviceInfo.Methods {
		for _, httpRule := range method.HttpRule {
			if pr, err := g.GenPerRouteConfig(method, httpRule); err == nil && pr != nil {
				return g
			}
		}
	}

	g.skipFilter = true
	return g
}

func (g *PathRewriteGenerator) FilterName() string {
	return PathRewriteFilterName
}

func (g *PathRewriteGenerator) IsEnabled() bool {
	return !g.skipFilter
}

func (g *PathRewriteGenerator) GenFilterConfig(serviceInfo *ci.ServiceInfo) (proto.Message, error) {
	return &prpb.FilterConfig{}, nil
}

func (g *PathRewriteGenerator) GenPerRouteConfig(method *ci.MethodInfo, httpRule *httppattern.Pattern) (proto.Message, error) {
	if method.BackendInfo == nil {
		return nil, nil
	}

	if method.BackendInfo.TranslationType == confpb.BackendRule_APPEND_PATH_TO_ADDRESS {
		if method.BackendInfo.Path != "" {
			return &prpb.PerRouteFilterConfig{
				PathTranslationSpecifier: &prpb.PerRouteFilterConfig_PathPrefix{
					PathPrefix: method.BackendInfo.Path,
				},
			}, nil
		}
	}
	if method.BackendInfo.TranslationType == confpb.BackendRule_CONSTANT_ADDRESS {
		constPath := &prpb.ConstantPath{
			Path: method.BackendInfo.Path,
		}

		if uriTemplate := httpRule.UriTemplate; uriTemplate != nil && len(uriTemplate.Variables) > 0 {
			constPath.UrlTemplate = uriTemplate.ExactMatchString(false)
		}
		return &prpb.PerRouteFilterConfig{
			PathTranslationSpecifier: &prpb.PerRouteFilterConfig_ConstantPath{
				ConstantPath: constPath,
			},
		}, nil
	}
	return nil, nil
}
