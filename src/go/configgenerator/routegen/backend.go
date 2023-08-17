package routegen

import (
	"fmt"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/routegen/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// BackendGenerator is a RouteGenerator to configure routes to the local
// or remote backend service.
type BackendGenerator struct {
	HTTPPatterns             *httppattern.MethodSlice
	BackendClusterBySelector map[string]*BackendClusterSpecifier
	BackendRouteGen          *helpers.BackendRouteGenerator
}

// NewBackendRouteGensFromOPConfig creates BackendGenerator
// from OP service config + descriptor + ESPv2 options.
// It is a RouteGeneratorOPFactory.
func NewBackendRouteGensFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) ([]RouteGenerator, error) {
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

	return []RouteGenerator{
		&BackendGenerator{
			HTTPPatterns:             httpPatterns,
			BackendClusterBySelector: backendClusterBySelector,
			BackendRouteGen:          helpers.NewBackendRouteGeneratorFromOPConfig(opts),
		},
	}, nil
}

// GenRouteConfig implements interface RouteGenerator.
func (g *BackendGenerator) GenRouteConfig() ([]*routepb.Route, error) {
	var routes []*routepb.Route
	for _, httpPattern := range *g.HTTPPatterns {
		selector := httpPattern.Operation
		backendCluster := g.BackendClusterBySelector[selector]
		if backendCluster == nil {
			return nil, fmt.Errorf("could not find any backend cluster for selector %q", selector)
		}

		methodCfg := &helpers.MethodCfg{
			OperationName:      selector,
			BackendClusterName: backendCluster.Name,
			HostRewrite:        backendCluster.HostName,
			// TODO(nareddyt)
			Deadline:    util.DefaultResponseDeadline,
			HTTPPattern: httpPattern.Pattern,
		}
		methodRoutes, err := g.BackendRouteGen.GenRoutesForMethod(methodCfg)
		if err != nil {
			return nil, fmt.Errorf("fail to generate routes for operation %q: %v", selector, err)
		}
		routes = append(routes, methodRoutes...)
	}

	return routes, nil
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
