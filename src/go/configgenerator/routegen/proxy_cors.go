package routegen

import (
	"fmt"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/glog"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// ProxyCORSGenerator is a RouteGenerator to autogenerate CORS routes that are
// proxied to the backend.
type ProxyCORSGenerator struct {
	DisallowColonInWildcardPathSegment      bool
	HealthCheckAutogeneratedOperationPrefix string

	// ProxyCORSGenerator wraps ProxyBackendGenerator directly.
	*ProxyBackendGenerator
}

// NewProxyCORSRouteGenFromOPConfig creates ProxyCORSGenerator
// from OP service config + ESPv2 options.
// It is a RouteGeneratorOPFactory.
func NewProxyCORSRouteGenFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) (RouteGenerator, error) {
	allowCors := false
	for _, endpoint := range serviceConfig.GetEndpoints() {
		if endpoint.GetName() == serviceConfig.GetName() && endpoint.GetAllowCors() {
			allowCors = true
		}
	}
	if !allowCors {
		glog.Infof("Not adding Proxy CORS route gen because the feature is disabled by config.")
		return nil, nil
	}

	backendGen, err := NewProxyBackendRouteGenFromOPConfig(serviceConfig, opts)
	if err != nil {
		return nil, fmt.Errorf("while creating Proxy CORS routes, could not create underlying wrap route gen, failed with error: %v", err)
	}

	return &ProxyCORSGenerator{
		DisallowColonInWildcardPathSegment:      opts.DisallowColonInWildcardPathSegment,
		HealthCheckAutogeneratedOperationPrefix: opts.HealthCheckAutogeneratedOperationPrefix,
		ProxyBackendGenerator:                   backendGen.(*ProxyBackendGenerator),
	}, nil
}

// RouteType implements interface RouteGenerator.
func (g *ProxyCORSGenerator) RouteType() string {
	return "cors_routes"
}

// GenRouteConfig implements interface RouteGenerator.
func (g *ProxyCORSGenerator) GenRouteConfig(filterGens []filtergen.FilterGenerator) ([]*routepb.Route, error) {
	backendHTTPPatterns := g.ProxyBackendGenerator.AffectedHTTPPatterns()

	var corsHTTPPatterns httppattern.MethodSlice
	seenUriTemplatesInRoute := make(map[string]bool)
	for _, httpPattern := range backendHTTPPatterns {

		if httpPattern.HttpMethod != util.OPTIONS {
			uriTemplate, err := httppattern.ParseUriTemplate(httpPattern.UriTemplate.Origin)
			if err != nil {
				return nil, fmt.Errorf("error parsing URI template for http rule for operation (%v): %v", httpPattern.Operation, err)
			}

			dedupUriTemplate := uriTemplate.Regex(g.DisallowColonInWildcardPathSegment)
			if ok, _ := seenUriTemplatesInRoute[dedupUriTemplate]; !ok {
				seenUriTemplatesInRoute[dedupUriTemplate] = true

				originalSelector := httpPattern.Operation
				methodShortName, err := util.SelectorToMethodName(originalSelector)
				if err != nil {
					return nil, err
				}
				apiName, err := util.SelectorToAPIName(originalSelector)
				if err != nil {
					return nil, err
				}
				genOperation := fmt.Sprintf("%s.%s_CORS_%s", apiName, g.HealthCheckAutogeneratedOperationPrefix, methodShortName)

				// Shallow clone.
				corsHTTPPatterns = append(corsHTTPPatterns, &httppattern.Method{
					Pattern: &httppattern.Pattern{
						HttpMethod:  util.OPTIONS,
						UriTemplate: httpPattern.UriTemplate,
					},
					Operation: genOperation,
				})

				// Update backend selectors. This ensures CORS routes are proxied to
				// remote backend clusters.
				cluster := g.ProxyBackendGenerator.BackendClusterBySelector[originalSelector]
				g.ProxyBackendGenerator.BackendClusterBySelector[genOperation] = cluster
			}
		}
	}

	// Override the backend HTTP patterns, so proxy CORS routes are generated.
	g.ProxyBackendGenerator.HTTPPatterns = corsHTTPPatterns
	return g.ProxyBackendGenerator.GenRouteConfig(filterGens)
}
