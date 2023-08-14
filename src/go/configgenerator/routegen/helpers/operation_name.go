package helpers

import (
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const ()

const (
	headerKeySuffix = "Api-Operation-Name"
)

// RouteOperationNameConfiger is a helper to add the ESPv2 operation name to the
// upstream request.
type RouteOperationNameConfiger struct {
	GeneratedHeaderPrefix string
}

// NewRouteOperationNameConfigerFromOPConfig creates a RouteOperationNameConfiger from
// ESPv2 options.
func NewRouteOperationNameConfigerFromOPConfig(opts options.ConfigGeneratorOptions) *RouteOperationNameConfiger {
	if !opts.EnableOperationNameHeader {
		return nil
	}

	return &RouteOperationNameConfiger{
		GeneratedHeaderPrefix: opts.GeneratedHeaderPrefix,
	}
}

// MaybeAddOperationNameHeader adds the generated operation name config to the
// route.
func MaybeAddOperationNameHeader(c *RouteOperationNameConfiger, route *routepb.Route, operation string) {
	if c == nil {
		return
	}

	route.RequestHeadersToAdd = c.MakeOperationNameConfig(operation)
}

// MakeOperationNameConfig creates the response headers to add to the route.
func (c *RouteOperationNameConfiger) MakeOperationNameConfig(operation string) []*corepb.HeaderValueOption {
	return []*corepb.HeaderValueOption{
		{
			Header: &corepb.HeaderValue{
				Key:   c.GeneratedHeaderPrefix + headerKeySuffix,
				Value: operation,
			},
			Append: &wrapperspb.BoolValue{
				Value: false,
			},
		},
	}
}
