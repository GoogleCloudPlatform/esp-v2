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
	prpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v11/http/path_rewrite"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/golang/protobuf/ptypes"
	anypb "github.com/golang/protobuf/ptypes/any"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

type PathRewriteGenerator struct{}

func (g *PathRewriteGenerator) FilterName() string {
	return util.PathRewrite
}

func (g *PathRewriteGenerator) GenPerRouteConfig(method *ci.MethodInfo, httpRule *httppattern.Pattern) (*anypb.Any, error) {
	pr := makePathRewriteConfig(method, httpRule)
	if pr == nil {
		return nil, nil
	}

	prAny, err := ptypes.MarshalAny(pr)
	if err != nil {
		return nil, fmt.Errorf("error marshaling path_rewrite per-route config to Any: %v", err)
	}
	return prAny, nil
}

func (g *PathRewriteGenerator) GenFilterConfig(serviceInfo *ci.ServiceInfo) (*hcmpb.HttpFilter, []*ci.MethodInfo, error) {
	perRouteConfigRequiredMethods, needed := needPathRewrite(serviceInfo)
	if !needed {
		return nil, nil, nil
	}
	a, err := ptypes.MarshalAny(&prpb.FilterConfig{})
	if err != nil {
		return nil, nil, err
	}
	return &hcmpb.HttpFilter{
		Name:       util.PathRewrite,
		ConfigType: &hcmpb.HttpFilter_TypedConfig{TypedConfig: a},
	}, perRouteConfigRequiredMethods, nil
}

func needPathRewrite(serviceInfo *ci.ServiceInfo) ([]*ci.MethodInfo, bool) {
	needed := false
	var perRouteConfigRequiredMethods []*ci.MethodInfo
	for _, method := range serviceInfo.Methods {
		for _, httpRule := range method.HttpRule {
			if pr := makePathRewriteConfig(method, httpRule); pr != nil {
				needed = true
				perRouteConfigRequiredMethods = append(perRouteConfigRequiredMethods, method)
			}
		}
	}
	return perRouteConfigRequiredMethods, needed
}

func makePathRewriteConfig(method *ci.MethodInfo, httpRule *httppattern.Pattern) *prpb.PerRouteFilterConfig {
	if method.BackendInfo == nil {
		return nil
	}

	if method.BackendInfo.TranslationType == confpb.BackendRule_APPEND_PATH_TO_ADDRESS {
		if method.BackendInfo.Path != "" {
			return &prpb.PerRouteFilterConfig{
				PathTranslationSpecifier: &prpb.PerRouteFilterConfig_PathPrefix{
					PathPrefix: method.BackendInfo.Path,
				},
			}
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
		}
	}
	return nil
}
