package helpers

import (
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
)

const (
	headerKey   = "Strict-Transport-Security"
	headerValue = "max-age=31536000; includeSubdomains"
)

// RouteHSTSConfiger is a helper to enable strict transport security, preventing
// downstream from protocol downgrades to HTTP.
type RouteHSTSConfiger struct{}

// NewRouteHSTSConfigerFromOPConfig creates a RouteHSTSConfiger from
// ESPv2 options.
func NewRouteHSTSConfigerFromOPConfig(opts options.ConfigGeneratorOptions) *RouteHSTSConfiger {
	if !opts.EnableHSTS {
		return nil
	}

	return &RouteHSTSConfiger{}
}

// MaybeAddHSTSHeader adds the generated HSTS config to the route.
func MaybeAddHSTSHeader(c *RouteHSTSConfiger, route *routepb.Route) {
	if c == nil {
		return
	}

	route.ResponseHeadersToAdd = c.MakeHSTSConfig()
}

// MakeHSTSConfig creates the response headers to add to the route.
func (c *RouteHSTSConfiger) MakeHSTSConfig() []*corepb.HeaderValueOption {
	return []*corepb.HeaderValueOption{
		{
			Header: &corepb.HeaderValue{
				Key:   headerKey,
				Value: headerValue,
			},
		},
	}
}
