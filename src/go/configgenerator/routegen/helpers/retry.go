package helpers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	onRetriableStatusCodes = "retriable-status-codes"
)

// RouteRetryConfiger is a helper to add backend retry policy to the route.
type RouteRetryConfiger struct {
	RetryOns           string
	RetryNum           uint
	RetryOnStatusCodes string
	PerTryTimeout      time.Duration
}

// NewRouteRetryConfigerFromOPConfig creates a RouteRetryConfiger from
// ESPv2 options.
func NewRouteRetryConfigerFromOPConfig(opts options.ConfigGeneratorOptions) *RouteRetryConfiger {
	return &RouteRetryConfiger{
		RetryOns:           opts.BackendRetryOns,
		RetryNum:           opts.BackendRetryNum,
		RetryOnStatusCodes: opts.BackendRetryOnStatusCodes,
		PerTryTimeout:      opts.BackendPerTryTimeout,
	}
}

// MaybeAddRetryPolicy adds the generated Retry config to the route action.
func MaybeAddRetryPolicy(c *RouteRetryConfiger, routeAction *routepb.RouteAction) error {
	if c == nil {
		return nil
	}

	retryPolicy, err := c.MakeRetryConfig()
	if err != nil {
		return fmt.Errorf("fail to create backend retry policy for routeAction: %v", err)
	}

	routeAction.RetryPolicy = retryPolicy
	return nil
}

// MakeRetryConfig creates the backend retry config.
func (c *RouteRetryConfiger) MakeRetryConfig() (*routepb.RetryPolicy, error) {
	retryOns := c.RetryOns
	retryNum := c.RetryNum
	perTryTimeout := c.PerTryTimeout
	var retriableStatusCodes []uint32

	if c.RetryOnStatusCodes != "" {
		var err error
		retriableStatusCodes, err = parseRetriableStatusCodes(c.RetryOnStatusCodes)
		if err != nil {
			return nil, fmt.Errorf("invalid retriable status codes: %v", err)
		}

		if retryOns == "" {
			retryOns = onRetriableStatusCodes
		} else if !strings.Contains(retryOns, onRetriableStatusCodes) {
			retryOns = retryOns + "," + onRetriableStatusCodes
		}
	}

	retryPolicy := &routepb.RetryPolicy{
		RetryOn: retryOns,
		NumRetries: &wrapperspb.UInt32Value{
			Value: uint32(retryNum),
		},
		RetriableStatusCodes: retriableStatusCodes,
	}

	if c.PerTryTimeout.Nanoseconds() > 0 {
		retryPolicy.PerTryTimeout = durationpb.New(perTryTimeout)
	}

	return retryPolicy, nil
}

func parseRetriableStatusCodes(statusCodes string) ([]uint32, error) {
	codeList := strings.Split(statusCodes, ",")
	var codes []uint32
	for _, codeStr := range codeList {
		if code, err := strconv.Atoi(codeStr); err != nil || code < 100 || code >= 600 {
			return nil, fmt.Errorf("invalid http status codes: %v, the valid one should be a number in [100, 600)", code)
		} else {
			codes = append(codes, uint32(code))
		}
	}
	return codes, nil
}
