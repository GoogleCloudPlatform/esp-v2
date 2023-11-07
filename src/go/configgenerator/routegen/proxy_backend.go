package routegen

import (
	"fmt"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/routegen/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// ProxyBackendGenerator is a RouteGenerator to configure routes to the local
// or remote backend service.
type ProxyBackendGenerator struct {
	HTTPPatterns             httppattern.MethodSlice
	BackendClusterBySelector map[string]*BackendClusterSpecifier
	BackendRouteGen          *helpers.BackendRouteGenerator

	*NoopRouteGenerator
}

// NewProxyBackendRouteGenFromOPConfig creates ProxyBackendGenerator
// from OP service config + ESPv2 options.
// It is a RouteGeneratorOPFactory.
func NewProxyBackendRouteGenFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) (RouteGenerator, error) {
	httpPatternsBySelector, err := ParseHTTPPatternsBySelectorFromOPConfig(serviceConfig, opts)
	if err != nil {
		return nil, fmt.Errorf("fail to parse http patterns from OP config: %v", err)
	}

	httpPatterns, err := sortHttpPatterns(httpPatternsBySelector)
	if err != nil {
		return nil, fmt.Errorf("fail to sort http patterns: %v", err)
	}

	backendClusterBySelector, err := ParseBackendClusterBySelectorFromOPConfig(serviceConfig, opts)
	if err != nil {
		return nil, fmt.Errorf("fail to parse backend cluster specifiers from OP config: %v", err)
	}

	return &ProxyBackendGenerator{
		HTTPPatterns:             *httpPatterns,
		BackendClusterBySelector: backendClusterBySelector,
		BackendRouteGen:          helpers.NewBackendRouteGeneratorFromOPConfig(opts),
	}, nil
}

// RouteType implements interface RouteGenerator.
func (g *ProxyBackendGenerator) RouteType() string {
	return "backend_routes"
}

// GenRouteConfig implements interface RouteGenerator.
func (g *ProxyBackendGenerator) GenRouteConfig(filterGens []filtergen.FilterGenerator) ([]*routepb.Route, error) {
	var routes []*routepb.Route
	for _, httpPattern := range g.HTTPPatterns {
		selector := httpPattern.Operation

		backendCluster, ok := g.BackendClusterBySelector[selector]
		if !ok {
			return nil, fmt.Errorf("could not find any backend cluster for selector %q", selector)
		}

		methodCfg := &helpers.MethodCfg{
			OperationName:      selector,
			BackendClusterName: backendCluster.Name,
			HostRewrite:        backendCluster.HostName,
			// TODO(nareddyt): Set deadline and idle timeouts from backend rule
			Deadline:    util.DefaultResponseDeadline,
			HTTPPattern: httpPattern.Pattern,
		}

		if backendCluster.HTTPBackend != nil {
			// Special support for HTTP backend.
			isGrpc, err := httpPattern.IsGRPCPathForOperation(selector)
			if err != nil {
				return nil, err
			}

			if !isGrpc {
				methodCfg.BackendClusterName = backendCluster.HTTPBackend.Name
				methodCfg.HostRewrite = backendCluster.HTTPBackend.HostName
			}
		}

		methodRoutes, err := g.BackendRouteGen.GenRoutesForMethod(methodCfg, filterGens)
		if err != nil {
			return nil, fmt.Errorf("fail to generate routes for operation %q with HTTP pattern %q: %v", selector, httpPattern.String(), err)
		}
		routes = append(routes, methodRoutes...)
	}

	return routes, nil
}

// AffectedHTTPPatterns implements interface RouteGenerator.
func (g *ProxyBackendGenerator) AffectedHTTPPatterns() httppattern.MethodSlice {
	return g.HTTPPatterns
}

// sortHttpPatterns implements go/esp-v2-route-match-ordering-implementation.
// Sorting is needed for route match correctness.
//
// Forked from `route_generator.go: sortHttpPatterns()`
func sortHttpPatterns(httpPatternsBySelector map[string][]*httppattern.Pattern) (*httppattern.MethodSlice, error) {
	httpPatternMethods := &httppattern.MethodSlice{}
	for selector, httpPatterns := range httpPatternsBySelector {
		for _, httpPattern := range httpPatterns {
			httpPatternMethods.AppendMethod(&httppattern.Method{
				Pattern:   httpPattern,
				Operation: selector,
			})
		}
	}

	if err := httppattern.Sort(httpPatternMethods); err != nil {
		return nil, err
	}

	return httpPatternMethods, nil
}

//func (g *ProxyBackendGenerator) determineBackendCluster(selector string) (*BackendClusterSpecifier, error) {
//	httpBackendSelector, ok := g.HTTPBackendClusterBySelector[selector]
//	if ok {
//		return httpBackendSelector, nil
//	}
//
//	backendSelector, ok := g.BackendClusterBySelector[selector]
//	if !ok {
//		return nil, fmt.Errorf("could not find any backend cluster for selector %q", selector)
//	}
//
//	return backendSelector, nil
//}
