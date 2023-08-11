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

// CORSGenerator is a FilterGenerator to configure CORS config.
type CORSGenerator struct {
	Preset string
	// AllowOrigin should only be set if preset=basic
	AllowOrigin string
	// AllowOriginRegex should only be set if preset=cors_with_regex
	AllowOriginRegex string
	MaxAge           time.Duration
	AllowMethods     string
	AllowHeaders     string
	ExposeHeaders    string
	AllowCredentials bool

	NoopFilterGenerator
}

// NewCORSFilterGensFromOPConfig creates a CORSGenerator from
// OP service config + descriptor + ESPv2 options. It is a FilterGeneratorOPFactory.
func NewCORSFilterGensFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) ([]FilterGenerator, error) {
	if opts.CorsPreset == "" {
		glog.Infof("Not adding CORS filter gen because the feature is disabled by option, option is currently %q", opts.CorsPreset)
		return nil, nil
	}

	return []FilterGenerator{
		&CORSGenerator{
			Preset:           opts.CorsPreset,
			AllowOrigin:      opts.CorsAllowOrigin,
			AllowOriginRegex: opts.CorsAllowOriginRegex,
			MaxAge:           opts.CorsMaxAge,
			AllowMethods:     opts.CorsAllowMethods,
			AllowHeaders:     opts.CorsAllowHeaders,
			ExposeHeaders:    opts.CorsExposeHeaders,
			AllowCredentials: opts.CorsAllowCredentials,
		},
	}, nil
}

func (g *CORSGenerator) FilterName() string {
	return CORSFilterName
}

func (g *CORSGenerator) GenFilterConfig() (proto.Message, error) {
	return &corspb.Cors{}, nil
}

func (g *CORSGenerator) GenPerHostConfig(vHostName string) (proto.Message, error) {
	policy := &corspb.CorsPolicy{
		MaxAge:        strconv.Itoa(int(g.MaxAge.Seconds())),
		AllowMethods:  g.AllowMethods,
		AllowHeaders:  g.AllowHeaders,
		ExposeHeaders: g.ExposeHeaders,
		AllowCredentials: &wrapperspb.BoolValue{
			Value: g.AllowCredentials,
		},
	}

	switch g.Preset {
	case "basic":
		policy.AllowOriginStringMatch = []*matcherpb.StringMatcher{
			{
				MatchPattern: &matcherpb.StringMatcher_Exact{
					Exact: g.AllowOrigin,
				},
			},
		}

	case "cors_with_regex":
		policy.AllowOriginStringMatch = []*matcherpb.StringMatcher{
			{
				MatchPattern: &matcherpb.StringMatcher_SafeRegex{
					SafeRegex: &matcherpb.RegexMatcher{
						Regex: g.AllowOriginRegex,
					},
				},
			},
		}

	default:
		return nil, fmt.Errorf(`cors_preset must be either "basic" or "cors_with_regex"`)
	}

	return policy, nil
}
