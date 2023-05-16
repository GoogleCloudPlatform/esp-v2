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

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	prpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/path_rewrite"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	"github.com/golang/glog"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/proto"
)

const (
	// PathRewriteFilterName is the Envoy filter name for debug logging.
	PathRewriteFilterName = "com.google.espv2.filters.http.path_rewrite"
)

type PathRewriteGenerator struct {
	TranslationInfoBySelector map[string]TranslationInfo
}

// TranslationInfo captures https://cloud.google.com/endpoints/docs/openapi/openapi-extensions#understanding_path_translation.
type TranslationInfo struct {
	// TranslationType cannot be UNSPECIFIED. Do not add it to the config if so.
	TranslationType confpb.BackendRule_PathTranslation

	// Path cannot be empty. Do NOT add it to the config if so.
	Path string
}

// NewPathRewriteFilterGensFromOPConfig creates a PathRewriteGenerator from
// OP service config + descriptor + ESPv2 options. It is a FilterGeneratorOPFactory.
func NewPathRewriteFilterGensFromOPConfig(serviceConfig *confpb.Service, opts options.ConfigGeneratorOptions) ([]FilterGenerator, error) {
	info, err := GenTranslationInfoFromOPConfig(serviceConfig, opts)
	if err != nil {
		return nil, err
	}

	if len(info) == 0 {
		glog.Infof("Skipping path rewrite filter because there are no backend rules that need path translation")
		return nil, nil
	}

	return []FilterGenerator{
		&PathRewriteGenerator{
			TranslationInfoBySelector: info,
		},
	}, nil
}

func (g *PathRewriteGenerator) FilterName() string {
	return PathRewriteFilterName
}

func (g *PathRewriteGenerator) GenFilterConfig() (proto.Message, error) {
	return &prpb.FilterConfig{}, nil
}

// matchTranslationInfo matches the selector to the configured info.
// Accounts for CORS selectors.
func (g *PathRewriteGenerator) matchTranslationInfo(selector string) (TranslationInfo, error) {
	if translationInfo, ok := g.TranslationInfoBySelector[selector]; ok {
		return translationInfo, nil
	}

	// Try matching CORS selector.
	originalSelector, err := CORSSelectorToSelector(selector)
	if err != nil {
		return TranslationInfo{}, err
	}
	if originalSelector == "" {
		// No route match.
		return TranslationInfo{}, nil
	}

	if translationInfo, ok := g.TranslationInfoBySelector[originalSelector]; ok {
		return translationInfo, nil
	}

	// No route match.
	return TranslationInfo{}, nil
}

func (g *PathRewriteGenerator) GenPerRouteConfig(selector string, httpRule *httppattern.Pattern) (proto.Message, error) {
	translationInfo, err := g.matchTranslationInfo(selector)
	if err != nil {
		return nil, err
	}

	if translationInfo.TranslationType == confpb.BackendRule_APPEND_PATH_TO_ADDRESS {
		return &prpb.PerRouteFilterConfig{
			PathTranslationSpecifier: &prpb.PerRouteFilterConfig_PathPrefix{
				PathPrefix: translationInfo.Path,
			},
		}, nil
	}
	if translationInfo.TranslationType == confpb.BackendRule_CONSTANT_ADDRESS {
		constPath := &prpb.ConstantPath{
			Path: translationInfo.Path,
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

// GenTranslationInfoFromOPConfig returns per-route related translation information for each selector.
//
// Replaces ServiceInfo::ruleToBackendInfo.
func GenTranslationInfoFromOPConfig(serviceConfig *confpb.Service, opts options.ConfigGeneratorOptions) (map[string]TranslationInfo, error) {
	if opts.EnableBackendAddressOverride {
		glog.Infof("Skipping create path rewrite translation info because backend address override is enabled.")
		return nil, nil
	}

	infoBySelector := make(map[string]TranslationInfo)
	for _, rule := range serviceConfig.GetBackend().GetRules() {
		if util.ShouldSkipOPDiscoveryAPI(rule.GetSelector(), opts.AllowDiscoveryAPIs) {
			glog.Warningf("Skip backend rule %q because discovery API is not supported.", rule.GetSelector())
			continue
		}

		if rule.GetAddress() == "" {
			glog.Infof("Skip backend rule %q because it does not have dynamic routing address.", rule.GetSelector())
			continue
		}

		_, _, _, path, err := util.ParseURI(rule.GetAddress())
		if err != nil {
			return nil, fmt.Errorf("error parsing remote backend rule's address for operation %q: %v", rule.GetAddress(), err)
		}

		// For CONSTANT_ADDRESS, an empty uri will generate an empty path header.
		// It is an invalid Http header if path is empty.
		if path == "" && rule.GetPathTranslation() == confpb.BackendRule_CONSTANT_ADDRESS {
			path = "/"
		}

		if path == "" || rule.GetPathTranslation() == confpb.BackendRule_PATH_TRANSLATION_UNSPECIFIED {
			continue
		}

		infoBySelector[rule.GetSelector()] = TranslationInfo{
			TranslationType: rule.GetPathTranslation(),
			Path:            path,
		}
	}

	return infoBySelector, nil
}
