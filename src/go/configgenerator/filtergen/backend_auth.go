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
	"sort"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	ci "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	bapb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/backend_auth"
	commonpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/common"
)

const (
	// BackendAuthFilterName is the Envoy filter name for debug logging.
	BackendAuthFilterName = "com.google.espv2.filters.http.backend_auth"
)

type BackendAuthGenerator struct {
	// audMap is the list of unique audiences in the config.
	audMap map[string]bool
}

// NewBackendAuthGenerator creates the BackendAuthGenerator with cached config.
func NewBackendAuthGenerator(serviceInfo *ci.ServiceInfo) *BackendAuthGenerator {
	return &BackendAuthGenerator{
		audMap: getUniqueAudiences(serviceInfo),
	}
}

func (g *BackendAuthGenerator) FilterName() string {
	return BackendAuthFilterName
}

func (g *BackendAuthGenerator) IsEnabled() bool {
	return len(g.audMap) > 0
}

func (g *BackendAuthGenerator) GenPerRouteConfig(method *ci.MethodInfo, httpRule *httppattern.Pattern) (proto.Message, error) {
	if method.BackendInfo == nil || method.BackendInfo.JwtAudience == "" {
		return nil, nil
	}

	return &bapb.PerRouteFilterConfig{
		JwtAudience: method.BackendInfo.JwtAudience,
	}, nil
}

func (g *BackendAuthGenerator) GenFilterConfig(serviceInfo *ci.ServiceInfo) (proto.Message, error) {
	var audList []string
	for aud := range g.audMap {
		audList = append(audList, aud)
	}

	// This sort is just for unit-test to compare with expected result.
	sort.Strings(audList)
	backendAuthConfig := &bapb.FilterConfig{
		JwtAudienceList: audList,
	}

	depErrorBehaviorEnum, err := parseDepErrorBehavior(serviceInfo.Options.DependencyErrorBehavior)
	if err != nil {
		return nil, err
	}
	backendAuthConfig.DepErrorBehavior = depErrorBehaviorEnum

	if serviceInfo.Options.BackendAuthCredentials != nil {
		backendAuthConfig.IdTokenInfo = &bapb.FilterConfig_IamToken{
			IamToken: &commonpb.IamTokenInfo{
				IamUri: &commonpb.HttpUri{
					Uri:     fmt.Sprintf("%s%s", serviceInfo.Options.IamURL, util.IamIdentityTokenPath(serviceInfo.Options.BackendAuthCredentials.ServiceAccountEmail)),
					Cluster: clustergen.IAMServerClusterName,
					Timeout: durationpb.New(serviceInfo.Options.HttpRequestTimeout),
				},
				// Currently only support fetching access token from instance metadata
				// server, not by service account file.
				AccessToken:         serviceInfo.AccessToken,
				ServiceAccountEmail: serviceInfo.Options.BackendAuthCredentials.ServiceAccountEmail,
				Delegates:           serviceInfo.Options.BackendAuthCredentials.Delegates,
			}}
	} else {
		backendAuthConfig.IdTokenInfo = &bapb.FilterConfig_ImdsToken{
			ImdsToken: &commonpb.HttpUri{
				Uri:     fmt.Sprintf("%s%s", serviceInfo.Options.MetadataURL, util.IdentityTokenPath),
				Cluster: clustergen.MetadataServerClusterName,
				Timeout: durationpb.New(serviceInfo.Options.HttpRequestTimeout),
			},
		}
	}

	return backendAuthConfig, nil
}

// getUniqueAudiences returns a list of all unique audiences specified in ServiceInfo.
func getUniqueAudiences(serviceInfo *ci.ServiceInfo) map[string]bool {
	audMap := make(map[string]bool)
	for _, method := range serviceInfo.Methods {
		if method.BackendInfo != nil && method.BackendInfo.JwtAudience != "" {
			audMap[method.BackendInfo.JwtAudience] = true
		}
	}
	return audMap
}
