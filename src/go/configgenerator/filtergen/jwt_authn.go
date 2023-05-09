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
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	"github.com/golang/glog"
	"google.golang.org/protobuf/proto"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"

	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	jwtpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	// JWTAuthnFilterName is the Envoy filter name for debug logging.
	JWTAuthnFilterName = "envoy.filters.http.jwt_authn"
)

type JwtAuthnGenerator struct {
	// ServiceName is the service config name.
	ServiceName string

	// AuthConfig is the full authentication config from the OP service config.
	// TODO(nareddyt): This should be generalized, but since we only use jwt_authn
	// filter in ESPv2, we can generalize it later.
	AuthConfig *confpb.Authentication

	// AuthRequiredBySelector maps which selectors require per-route level authn
	// config.
	AuthRequiredBySelector map[string]bool

	// General options below.

	HttpRequestTimeout    time.Duration
	GeneratedHeaderPrefix string

	// JWT Authn specific options below.

	JwksCacheDurationInS               int
	DisableJwksAsyncFetch              bool
	JwksAsyncFetchFastListener         bool
	JwksFetchNumRetries                int
	JwksFetchRetryBackOffBaseInterval  time.Duration
	JwksFetchRetryBackOffMaxInterval   time.Duration
	JwtPadForwardPayloadHeader         bool
	DisableJwtAudienceServiceNameCheck bool
	JwtCacheSize                       uint
}

// NewJwtAuthnFilterGensFromOPConfig creates a JwtAuthnGenerator from
// OP service config + descriptor + ESPv2 options. It is a FilterGeneratorOPFactory.
func NewJwtAuthnFilterGensFromOPConfig(serviceConfig *confpb.Service, opts options.ConfigGeneratorOptions, params FactoryParams) ([]FilterGenerator, error) {
	if opts.SkipJwtAuthnFilter {
		glog.Infof("Not adding JWT authn filter gen because the feature is disabled by option.")
		return nil, nil
	}

	auth := serviceConfig.GetAuthentication()
	if len(auth.GetProviders()) == 0 {
		glog.Infof("Not adding JWT authn filter gen because there are no authentication rules in OP config.")
		return nil, nil
	}

	authRequiredBySelector, err := GetAuthRequiredSelectorsFromOPConfig(serviceConfig, opts)
	if err != nil {
		return nil, err
	}

	return []FilterGenerator{
		&JwtAuthnGenerator{
			ServiceName:                        serviceConfig.GetName(),
			AuthConfig:                         auth,
			AuthRequiredBySelector:             authRequiredBySelector,
			HttpRequestTimeout:                 opts.HttpRequestTimeout,
			GeneratedHeaderPrefix:              opts.GeneratedHeaderPrefix,
			JwksCacheDurationInS:               opts.JwksCacheDurationInS,
			DisableJwksAsyncFetch:              opts.DisableJwksAsyncFetch,
			JwksAsyncFetchFastListener:         opts.JwksAsyncFetchFastListener,
			JwksFetchNumRetries:                opts.JwksFetchNumRetries,
			JwksFetchRetryBackOffBaseInterval:  opts.JwksFetchRetryBackOffBaseInterval,
			JwksFetchRetryBackOffMaxInterval:   opts.JwksFetchRetryBackOffMaxInterval,
			JwtPadForwardPayloadHeader:         opts.JwtPadForwardPayloadHeader,
			DisableJwtAudienceServiceNameCheck: opts.DisableJwtAudienceServiceNameCheck,
			JwtCacheSize:                       opts.JwtCacheSize,
		},
	}, nil
}

func (g *JwtAuthnGenerator) FilterName() string {
	return JWTAuthnFilterName
}

func (g *JwtAuthnGenerator) GenPerRouteConfig(selector string, httpRule *httppattern.Pattern) (proto.Message, error) {
	if authRequired := g.AuthRequiredBySelector[selector]; !authRequired {
		return nil, nil
	}

	return &jwtpb.PerRouteConfig{
		RequirementSpecifier: &jwtpb.PerRouteConfig_RequirementName{
			RequirementName: selector,
		},
	}, nil
}

