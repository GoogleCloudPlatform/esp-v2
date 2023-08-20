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
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	"github.com/golang/glog"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	bapb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/backend_auth"
	commonpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/common"
)

const (
	// BackendAuthFilterName is the Envoy filter name for debug logging.
	BackendAuthFilterName = "com.google.espv2.filters.http.backend_auth"
)

type BackendAuthGenerator struct {
	// UniqueAudiences is the list of unique JWT audiences in the config.
	UniqueAudiences map[string]bool

	// AudienceBySelector lists the JWT audience for each method.
	AudienceBySelector map[string]string

	IamURL                  string
	MetadataURL             string
	HttpRequestTimeout      time.Duration
	DependencyErrorBehavior string
	CORSOperationDelimiter  string
	BackendAuthCredentials  *options.IAMCredentialsOptions

	AccessToken *helpers.FilterAccessTokenConfiger

	NoopFilterGenerator
}

// NewBackendAuthFilterGensFromOPConfig creates a BackendAuthGenerator from
// OP service config + descriptor + ESPv2 options. It is a FilterGeneratorOPFactory.
func NewBackendAuthFilterGensFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) ([]FilterGenerator, error) {
	audienceBySelector, uniqueAudiences, err := GetJWTAudiencesBySelectorFromOPConfig(serviceConfig, opts)
	if err != nil {
		return nil, err
	}
	if len(uniqueAudiences) == 0 {
		glog.Info("Not adding backend auth filter gens because there are no audiences.")
		return nil, nil
	}

	return []FilterGenerator{
		&BackendAuthGenerator{
			UniqueAudiences:         uniqueAudiences,
			AudienceBySelector:      audienceBySelector,
			IamURL:                  opts.IamURL,
			MetadataURL:             opts.MetadataURL,
			HttpRequestTimeout:      opts.HttpRequestTimeout,
			DependencyErrorBehavior: opts.DependencyErrorBehavior,
			BackendAuthCredentials:  opts.BackendAuthCredentials,
			CORSOperationDelimiter:  opts.CorsOperationDelimiter,
			AccessToken:             helpers.NewFilterAccessTokenConfigerFromOPConfig(opts),
		},
	}, nil
}

func (g *BackendAuthGenerator) FilterName() string {
	return BackendAuthFilterName
}

// matchAudience matches the selector to the configured audience.
// Accounts for CORS selectors.
func (g *BackendAuthGenerator) matchAudience(selector string) (string, error) {
	if audience, ok := g.AudienceBySelector[selector]; ok {
		return audience, nil
	}

	// Try matching CORS selector.
	originalSelector, err := CORSSelectorToSelector(selector, g.CORSOperationDelimiter)
	if err != nil {
		return "", err
	}
	if originalSelector == "" {
		// No route match.
		return "", nil
	}

	if audience, ok := g.AudienceBySelector[originalSelector]; ok {
		return audience, nil
	}

	// No route match.
	return "", nil
}

func (g *BackendAuthGenerator) GenPerRouteConfig(selector string, httpRule *httppattern.Pattern) (proto.Message, error) {
	audience, err := g.matchAudience(selector)
	if err != nil {
		return nil, err
	}
	if audience == "" {
		return nil, nil
	}

	return &bapb.PerRouteFilterConfig{
		JwtAudience: audience,
	}, nil
}

