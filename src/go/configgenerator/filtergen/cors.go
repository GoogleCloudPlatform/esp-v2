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
	"fmt"
	"strconv"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	corspb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/cors/v3"
	matcherpb "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/golang/glog"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	// CORSFilterName is the Envoy filter name for debug logging.
	CORSFilterName = "envoy.filters.http.cors"
)

type CORSBasicGenerator struct {
	// AllowOrigin is the name of the origin to allow through.
	AllowOrigin string
	Options     *CorsOptions

	NoopFilterGenerator
}

type CORSRegexGenerator struct {
	// AllowOriginRegex is the regex of the origins to allow through.
	AllowOriginRegex string
	Options          *CorsOptions

	NoopFilterGenerator
}

type CorsOptions struct {
	MaxAge           time.Duration
	AllowMethods     string
	AllowHeaders     string
	ExposeHeaders    string
	AllowCredentials bool
}

// NewCORSFilterGensFromOPConfig creates a CORSGenerator from
// OP service config + descriptor + ESPv2 options. It is a FilterGeneratorOPFactory.
func NewCORSFilterGensFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) ([]FilterGenerator, error) {
	corsOptions := &CorsOptions{
		MaxAge:           opts.CorsMaxAge,
		AllowMethods:     opts.CorsAllowMethods,
		AllowHeaders:     opts.CorsAllowHeaders,
		ExposeHeaders:    opts.CorsExposeHeaders,
		AllowCredentials: opts.CorsAllowCredentials,
	}

	switch opts.CorsPreset {
	case "":
		glog.Infof("Not adding CORS filter gen because the feature is disabled by option, option is currently %q", opts.CorsPreset)
		return nil, nil

	case "basic":
		return []FilterGenerator{
			&CORSBasicGenerator{
				AllowOrigin: opts.CorsAllowOrigin,
				Options:     corsOptions,
			},
		}, nil

	case "cors_with_regex":
		return []FilterGenerator{
			&CORSRegexGenerator{
				AllowOriginRegex: opts.CorsAllowOriginRegex,
				Options:          corsOptions,
			},
		}, nil

	default:
		return nil, fmt.Errorf(`cors_preset must be either "basic" or "cors_with_regex"`)
	}
}

func (g *CORSBasicGenerator) FilterName() string {
	return CORSFilterName
}

func (g *CORSBasicGenerator) GenFilterConfig() (proto.Message, error) {
	return &corspb.Cors{}, nil
}

func (g *CORSBasicGenerator) GenPerHostConfig(vHostName string) (proto.Message, error) {
	policy := genCorsPolicyFromOptions(g.Options)
	policy.AllowOriginStringMatch = []*matcherpb.StringMatcher{
		{
			MatchPattern: &matcherpb.StringMatcher_Exact{
				Exact: g.AllowOrigin,
			},
		},
	}
	return policy, nil
}

func (g *CORSRegexGenerator) FilterName() string {
	return CORSFilterName
}

func (g *CORSRegexGenerator) GenFilterConfig() (proto.Message, error) {
	return &corspb.Cors{}, nil
}

func (g *CORSRegexGenerator) GenPerHostConfig(vHostName string) (proto.Message, error) {
	policy := genCorsPolicyFromOptions(g.Options)
	policy.AllowOriginStringMatch = []*matcherpb.StringMatcher{
		{
			MatchPattern: &matcherpb.StringMatcher_SafeRegex{
				SafeRegex: &matcherpb.RegexMatcher{
					Regex: g.AllowOriginRegex,
				},
			},
		},
	}
	return policy, nil
}

func genCorsPolicyFromOptions(options *CorsOptions) *corspb.CorsPolicy {
	return &corspb.CorsPolicy{
		MaxAge:        strconv.Itoa(int(options.MaxAge.Seconds())),
		AllowMethods:  options.AllowMethods,
		AllowHeaders:  options.AllowHeaders,
		ExposeHeaders: options.ExposeHeaders,
		AllowCredentials: &wrapperspb.BoolValue{
			Value: options.AllowCredentials,
		},
	}
}
