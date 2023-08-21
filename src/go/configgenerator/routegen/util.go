package routegen

import (
	"fmt"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util/httppattern"
	"github.com/golang/glog"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// ParseSelectorsFromOPConfig returns a list of selectors in the config.
// Preserves original order of APIs in the service config.
func ParseSelectorsFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) []string {
	var selectors []string
	for _, api := range serviceConfig.GetApis() {
		if util.ShouldSkipOPDiscoveryAPI(api.GetName(), opts.AllowDiscoveryAPIs) {
			glog.Warningf("Skip API %q because discovery API is not supported.", api.GetName())
			continue
		}

		for _, method := range api.GetMethods() {
			selector := filtergen.MethodToSelector(api, method)
			selectors = append(selectors, selector)
		}
	}
	return selectors
}

// BackendClusterSpecifier specifies a local or remote backend cluster.
// If remote cluster, then HostRewrite will also be set.
type BackendClusterSpecifier struct {
	Name     string
	HostName string
}

// ParseBackendClusterBySelectorFromOPConfig parses the service config into a
// map of selector to the backend cluster to route to.
//
// Forks `service_info.go: processBackendRule()`
func ParseBackendClusterBySelectorFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) (map[string]*BackendClusterSpecifier, error) {
	selectors := ParseSelectorsFromOPConfig(serviceConfig, opts)
	backendRuleBySelector := PrecomputeBackendRuleBySelectorFromOPConfig(serviceConfig, opts)

	backendClusterBySelector := make(map[string]*BackendClusterSpecifier)
	for _, selector := range selectors {
		clusterSpecifier, err := determineBackendClusterForSelector(selector, backendRuleBySelector, serviceConfig, opts)
		if err != nil {
			return nil, fmt.Errorf("error determining backend cluster for operation %q: %v", selector, err)
		}
		backendClusterBySelector[selector] = clusterSpecifier
	}

	return backendClusterBySelector, nil
}

func determineBackendClusterForSelector(selector string, backendRuleBySelector map[string]*servicepb.BackendRule, serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) (*BackendClusterSpecifier, error) {
	localCluster := &BackendClusterSpecifier{
		Name: clustergen.MakeLocalBackendClusterName(serviceConfig),
	}

	if opts.EnableBackendAddressOverride {
		return localCluster, nil
	}

	backendRule, ok := backendRuleBySelector[selector]
	if !ok {
		return localCluster, nil
	}

	if backendRule.GetAddress() == "" {
		return localCluster, nil
	}

	_, hostname, port, _, err := util.ParseURI(backendRule.GetAddress())
	if err != nil {
		return nil, fmt.Errorf("error parsing remote backend rule's address: %v", err)
	}

	address := fmt.Sprintf("%v:%v", hostname, port)
	return &BackendClusterSpecifier{
		Name:     clustergen.RemoteAddressToClusterName(address),
		HostName: hostname,
	}, nil
}

// PrecomputeBackendRuleBySelectorFromOPConfig pre-processes the service config to
// return a map of selector to the corresponding backend rule.
func PrecomputeBackendRuleBySelectorFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) map[string]*servicepb.BackendRule {
	backendRuleBySelector := make(map[string]*servicepb.BackendRule)

	for _, rule := range serviceConfig.GetBackend().GetRules() {
		if util.ShouldSkipOPDiscoveryAPI(rule.GetSelector(), opts.AllowDiscoveryAPIs) {
			glog.Warningf("Skip backend rule %q because discovery API is not supported.", rule.GetSelector())
			continue
		}
		backendRuleBySelector[rule.GetSelector()] = rule
	}

	return backendRuleBySelector
}

