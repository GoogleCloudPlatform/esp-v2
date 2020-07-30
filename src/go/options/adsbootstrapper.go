// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package options

import (
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
)

// AdsBootstrapperOptions describes the possible overrides used by the ADS bootstrapper to create the envoy bootstrap config.
type AdsBootstrapperOptions struct {
	CommonOptions

	// Flags for ADS
	AdsConnectTimeout time.Duration
	DiscoveryAddress  string
}

// DefaultAdsBootstrapperOptions returns AdsBootstrapperOptions with default values.
//
// The default values are expected to match the default values from the flags.
func DefaultAdsBootstrapperOptions() AdsBootstrapperOptions {
	return AdsBootstrapperOptions{
		CommonOptions:     DefaultCommonOptions(),
		AdsConnectTimeout: 10 * time.Second,
		DiscoveryAddress:  fmt.Sprintf("%s:8790", util.LoopbackIPv4Addr),
	}
}
