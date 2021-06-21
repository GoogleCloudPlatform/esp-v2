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
	"sort"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	hcmpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/golang/protobuf/ptypes"
	anypb "github.com/golang/protobuf/ptypes/any"

	ci "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"
	aupb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v10/http/backend_auth"
	bapb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v10/http/backend_auth"
	commonpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v10/http/common"
)

var baPerRouteFilterConfigGen = func(method *ci.MethodInfo, httpRule *httppattern.Pattern) (*anypb.Any, error) {
	auPerRoute := &aupb.PerRouteFilterConfig{
		JwtAudience: method.BackendInfo.JwtAudience,
	}
	aupr, err := ptypes.MarshalAny(auPerRoute)
	if err != nil {
		return nil, fmt.Errorf("error marshaling backend_auth per-route config to Any: %v", err)
	}
	return aupr, nil
}

var baFilterGenFunc = func(serviceInfo *ci.ServiceInfo) (*hcmpb.HttpFilter, []*ci.MethodInfo, error) {
	// Use map to collect list of unique jwt audiences.
	var perRouteConfigRequiredMethods []*ci.MethodInfo
	audMap := make(map[string]bool)
	for _, method := range serviceInfo.Methods {
		if method.BackendInfo != nil && method.BackendInfo.JwtAudience != "" {
			audMap[method.BackendInfo.JwtAudience] = true
			perRouteConfigRequiredMethods = append(perRouteConfigRequiredMethods, method)
		}
	}
	// If audMap is empty, not need to add the filter.
	if len(audMap) == 0 {
		return nil, nil, nil
	}

	var audList []string
	for aud := range audMap {
		audList = append(audList, aud)
	}
	// This sort is just for unit-test to compare with expected result.
	sort.Strings(audList)
	backendAuthConfig := &bapb.FilterConfig{
		JwtAudienceList: audList,
	}

	depErrorBehaviorEnum, err := parseDepErrorBehavior(serviceInfo.Options.DependencyErrorBehavior)
	if err != nil {
		return nil, nil, err
	}
	backendAuthConfig.DepErrorBehavior = depErrorBehaviorEnum

	if serviceInfo.Options.BackendAuthCredentials != nil {
		backendAuthConfig.IdTokenInfo = &bapb.FilterConfig_IamToken{
			IamToken: &commonpb.IamTokenInfo{
				IamUri: &commonpb.HttpUri{
					Uri:     fmt.Sprintf("%s%s", serviceInfo.Options.IamURL, util.IamIdentityTokenPath(serviceInfo.Options.BackendAuthCredentials.ServiceAccountEmail)),
					Cluster: util.IamServerClusterName,
					Timeout: ptypes.DurationProto(serviceInfo.Options.HttpRequestTimeout),
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
				Cluster: util.MetadataServerClusterName,
				Timeout: ptypes.DurationProto(serviceInfo.Options.HttpRequestTimeout),
			},
		}
	}
	backendAuthConfigStruct, err := ptypes.MarshalAny(backendAuthConfig)
	if err != nil {
		return nil, nil, err
	}

	backendAuthFilter := &hcmpb.HttpFilter{
		Name:       util.BackendAuth,
		ConfigType: &hcmpb.HttpFilter_TypedConfig{TypedConfig: backendAuthConfigStruct},
	}
	return backendAuthFilter, perRouteConfigRequiredMethods, nil
}
