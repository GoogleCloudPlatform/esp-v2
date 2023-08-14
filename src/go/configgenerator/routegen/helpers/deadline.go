package helpers

import (
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"google.golang.org/protobuf/types/known/durationpb"
)

// RouteDeadlineConfiger is a helper to configure deadlines and timeouts on
// backend routes.
type RouteDeadlineConfiger struct {
	GlobalStreamIdleTimeout time.Duration
	// TODO(nareddyt): Options to disable this for gRPC streaming methods, etc.
}

// NewRouteDeadlineConfigerFromOPConfig creates a RouteDeadlineConfiger from
// ESPv2 options.
func NewRouteDeadlineConfigerFromOPConfig(opts options.ConfigGeneratorOptions) *RouteDeadlineConfiger {
	return &RouteDeadlineConfiger{
		GlobalStreamIdleTimeout: opts.StreamIdleTimeout,
	}
}

// MaybeAddDeadlines adds the generated deadline config to the route action.
func MaybeAddDeadlines(c *RouteDeadlineConfiger, routeAction *routepb.RouteAction, deadline time.Duration) {
	if c == nil {
		return
	}

	streamIdleTimeout := calculateStreamIdleTimeout(deadline, c.GlobalStreamIdleTimeout)
	routeAction.Timeout = durationpb.New(deadline)
	routeAction.IdleTimeout = durationpb.New(streamIdleTimeout)
}

// Calculates the stream idle timeout based on the response deadline for that route and the global stream idle timeout.
func calculateStreamIdleTimeout(operationDeadline time.Duration, streamIdleTimeout time.Duration) time.Duration {
	// If the deadline and stream idle timeout have the exact same timeout,
	// the error code returned to the client is inconsistent based on which event is processed first.
	// (504 for response deadline, 408 for idle timeout)
	// So offset the idle timeout to ensure response deadline is always hit first.
	operationIdleTimeout := operationDeadline + time.Second
	return util.MaxDuration(operationIdleTimeout, streamIdleTimeout)
}