func (g *JwtAuthnGenerator) GenFilterConfig() (proto.Message, error) {
	providers := make(map[string]*jwtpb.JwtProvider)
	for _, provider := range g.AuthConfig.GetProviders() {
		addr, err := util.ExtractAddressFromURI(provider.GetJwksUri())
		if err != nil {
			return nil, fmt.Errorf("for provider (%v), failed to parse JWKS URI: %v", provider.Id, err)
		}
		clusterName := util.JwtProviderClusterName(addr)
		fromHeaders, fromParams, err := processJwtLocations(provider)
		if err != nil {
			return nil, err
		}

		jwks := &jwtpb.RemoteJwks{
			HttpUri: &corepb.HttpUri{
				Uri: provider.GetJwksUri(),
				HttpUpstreamType: &corepb.HttpUri_Cluster{
					Cluster: clusterName,
				},
				Timeout: durationpb.New(g.HttpRequestTimeout),
			},
			CacheDuration: &durationpb.Duration{
				Seconds: int64(g.JwksCacheDurationInS),
			},
		}
		if !g.DisableJwksAsyncFetch {
			jwks.AsyncFetch = &jwtpb.JwksAsyncFetch{
				FastListener: g.JwksAsyncFetchFastListener,
			}
		}
		if g.JwksFetchNumRetries > 0 {
			// only create a retry policy, evenutally with a backoff if it is required.
			rp := &corepb.RetryPolicy{
				NumRetries: &wrapperspb.UInt32Value{
					Value: uint32(g.JwksFetchNumRetries),
				},
				RetryBackOff: &corepb.BackoffStrategy{
					BaseInterval: durationpb.New(g.JwksFetchRetryBackOffBaseInterval),
					MaxInterval:  durationpb.New(g.JwksFetchRetryBackOffMaxInterval),
				},
			}
			jwks.RetryPolicy = rp
		}

		jp := &jwtpb.JwtProvider{
			Issuer: provider.GetIssuer(),
			JwksSourceSpecifier: &jwtpb.JwtProvider_RemoteJwks{
				RemoteJwks: jwks,
			},
			FromHeaders:             fromHeaders,
			FromParams:              fromParams,
			ForwardPayloadHeader:    g.GeneratedHeaderPrefix + util.JwtAuthnForwardPayloadHeaderSuffix,
			Forward:                 true,
			PadForwardPayloadHeader: g.JwtPadForwardPayloadHeader,
		}

		if len(provider.GetAudiences()) != 0 {
			for _, a := range strings.Split(provider.GetAudiences(), ",") {
				jp.Audiences = append(jp.Audiences, strings.TrimSpace(a))
			}
		} else if !g.DisableJwtAudienceServiceNameCheck {
			// No providers specified by user.
			// For backwards-compatibility with ESPv1, auto-generate audiences.
			// See b/147834348 for more information on this default behavior.
			defaultAudience := fmt.Sprintf("https://%v", g.ServiceName)
			jp.Audiences = append(jp.Audiences, defaultAudience)
		}

		if g.JwtCacheSize > 0 {
			jp.JwtCacheConfig = &jwtpb.JwtCacheConfig{
				JwtCacheSize: uint32(g.JwtCacheSize),
			}
		}

		// TODO(taoxuy): add unit test
		// the JWT Payload will be send to metadata by envoy and it will be used by service control filter
		// for logging and setting credential_id
		jp.PayloadInMetadata = util.JwtPayloadMetadataName
		providers[provider.GetId()] = jp
	}

	requirements := make(map[string]*jwtpb.JwtRequirement)
	for _, rule := range g.AuthConfig.GetRules() {
		if len(rule.GetRequirements()) > 0 {
			requirements[rule.GetSelector()] = makeJwtRequirement(rule.GetRequirements(), rule.GetAllowWithoutCredential())
		}
	}

	return &jwtpb.JwtAuthentication{
		Providers:      providers,
		RequirementMap: requirements,
	}, nil
}

