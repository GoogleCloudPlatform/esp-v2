package routegen

import (
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/glog"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// CORSPassThroughGenerator is a RouteGenerator to configure CORS requests to
// be passed through to the backend.
// Also known as legacy name "AllowCORS" in integration tests.
type CORSPassThroughGenerator struct{}

// NewCORSPassThroughRouteGensFromOPConfig creates CORSPassThroughGenerator
// from OP service config + descriptor + ESPv2 options.
// It is a RouteGeneratorOPFactory.
func NewCORSPassThroughRouteGensFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) ([]RouteGenerator, error) {
	if !isCORSPassThroughRequired(serviceConfig) {
		glog.Infof("Not adding cors pass through (allow_cors) route gen because the feature is disabled by OP service config.")
		return nil, nil
	}

	return []RouteGenerator{
		&CORSPassThroughGenerator{},
	}, nil
}

// GenRouteConfig implements interface RouteGenerator.
func (g *CORSPassThroughGenerator) GenRouteConfig() ([]*routepb.Route, error) {
	return nil, nil
}

// isCORSPassThroughRequired determines if the feature is enabled.
//
// Forked from `service_info.go: processEndpoints()`
func isCORSPassThroughRequired(serviceConfig *servicepb.Service) bool {
	for _, endpoint := range serviceConfig.GetEndpoints() {
		if endpoint.GetName() == serviceConfig.GetName() && endpoint.GetAllowCors() {
			return true
		}
	}

	return false
}
