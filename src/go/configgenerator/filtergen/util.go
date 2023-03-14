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
	"sort"

	commonpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v11/http/common"
)

func parseDepErrorBehavior(stringVal string) (commonpb.DependencyErrorBehavior, error) {
	depErrorBehaviorInt, ok := commonpb.DependencyErrorBehavior_value[stringVal]
	if !ok {
		keys := make([]string, 0, len(commonpb.DependencyErrorBehavior_value))
		for k := range commonpb.DependencyErrorBehavior_value {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return commonpb.DependencyErrorBehavior_UNSPECIFIED, fmt.Errorf("unknown value for DependencyErrorBehavior (%v), accepted values are: %+q", stringVal, keys)
	}
	return commonpb.DependencyErrorBehavior(depErrorBehaviorInt), nil
}