// ParseHTTPPatternsBySelectorFromOPConfig parses the service config into a list
// of internal HTTP pattern representations, keyed by OP selector.
//
// Note: HTTP here refers to pattern serving as a route on the HTTP protocol.
// It does not differentiation between HTTP vs gRPC backend.
// By default, this function will generate HTTP patterns for gRPC backends.
//
// Forked from `service_info.go: processHttpRule()`
// and `service_info.go: addGrpcHttpRules()`
func ParseHTTPPatternsBySelectorFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) (map[string][]*httppattern.Pattern, error) {
	httpRuleBySelector := PrecomputeHTTPRuleBySelectorFromOPConfig(serviceConfig, opts)
	httpPatternsBySelector := make(map[string][]*httppattern.Pattern)

	for selector, rule := range httpRuleBySelector {
		pattern, err := httpRuleToHTTPPattern(rule)
		if err != nil {
			return nil, fmt.Errorf("fail to process http rule for operation %q: %v", selector, err)
		}
		httpPatternsBySelector[selector] = append(httpPatternsBySelector[selector], pattern)

		// additional_bindings cannot be nested inside themselves according to
		// https://aip.dev/127. Service Management will enforce this restriction
		// when interpret the http rules from the descriptor. Therefore, no need to
		// check for nested additional_bindings.
		for i, additionalRule := range rule.AdditionalBindings {
			additionalPattern, err := httpRuleToHTTPPattern(additionalRule)
			if err != nil {
				return nil, fmt.Errorf("fail to process http rule's additional_binding at index %d for operation %q: %v", i, selector, err)
			}
			httpPatternsBySelector[selector] = append(httpPatternsBySelector[selector], additionalPattern)
		}
	}

	isGRPCSupportRequired, err := filtergen.IsGRPCSupportRequiredForOPConfig(serviceConfig, opts)
	if err != nil {
		return nil, fmt.Errorf("fail to check if gRPC support is required: %v", err)
	}
	if !isGRPCSupportRequired {
		return httpPatternsBySelector, nil
	}

	// Add gRPC paths for gRPC backends.
	for _, api := range serviceConfig.GetApis() {
		if util.ShouldSkipOPDiscoveryAPI(api.GetName(), opts.AllowDiscoveryAPIs) {
			glog.Warningf("Skip API %q because discovery API is not supported.", api.GetName())
			continue
		}

		for _, method := range api.GetMethods() {
			selector := filtergen.MethodToSelector(api, method)
			gRPCPath := fmt.Sprintf("/%s/%s", api.GetName(), method.GetName())

			// For the OP config generated by api compiler, the path/uri template for grpc
			// method should always be valid.
			uriTemplate, err := httppattern.ParseUriTemplate(gRPCPath)
			if err != nil {
				return nil, fmt.Errorf("error parsing auto-generated gRPC http rule's URI template for operation %q: %v", selector, err)
			}

			pattern := &httppattern.Pattern{
				UriTemplate: uriTemplate,
				HttpMethod:  util.POST,
			}
			httpPatternsBySelector[selector] = append(httpPatternsBySelector[selector], pattern)
		}
	}

	return httpPatternsBySelector, nil
}

func httpRuleToHTTPPattern(rule *annotationspb.HttpRule) (*httppattern.Pattern, error) {
	parsedRule, err := parseHttpRule(rule)
	if err != nil {
		return nil, fmt.Errorf("fail to parse http rule: %v", err)
	}

	uriTemplate, err := httppattern.ParseUriTemplate(parsedRule.path)
	if err != nil {
		return nil, fmt.Errorf("fail to parse http rule path into uri template: %v", err)
	}

	return &httppattern.Pattern{
		HttpMethod:  parsedRule.method,
		UriTemplate: uriTemplate,
	}, nil
}

type httpRuleParseOutput struct {
	path   string
	method string
}

func parseHttpRule(rule *annotationspb.HttpRule) (*httpRuleParseOutput, error) {
	switch rule.GetPattern().(type) {
	case *annotationspb.HttpRule_Get:
		return &httpRuleParseOutput{
			path:   rule.GetGet(),
			method: util.GET,
		}, nil
	case *annotationspb.HttpRule_Put:
		return &httpRuleParseOutput{
			path:   rule.GetPut(),
			method: util.PUT,
		}, nil
	case *annotationspb.HttpRule_Post:
		return &httpRuleParseOutput{
			path:   rule.GetPost(),
			method: util.POST,
		}, nil
	case *annotationspb.HttpRule_Delete:
		return &httpRuleParseOutput{
			path:   rule.GetDelete(),
			method: util.DELETE,
		}, nil
	case *annotationspb.HttpRule_Patch:
		return &httpRuleParseOutput{
			path:   rule.GetPatch(),
			method: util.PATCH,
		}, nil
	case *annotationspb.HttpRule_Custom:
		return &httpRuleParseOutput{
			path:   rule.GetCustom().GetPath(),
			method: rule.GetCustom().GetKind(),
		}, nil
	default:
		return nil, fmt.Errorf("error parsing http rule type: unsupported http method %T", rule.GetPattern())
	}
}

// PrecomputeHTTPRuleBySelectorFromOPConfig pre-processes the service config to return
// a map of selector to the corresponding HTTP rule.
func PrecomputeHTTPRuleBySelectorFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) map[string]*annotationspb.HttpRule {
	httpRuleBySelector := make(map[string]*annotationspb.HttpRule)

	for _, rule := range serviceConfig.GetHttp().GetRules() {
		if util.ShouldSkipOPDiscoveryAPI(rule.GetSelector(), opts.AllowDiscoveryAPIs) {
			glog.Warningf("Skip HTTP rule %q because discovery API is not supported.", rule.GetSelector())
			continue
		}
		httpRuleBySelector[rule.GetSelector()] = rule
	}

	return httpRuleBySelector
}
