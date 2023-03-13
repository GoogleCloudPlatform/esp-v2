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
