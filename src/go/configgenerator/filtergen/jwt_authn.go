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

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"

	ci "github.com/GoogleCloudPlatform/esp-v2/src/go/configinfo"

	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	jwtpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	durationpb "github.com/golang/protobuf/ptypes/duration"
	emptypb "github.com/golang/protobuf/ptypes/empty"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

const (
	// JWTAuthnFilterName is the Envoy filter name for debug logging.
	JWTAuthnFilterName = "envoy.filters.http.jwt_authn"
)

type JwtAuthnGenerator struct {
	// skipFilter indicates if this filter is disabled based on options and config.
	skipFilter bool
}

// NewJwtAuthnGenerator creates the JwtAuthnGenerator with cached config.
func NewJwtAuthnGenerator(serviceInfo *ci.ServiceInfo) *JwtAuthnGenerator {
	if serviceInfo.Options.SkipJwtAuthnFilter {
		return &JwtAuthnGenerator{
			skipFilter: true,
		}
	}

	auth := serviceInfo.ServiceConfig().GetAuthentication()
	if len(auth.GetProviders()) == 0 {
		return &JwtAuthnGenerator{
			skipFilter: true,
		}
	}

	return &JwtAuthnGenerator{}
}

func (g *JwtAuthnGenerator) FilterName() string {
	return JWTAuthnFilterName
}

func (g *JwtAuthnGenerator) IsEnabled() bool {
	return !g.skipFilter
}

func (g *JwtAuthnGenerator) GenPerRouteConfig(method *ci.MethodInfo, httpRule *httppattern.Pattern) (proto.Message, error) {
	if !method.RequireAuth {
		return nil, nil
	}

	return &jwtpb.PerRouteConfig{
		RequirementSpecifier: &jwtpb.PerRouteConfig_RequirementName{
			RequirementName: method.Operation(),
		},
	}, nil
}

func (g *JwtAuthnGenerator) GenFilterConfig(serviceInfo *ci.ServiceInfo) (proto.Message, error) {
	auth := serviceInfo.ServiceConfig().GetAuthentication()
	providers := make(map[string]*jwtpb.JwtProvider)
	for _, provider := range auth.GetProviders() {
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
				Timeout: ptypes.DurationProto(serviceInfo.Options.HttpRequestTimeout),
			},
			CacheDuration: &durationpb.Duration{
				Seconds: int64(serviceInfo.Options.JwksCacheDurationInS),
			},
		}
		if !serviceInfo.Options.DisableJwksAsyncFetch {
			jwks.AsyncFetch = &jwtpb.JwksAsyncFetch{
				FastListener: serviceInfo.Options.JwksAsyncFetchFastListener,
			}
		}
		if serviceInfo.Options.JwksFetchNumRetries > 0 {
			// only create a retry policy, evenutally with a backoff if it is required.
			rp := &corepb.RetryPolicy{
				NumRetries: &wrapperspb.UInt32Value{
					Value: uint32(serviceInfo.Options.JwksFetchNumRetries),
				},
				RetryBackOff: &corepb.BackoffStrategy{
					BaseInterval: ptypes.DurationProto(serviceInfo.Options.JwksFetchRetryBackOffBaseInterval),
					MaxInterval:  ptypes.DurationProto(serviceInfo.Options.JwksFetchRetryBackOffMaxInterval),
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
			ForwardPayloadHeader:    serviceInfo.Options.GeneratedHeaderPrefix + util.JwtAuthnForwardPayloadHeaderSuffix,
			Forward:                 true,
			PadForwardPayloadHeader: serviceInfo.Options.JwtPadForwardPayloadHeader,
		}

		if len(provider.GetAudiences()) != 0 {
			for _, a := range strings.Split(provider.GetAudiences(), ",") {
				jp.Audiences = append(jp.Audiences, strings.TrimSpace(a))
			}
		} else if !serviceInfo.Options.DisableJwtAudienceServiceNameCheck {
			// No providers specified by user.
			// For backwards-compatibility with ESPv1, auto-generate audiences.
			// See b/147834348 for more information on this default behavior.
			defaultAudience := fmt.Sprintf("https://%v", serviceInfo.Name)
			jp.Audiences = append(jp.Audiences, defaultAudience)
		}

		if serviceInfo.Options.JwtCacheSize > 0 {
			jp.JwtCacheConfig = &jwtpb.JwtCacheConfig{
				JwtCacheSize: uint32(serviceInfo.Options.JwtCacheSize),
			}
		}

		// TODO(taoxuy): add unit test
		// the JWT Payload will be send to metadata by envoy and it will be used by service control filter
		// for logging and setting credential_id
		jp.PayloadInMetadata = util.JwtPayloadMetadataName
		providers[provider.GetId()] = jp
	}

	requirements := make(map[string]*jwtpb.JwtRequirement)
	for _, rule := range auth.GetRules() {
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