func defaultJwtLocations() ([]*jwtpb.JwtHeader, []string, error) {
	return []*jwtpb.JwtHeader{
			{
				Name:        util.DefaultJwtHeaderNameAuthorization,
				ValuePrefix: util.DefaultJwtHeaderValuePrefixBearer,
			},
			{
				Name: util.DefaultJwtHeaderNameXGoogleIapJwtAssertion,
			},
		}, []string{
			util.DefaultJwtQueryParamAccessToken,
		}, nil
}

func processJwtLocations(provider *confpb.AuthProvider) ([]*jwtpb.JwtHeader, []string, error) {
	if len(provider.JwtLocations) == 0 {
		return defaultJwtLocations()
	}

	jwtHeaders := []*jwtpb.JwtHeader{}
	jwtParams := []string{}

	for _, jwtLocation := range provider.JwtLocations {
		switch x := jwtLocation.In.(type) {
		case *confpb.JwtLocation_Header:
			jwtHeaders = append(jwtHeaders, &jwtpb.JwtHeader{
				Name:        jwtLocation.GetHeader(),
				ValuePrefix: jwtLocation.GetValuePrefix(),
			})
		case *confpb.JwtLocation_Query:
			jwtParams = append(jwtParams, jwtLocation.GetQuery())
		default:
			// TODO(b/176432170): Handle errors here, prevent startup.
			glog.Errorf("error processing JWT location for provider (%v): unexpected type %T", provider.Id, x)
			continue
		}
	}
	return jwtHeaders, jwtParams, nil
}

func makeJwtRequirement(requirements []*confpb.AuthRequirement, allow_missing bool) *jwtpb.JwtRequirement {
	// By default, if there are multi requirements, treat it as RequireAny.
	requires := &jwtpb.JwtRequirement{
		RequiresType: &jwtpb.JwtRequirement_RequiresAny{
			RequiresAny: &jwtpb.JwtRequirementOrList{},
		},
	}

	for _, r := range requirements {
		var require *jwtpb.JwtRequirement
		if r.GetAudiences() == "" {
			require = &jwtpb.JwtRequirement{
				RequiresType: &jwtpb.JwtRequirement_ProviderName{
					ProviderName: r.GetProviderId(),
				},
			}
		} else {
			// Note: Audiences in requirements is deprecated.
			// But if it's specified, we should override the audiences for the provider.
			var audiences []string
			for _, a := range strings.Split(r.GetAudiences(), ",") {
				audiences = append(audiences, strings.TrimSpace(a))
			}
			require = &jwtpb.JwtRequirement{
				RequiresType: &jwtpb.JwtRequirement_ProviderAndAudiences{
					ProviderAndAudiences: &jwtpb.ProviderWithAudiences{
						ProviderName: r.GetProviderId(),
						Audiences:    audiences,
					},
				},
			}
		}
		if len(requirements) == 1 && !allow_missing {
			requires = require
		} else {
			requires.GetRequiresAny().Requirements = append(requires.GetRequiresAny().GetRequirements(), require)
		}
	}
	if allow_missing {
		require := &jwtpb.JwtRequirement{
			RequiresType: &jwtpb.JwtRequirement_AllowMissing{
				AllowMissing: &emptypb.Empty{},
			},
		}
		requires.GetRequiresAny().Requirements = append(requires.GetRequiresAny().GetRequirements(), require)
	}

	return requires
}

// GetAuthRequiredSelectorsFromOPConfig returns a list of selectors that require
// per-method level authn config.
func GetAuthRequiredSelectorsFromOPConfig(serviceConfig *confpb.Service, opts options.ConfigGeneratorOptions) (map[string]bool, error) {
	authRequiredMethods := make(map[string]bool)

	auth := serviceConfig.GetAuthentication()
	for _, rule := range auth.GetRules() {
		selector := rule.GetSelector()
		if util.ShouldSkipOPDiscoveryAPI(selector, opts.AllowDiscoveryAPIs) {
			glog.Warningf("Skip Auth rule %q because discovery API is not supported.", selector)
			continue
		}

		if len(rule.GetRequirements()) == 0 {
			continue
		}

		authRequiredMethods[selector] = true
	}

	return authRequiredMethods, nil
}
