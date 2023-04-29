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

package helpers

import (
	"strings"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
)

const (
	discoveryAPIPrefix = "google.discovery"
)

func IsOPDiscoveryAPI(operationName string) bool {
	return strings.HasPrefix(operationName, discoveryAPIPrefix)
}

func ShouldSkipOPDiscoveryAPI(operation string, opts options.ConfigGeneratorOptions) bool {
	return IsOPDiscoveryAPI(operation) && !opts.AllowDiscoveryAPIs
}