func (g *BackendAuthGenerator) GenFilterConfig() (proto.Message, error) {
	var audList []string
	for aud := range g.UniqueAudiences {
		audList = append(audList, aud)
	}

	// This sort is just for unit-test to compare with expected result.
	sort.Strings(audList)
	backendAuthConfig := &bapb.FilterConfig{
		JwtAudienceList: audList,
	}

	depErrorBehaviorEnum, err := ParseDepErrorBehavior(g.DependencyErrorBehavior)
	if err != nil {
		return nil, err
	}
	backendAuthConfig.DepErrorBehavior = depErrorBehaviorEnum

	if g.BackendAuthCredentials != nil {
		backendAuthConfig.IdTokenInfo = &bapb.FilterConfig_IamToken{
			IamToken: &commonpb.IamTokenInfo{
				IamUri: &commonpb.HttpUri{
					Uri:     fmt.Sprintf("%s%s", g.IamURL, util.IamIdentityTokenPath(g.BackendAuthCredentials.ServiceAccountEmail)),
					Cluster: clustergen.IAMServerClusterName,
					Timeout: durationpb.New(g.HttpRequestTimeout),
				},
				// Currently only support fetching access token from instance metadata
				// server, not by service account file.
				AccessToken:         g.AccessToken.MakeAccessTokenConfig(),
				ServiceAccountEmail: g.BackendAuthCredentials.ServiceAccountEmail,
				Delegates:           g.BackendAuthCredentials.Delegates,
			}}
	} else {
		backendAuthConfig.IdTokenInfo = &bapb.FilterConfig_ImdsToken{
			ImdsToken: &commonpb.HttpUri{
				Uri:     fmt.Sprintf("%s%s", g.MetadataURL, util.IdentityTokenPath),
				Cluster: clustergen.MetadataServerClusterName,
				Timeout: durationpb.New(g.HttpRequestTimeout),
			},
		}
	}

	return backendAuthConfig, nil
}

// GetJWTAudiencesBySelectorFromOPConfig returns:
// 1. A map of selector to JWT audience for the route.
// 2. A set of all unique JWT audiences (for optimization).
//
// Replaces ServiceInfo::ruleToBackendInfo.
func GetJWTAudiencesBySelectorFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) (map[string]string, map[string]bool, error) {
	uniqueAudiences := make(map[string]bool)
	audienceBySelector := make(map[string]string)

	for _, rule := range serviceConfig.GetBackend().GetRules() {
		if util.ShouldSkipOPDiscoveryAPI(rule.GetSelector(), opts.AllowDiscoveryAPIs) {
			glog.Warningf("Skip backend rule %q because discovery API is not supported.", rule.GetSelector())
			continue
		}

		jwtAud, err := parseJwtAudFromBackendRule(rule)
		if err != nil {
			return nil, nil, fmt.Errorf("fail to parse JWT audience for backend rule %q: %v", rule.GetSelector(), err)
		}

		if jwtAud != "" && opts.NonGCP {
			glog.Warningf("Backend authentication is enabled for method %q, "+
				"but ESPv2 is running on non-GCP. To prevent contacting GCP services, "+
				"backend authentication is automatically being disabled for this method.",
				rule.GetSelector())
			continue
		}

		if jwtAud != "" {
			uniqueAudiences[jwtAud] = true
			audienceBySelector[rule.GetSelector()] = jwtAud
		}
	}

	return audienceBySelector, uniqueAudiences, nil
}

// parseJwtAudFromBackendRule returns the correct JWT audience for the given BackendRule.
//
// Replaces ServiceInfo::determineBackendAuthJwtAud.
func parseJwtAudFromBackendRule(r *servicepb.BackendRule) (string, error) {
	//TODO(taoxuy): b/149334660 Check if the scopes for IAM include the path prefix
	switch r.GetAuthentication().(type) {
	case *servicepb.BackendRule_JwtAudience:
		return r.GetJwtAudience(), nil
	case *servicepb.BackendRule_DisableAuth:
		if r.GetDisableAuth() {
			return "", nil
		}
		return BackendAddressToJWTAud(r.GetAddress())
	default:
		if r.Address == "" {
			return "", nil
		}
		return BackendAddressToJWTAud(r.GetAddress())
	}
}

// BackendAddressToJWTAud transforms the backend address into the proper JWT
// audience.
//
// If the backend address's scheme is grpc/grpcs, it is changed to http/https.
func BackendAddressToJWTAud(address string) (string, error) {
	scheme, hostname, _, _, err := util.ParseURI(address)
	if err != nil {
		return "", fmt.Errorf("error parsing backend address for JWT audience: %v", err)
	}

	_, useTLS, err := util.ParseBackendProtocol(scheme, "")
	if err != nil {
		return "", fmt.Errorf("error parsing backend protocol for JWT audience: %v", err)
	}

	if useTLS {
		return fmt.Sprintf("https://%s", hostname), nil
	}
	return fmt.Sprintf("http://%s", hostname), nil
}
