package helpers

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

import (
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	commonpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v12/http/common"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"google.golang.org/protobuf/types/known/durationpb"
)

type FilterAccessTokenConfiger struct {
	HttpRequestTimeout time.Duration

	// Non-GCP deployment options.
	ServiceAccountKey string
	TokenAgentPort    uint

	// GCP deployment options.
	MetadataURL string
}

// NewFilterAccessTokenConfigerFromOPConfig creates a FilterAccessTokenConfiger from
// OP service config + descriptor + ESPv2 options.
func NewFilterAccessTokenConfigerFromOPConfig(opts options.ConfigGeneratorOptions) *FilterAccessTokenConfiger {
	return &FilterAccessTokenConfiger{
		HttpRequestTimeout: opts.HttpRequestTimeout,
		ServiceAccountKey:  opts.ServiceAccountKey,
		TokenAgentPort:     opts.TokenAgentPort,
		MetadataURL:        opts.MetadataURL,
	}
}

// MakeAccessTokenConfig creates the correct config to fetch an access token.
//
// Replaces ServiceInfo::processAccessToken.
func (c *FilterAccessTokenConfiger) MakeAccessTokenConfig() *commonpb.AccessToken {
	if c.ServiceAccountKey != "" {
		return &commonpb.AccessToken{
			TokenType: &commonpb.AccessToken_RemoteToken{
				RemoteToken: &commonpb.HttpUri{
					// Use http://127.0.0.1:8791/local/access_token by default.
					Uri:     fmt.Sprintf("http://%s:%v%s", util.LoopbackIPv4Addr, c.TokenAgentPort, util.TokenAgentAccessTokenPath),
					Cluster: clustergen.TokenAgentClusterName,
					Timeout: durationpb.New(c.HttpRequestTimeout),
				},
			},
		}
	}

	return &commonpb.AccessToken{
		TokenType: &commonpb.AccessToken_RemoteToken{
			RemoteToken: &commonpb.HttpUri{
				Uri:     fmt.Sprintf("%s%s", c.MetadataURL, util.AccessTokenPath),
				Cluster: clustergen.MetadataServerClusterName,
				Timeout: durationpb.New(c.HttpRequestTimeout),
			},
		},
	}
}
