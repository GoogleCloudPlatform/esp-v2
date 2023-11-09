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
}

// NewRouteDeadlineConfigerFromOPConfig creates a RouteDeadlineConfiger from
// ESPv2 options.
func NewRouteDeadlineConfigerFromOPConfig(opts options.ConfigGeneratorOptions) *RouteDeadlineConfiger {
	return &RouteDeadlineConfiger{
		GlobalStreamIdleTimeout: opts.StreamIdleTimeout,
	}
}

// MaybeAddDeadlines adds the generated deadline config to the route action.
func MaybeAddDeadlines(c *RouteDeadlineConfiger, routeAction *routepb.RouteAction, deadline time.Duration, isStreaming bool) {
	if c == nil {
		return
	}

	newDeadline, idleTimeout := c.CalcIdleTimeout(deadline, isStreaming)
	routeAction.Timeout = durationpb.New(newDeadline)
	routeAction.IdleTimeout = durationpb.New(idleTimeout)
}

// CalcIdleTimeout will return the correct idle timeout based on method properties.
//
// Forked from `service_info.go`
func (c *RouteDeadlineConfiger) CalcIdleTimeout(deadline time.Duration, isStreaming bool) (time.Duration, time.Duration) {
	// Response timeouts are not compatible with streaming methods (documented in Envoy).
	// This applies to methods with a streaming upstream OR downstream.
	var idleTimeout time.Duration
	if isStreaming {
		if deadline <= 0 {
			// When the backend deadline is unspecified , calculate the streamIdleTimeout based on max{defaultTimeout, globalStreamIdleTimeout} .
			idleTimeout = calculateStreamIdleTimeout(util.DefaultResponseDeadline, c.GlobalStreamIdleTimeout)
		} else {
			// User configured deadline serves as the stream idle timeout.
			idleTimeout = deadline
		}

		return 0, idleTimeout
	}

	if deadline <= 0 {
		// If no deadline specified by the user, explicitly use default.
		deadline = util.DefaultResponseDeadline
	}

	// Allow per-route response deadlines to override the global stream idle timeout.
	idleTimeout = calculateStreamIdleTimeout(deadline, c.GlobalStreamIdleTimeout)
	return deadline, idleTimeout
}

// Calculates the stream idle timeout based on the response deadline for that route and the global stream idle timeout.
//
// Forked from `service_info.go`
func calculateStreamIdleTimeout(operationDeadline time.Duration, streamIdleTimeout time.Duration) time.Duration {
	// If the deadline and stream idle timeout have the exact same timeout,
	// the error code returned to the client is inconsistent based on which event is processed first.
	// (504 for response deadline, 408 for idle timeout)
	// So offset the idle timeout to ensure response deadline is always hit first.
	operationIdleTimeout := operationDeadline + time.Second
	return util.MaxDuration(operationIdleTimeout, streamIdleTimeout)
}
