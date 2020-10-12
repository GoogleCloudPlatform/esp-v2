// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configinfo

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"

	commonpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v9/http/common"
	scpb "github.com/GoogleCloudPlatform/esp-v2/src/go/proto/api/envoy/v9/http/service_control"
	annotationspb "google.golang.org/genproto/googleapis/api/annotations"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	typepb "google.golang.org/genproto/protobuf/ptype"
)

// ServiceInfo contains service level information.
type ServiceInfo struct {
	Name     string
	ConfigID string

	// An array to store all the api names
	ApiNames []string

	// A ordered slice of operation names. Follows the same order as the `apis.methods` in service config.
	// All functions that output order-dependent configs should use this ordering.
	//
	// Ordering is important for Envoy route matching and testability.
	// Envoy's router matches linearly with first-to-win. When wildcard routes are introduced,
	// they must show up last as a fallback route. Otherwise we may match a less-specific route,
	// resulting in an incorrect host rewrite.
	Operations []string

	// Stores all methods info for this service, using selector as key.
	Methods map[string]*methodInfo

	// Stores all the query parameters to be ignored for json-grpc transcoder.
	AllTranscodingIgnoredQueryParams map[string]bool

	AllowCors         bool
	ServiceControlURI string
	GcpAttributes     *scpb.GcpAttributes
	// Keep a pointer to original service config. Should always process rules
	// inside ServiceInfo.
	serviceConfig *confpb.Service
	AccessToken   *commonpb.AccessToken
	Options       options.ConfigGeneratorOptions

	// Stores information about all backend clusters.
	GrpcSupportRequired   bool
	LocalBackendCluster   *BackendRoutingCluster
	RemoteBackendClusters []*BackendRoutingCluster
}

type BackendRoutingCluster struct {
	ClusterName string
	Hostname    string
	Port        uint32
	UseTLS      bool
	Protocol    util.BackendProtocol
}

// NewServiceInfoFromServiceConfig returns an instance of ServiceInfo.
func NewServiceInfoFromServiceConfig(serviceConfig *confpb.Service, id string, opts options.ConfigGeneratorOptions) (*ServiceInfo, error) {
	if serviceConfig == nil {
		return nil, fmt.Errorf("unexpected empty service config")
	}
	if len(serviceConfig.GetApis()) == 0 {
		return nil, fmt.Errorf("service config must have one api at least")
	}

	serviceInfo := &ServiceInfo{
		Name:                             serviceConfig.GetName(),
		ConfigID:                         id,
		serviceConfig:                    serviceConfig,
		Options:                          opts,
		Methods:                          make(map[string]*methodInfo),
		AllTranscodingIgnoredQueryParams: make(map[string]bool),
	}

	// Calling order is required due to following variable usage
	// * AllowCors:
	//    set by: processEndpoints
	//    used by: processHttpRule
	// * BackendInfo map to MethodInfo
	//    set by processApi
	//    used by processBackendRule
	// * GrpcSupportRequired:
	//     set by processBackendRule, buildLocalBackend
	//     used by addGrpcHttpRules
	// * Methods:
	//		 set by processApis, processHttpRule, addGrpcHttpRules, processUsageRule
	//     used by processApiKeyLocations
	if err := serviceInfo.buildLocalBackend(); err != nil {
		return nil, err
	}
	serviceInfo.processEndpoints()
	serviceInfo.processApis()
	serviceInfo.processQuota()
	if err := serviceInfo.processBackendRule(); err != nil {
		return nil, err
	}
	if err := serviceInfo.processHttpRule(); err != nil {
		return nil, err
	}
	if err := serviceInfo.processUsageRule(); err != nil {
		return nil, err
	}

	serviceInfo.processAccessToken()
	if err := serviceInfo.processTypes(); err != nil {
		return nil, err
	}
	serviceInfo.addGrpcHttpRules()
	if err := serviceInfo.processTranscodingIgnoredQueryParams(); err != nil {
		return nil, err
	}
	if err := serviceInfo.processApiKeyLocations(); err != nil {
		return nil, err
	}

	if err := serviceInfo.processEmptyJwksUriByOpenID(); err != nil {
		return nil, err
	}
	if err := serviceInfo.processLocalBackendOperations(); err != nil {
		return nil, err
	}

	return serviceInfo, nil
}

func (s *ServiceInfo) buildLocalBackend() error {

	scheme, hostname, port, _, err := util.ParseURI(s.Options.BackendAddress)
	if err != nil {
		return fmt.Errorf("error parsing backend uri: %v", err)
	}

	// For local backend, user cannot configure http protocol explicitly.
	protocol, tls, err := util.ParseBackendProtocol(scheme, "")
	if err != nil {
		return err
	}
	if protocol == util.GRPC {
		s.GrpcSupportRequired = true
	}

	s.LocalBackendCluster = &BackendRoutingCluster{
		UseTLS:      tls,
		Protocol:    protocol,
		ClusterName: s.LocalBackendClusterName(),
		Hostname:    hostname,
		Port:        port,
	}
	return nil
}

// Returns the pointer of the ServiceConfig that this API belongs to.
func (s *ServiceInfo) ServiceConfig() *confpb.Service {
	return s.serviceConfig
}

func (s *ServiceInfo) processEmptyJwksUriByOpenID() error {
	authn := s.serviceConfig.GetAuthentication()
	for _, provider := range authn.GetProviders() {
		jwksUri := provider.GetJwksUri()

		// Note: When jwksUri is empty, proxy will try to find jwksUri using the
		// OpenID Connect Discovery protocol.
		if jwksUri == "" {
			glog.Infof("jwks_uri is empty, using OpenID Connect Discovery protocol")
			jwksUriByOpenID, err := util.ResolveJwksUriUsingOpenID(provider.GetIssuer())
			if err != nil {
				return fmt.Errorf("failed OpenID Connect Discovery protocol: %v", err)
			} else {
				jwksUri = jwksUriByOpenID
			}
			provider.JwksUri = jwksUri
		}
	}
	return nil
}

func (s *ServiceInfo) processApis() {
	for _, api := range s.serviceConfig.GetApis() {
		s.ApiNames = append(s.ApiNames, api.Name)

		for _, method := range api.GetMethods() {
			selector := fmt.Sprintf("%s.%s", api.GetName(), method.GetName())
			mi, _ := s.getOrCreateMethod(selector)

			// Keep track of non-unary gRPC methods.
			if method.RequestStreaming || method.ResponseStreaming {
				mi.IsStreaming = true
			}
			mi.ApiVersion = api.Version

			// Keep track of request type name.
			if strings.HasPrefix(method.RequestTypeUrl, util.TypeUrlPrefix) {
				requestTypeName := strings.TrimPrefix(method.RequestTypeUrl, util.TypeUrlPrefix)
				mi.RequestTypeName = requestTypeName
			} else {
				glog.Warningf("For operation (%v), request type name (%v) is in an unexpected format", selector, method.RequestTypeUrl)
			}
		}
	}
}

func (s *ServiceInfo) addGrpcHttpRules() {
	// If there is not grpc backend, not to add grpc HttpRules
	if !s.GrpcSupportRequired {
		return
	}

	for _, api := range s.serviceConfig.GetApis() {
		for _, method := range api.GetMethods() {
			selector := fmt.Sprintf("%s.%s", api.GetName(), method.GetName())
			mi, _ := s.getOrCreateMethod(selector)
			mi.HttpRule = append(mi.HttpRule, &commonpb.Pattern{
				UriTemplate: fmt.Sprintf("/%s/%s", api.GetName(), method.GetName()),
				HttpMethod:  util.POST,
			})
		}
	}
}

func (s *ServiceInfo) processAccessToken() {
	if s.Options.ServiceAccountKey != "" {
		s.AccessToken = &commonpb.AccessToken{
			TokenType: &commonpb.AccessToken_RemoteToken{
				RemoteToken: &commonpb.HttpUri{
					// Use http://127.0.0.1:8791/local/access_token by default.
					Uri:     fmt.Sprintf("http://%s:%v%s", util.LoopbackIPv4Addr, s.Options.TokenAgentPort, util.TokenAgentAccessTokenPath),
					Cluster: util.TokenAgentClusterName,
					Timeout: ptypes.DurationProto(s.Options.HttpRequestTimeout),
				},
			},
		}

		return
	}

	s.AccessToken = &commonpb.AccessToken{
		TokenType: &commonpb.AccessToken_RemoteToken{
			RemoteToken: &commonpb.HttpUri{
				Uri:     fmt.Sprintf("%s%s", s.Options.MetadataURL, util.AccessTokenPath),
				Cluster: util.MetadataServerClusterName,
				Timeout: ptypes.DurationProto(s.Options.HttpRequestTimeout),
			},
		},
	}

}

func (s *ServiceInfo) processQuota() {
	for _, metricRule := range s.ServiceConfig().GetQuota().GetMetricRules() {
		var metricCosts []*scpb.MetricCost
		for name, cost := range metricRule.GetMetricCosts() {
			metricCosts = append(metricCosts, &scpb.MetricCost{
				Name: name,
				Cost: cost,
			})
		}
		s.Methods[metricRule.GetSelector()].MetricCosts = metricCosts
	}
}

func (s *ServiceInfo) processEndpoints() {
	for _, endpoint := range s.ServiceConfig().GetEndpoints() {
		if endpoint.GetName() == s.ServiceConfig().GetName() && endpoint.GetAllowCors() {
			s.AllowCors = true
		}
	}
}

func addHttpRule(method *methodInfo, r *annotationspb.HttpRule, httpMatcherWithOptionsSet map[string]bool) error {

	var httpRule *commonpb.Pattern
	switch r.GetPattern().(type) {
	case *annotationspb.HttpRule_Get:
		httpRule = &commonpb.Pattern{
			UriTemplate: r.GetGet(),
			HttpMethod:  util.GET,
		}
	case *annotationspb.HttpRule_Put:
		httpRule = &commonpb.Pattern{
			UriTemplate: r.GetPut(),
			HttpMethod:  util.PUT,
		}
	case *annotationspb.HttpRule_Post:
		httpRule = &commonpb.Pattern{
			UriTemplate: r.GetPost(),
			HttpMethod:  util.POST,
		}
	case *annotationspb.HttpRule_Delete:
		httpRule = &commonpb.Pattern{
			UriTemplate: r.GetDelete(),
			HttpMethod:  util.DELETE,
		}
	case *annotationspb.HttpRule_Patch:
		httpRule = &commonpb.Pattern{
			UriTemplate: r.GetPatch(),
			HttpMethod:  util.PATCH,
		}
	case *annotationspb.HttpRule_Custom:
		httpMethod := r.GetCustom().GetKind()
		httpRule = &commonpb.Pattern{
			UriTemplate: r.GetCustom().GetPath(),
			HttpMethod:  httpMethod,
		}

		if httpMethod == util.OPTIONS {
			// Ensure we don't generate duplicate methods later for AllowCors.
			matcher := util.WildcardMatcherForPath(r.GetCustom().GetPath())
			if matcher == "" {
				matcher = r.GetCustom().GetPath()
			}
			httpMatcherWithOptionsSet[matcher] = true
		}
	default:
		return fmt.Errorf("unsupported http method %T", r.GetPattern())
	}
	method.HttpRule = append(method.HttpRule, httpRule)

	return nil
}

func (s *ServiceInfo) processHttpRule() error {
	// An temporary map to record generated OPTION methods, to avoid duplication.
	httpMatcherWithOptionsSet := make(map[string]bool)

	for _, rule := range s.ServiceConfig().GetHttp().GetRules() {
		method, err := s.getOrCreateMethod(rule.GetSelector())
		if err != nil {
			return err
		}
		if err := addHttpRule(method, rule, httpMatcherWithOptionsSet); err != nil {
			return err
		}

		// additional_bindings cannot be nested inside themselves according to
		// https://aip.dev/127. Service Management will enforce this restriction
		// when interpret the httprules from the descriptor. Therefore, no need to
		// check for nested additional_bindings.
		for _, additionalRule := range rule.AdditionalBindings {
			if err := addHttpRule(method, additionalRule, httpMatcherWithOptionsSet); err != nil {
				return err
			}
		}
	}

	// In order to support CORS. HTTP method OPTIONS needs to be added to all
	// urls except the ones already with options.
	if s.AllowCors {
		for _, r := range s.ServiceConfig().GetHttp().GetRules() {
			method := s.Methods[r.GetSelector()]
			for _, httpRule := range method.HttpRule {
				if httpRule.HttpMethod != util.OPTIONS {
					matcher := util.WildcardMatcherForPath(httpRule.UriTemplate)
					if matcher == "" {
						matcher = httpRule.UriTemplate
					}
					if _, exist := httpMatcherWithOptionsSet[matcher]; !exist {
						if err := s.addOptionMethod(method, httpRule.UriTemplate); err != nil {
							return err
						}
						httpMatcherWithOptionsSet[matcher] = true
					}

				}
			}
		}
	}

	// Add HttpRule for HealthCheck method
	if s.Options.Healthz != "" {
		methodName := fmt.Sprintf("%s.%s_HealthCheck", util.EspOperation, util.AutogeneratedOperationPrefix)

		hcMethod, err := s.getOrCreateMethod(methodName)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(s.Options.Healthz, "/") {
			s.Options.Healthz = fmt.Sprintf("/%s", s.Options.Healthz)
		}

		hcMethod.HttpRule = append(hcMethod.HttpRule, &commonpb.Pattern{
			UriTemplate: s.Options.Healthz,
			HttpMethod:  util.GET,
		})
		hcMethod.SkipServiceControl = true
		hcMethod.IsGenerated = true
	}

	return nil
}

func (s *ServiceInfo) addOptionMethod(originalMethod *methodInfo, path string) error {
	genOperation := fmt.Sprintf("%s.%s_CORS_%s", originalMethod.ApiName, util.AutogeneratedOperationPrefix, originalMethod.ShortName)

	method, err := s.getOrCreateMethod(genOperation)
	if err != nil {
		return err
	}

	method.ApiVersion = originalMethod.ApiVersion
	method.BackendInfo = originalMethod.BackendInfo
	method.IsGenerated = true
	method.HttpRule = append(method.HttpRule, &commonpb.Pattern{
		UriTemplate: path,
		HttpMethod:  util.OPTIONS,
	})

	return nil
}

func (s *ServiceInfo) processBackendRule() error {
	backendRoutingClustersMap := make(map[string]string)

	for _, r := range s.ServiceConfig().Backend.GetRules() {

		if r.Address == "" {
			// Processing a backend rule associated with the local backend.
			if err := s.addBackendInfoToMethod(r, "", "", "", s.LocalBackendClusterName()); err != nil {
				return err
			}
		} else {
			// Processing a backend rule associated with a remote backend.
			scheme, hostname, port, path, err := util.ParseURI(r.Address)
			if err != nil {
				return err
			}
			address := fmt.Sprintf("%v:%v", hostname, port)

			if _, exist := backendRoutingClustersMap[address]; !exist {
				// Create cluster for the remote backend.
				protocol, tls, err := util.ParseBackendProtocol(scheme, r.Protocol)
				if err != nil {
					return err
				}
				if protocol == util.GRPC {
					s.GrpcSupportRequired = true
				}

				backendClusterName := util.BackendClusterName(address)
				s.RemoteBackendClusters = append(s.RemoteBackendClusters,
					&BackendRoutingCluster{
						ClusterName: backendClusterName,
						UseTLS:      tls,
						Protocol:    protocol,
						Hostname:    hostname,
						Port:        port,
					})
				backendRoutingClustersMap[address] = backendClusterName
			}

			backendClusterName := backendRoutingClustersMap[address]
			if err := s.addBackendInfoToMethod(r, scheme, hostname, path, backendClusterName); err != nil {
				return err
			}
		}

	}
	return nil
}

func (s *ServiceInfo) addBackendInfoToMethod(r *confpb.BackendRule, scheme string, hostname string, path string, backendClusterName string) error {
	method, err := s.getOrCreateMethod(r.GetSelector())
	if err != nil {
		return err
	}

	// For CONSTANT_ADDRESS, an empty uri will generate an empty path header.
	// It is an invalid Http header if path is empty.
	if path == "" && r.PathTranslation == confpb.BackendRule_CONSTANT_ADDRESS {
		path = "/"
	}

	var deadline time.Duration
	if r.Deadline == 0 {
		// If no deadline specified by the user, explicitly use default.
		deadline = util.DefaultResponseDeadline
	} else if r.Deadline < 0 {
		glog.Warningf("Negative deadline of %v specified for method %v. "+
			"Using default deadline %v instead.", r.Deadline, r.Selector, util.DefaultResponseDeadline)
		deadline = util.DefaultResponseDeadline
	} else {
		// The backend deadline from the BackendRule is a float64 that represents seconds.
		// But float64 has a large precision, so we must explicitly lower the precision.
		// For the purposes of a network proxy, round the deadline to the nearest millisecond.
		deadlineMs := int64(math.Round(r.Deadline * 1000))
		deadline = time.Duration(deadlineMs) * time.Millisecond
	}

	method.BackendInfo = &backendInfo{
		ClusterName:     backendClusterName,
		Path:            path,
		Hostname:        hostname,
		TranslationType: r.PathTranslation,
		Deadline:        deadline,
	}

	jwtAud := s.determineBackendAuthJwtAud(r, scheme, hostname)
	if jwtAud != "" && s.Options.CommonOptions.NonGCP {
		glog.Warningf("Backend authentication is enabled for method %v, "+
			"but ESPv2 is running on non-GCP. To prevent contacting GCP services, "+
			"backend authentication is automatically being disabled for this method.",
			r.Selector)
		jwtAud = ""
	}
	method.BackendInfo.JwtAudience = jwtAud

	return nil
}

func (s *ServiceInfo) determineBackendAuthJwtAud(r *confpb.BackendRule, scheme string, hostname string) string {
	//TODO(taoxuy): b/149334660 Check if the scopes for IAM include the path prefix
	switch r.GetAuthentication().(type) {
	case *confpb.BackendRule_JwtAudience:
		return r.GetJwtAudience()
	case *confpb.BackendRule_DisableAuth:
		if r.GetDisableAuth() {
			return ""
		}
		return getJwtAudienceFromBackendAddr(scheme, hostname)
	default:
		if r.Address == "" {
			return ""
		}
		return getJwtAudienceFromBackendAddr(scheme, hostname)
	}
}

// For methods that are not associated with any backend rules, create one
// to associate with the local backend cluster.
func (s *ServiceInfo) processLocalBackendOperations() error {
	for _, method := range s.Methods {
		if method.BackendInfo != nil {
			// This method is already associated with a backend rule.
			continue
		}

		// Associate the method with the local backend.
		method.BackendInfo = &backendInfo{
			ClusterName: s.LocalBackendCluster.ClusterName,
			Deadline:    util.DefaultResponseDeadline,
		}
	}

	return nil
}

func (s *ServiceInfo) processUsageRule() error {
	for _, r := range s.ServiceConfig().GetUsage().GetRules() {
		method, err := s.getOrCreateMethod(r.GetSelector())
		if err != nil {
			return err
		}
		method.AllowUnregisteredCalls = r.GetAllowUnregisteredCalls()
		method.SkipServiceControl = r.GetSkipServiceControl()
	}
	return nil
}

func (s *ServiceInfo) processTranscodingIgnoredQueryParams() error {
	// Process ignored query params from jwt locations
	authn := s.serviceConfig.GetAuthentication()
	for _, provider := range authn.GetProviders() {
		// no custom JwtLocation so use default ones and set the one in query
		// parameter for transcoder to ignore.
		if len(provider.JwtLocations) == 0 {
			s.AllTranscodingIgnoredQueryParams[util.DefaultJwtQueryParamAccessToken] = true
			continue
		}

		for _, jwtLocation := range provider.JwtLocations {
			switch jwtLocation.In.(type) {
			case *confpb.JwtLocation_Query:
				if jwtLocation.ValuePrefix != "" {
					return fmt.Errorf("JwtLocation_Query should be set without valuePrefix, get JwtLocation {%v}", jwtLocation)
				}
				// set the custom JwtLocation in query parameter for transcoder to ignore.
				s.AllTranscodingIgnoredQueryParams[jwtLocation.GetQuery()] = true
			default:
				continue
			}
		}
	}

	// Process ignored query params from flag transcoding_ignore_query_params
	if s.Options.TranscodingIgnoreQueryParameters != "" {
		IgnoredQueryParametersFlag := strings.Split(s.Options.TranscodingIgnoreQueryParameters, ",")
		for _, IgnoredQueryParameter := range IgnoredQueryParametersFlag {
			s.AllTranscodingIgnoredQueryParams[IgnoredQueryParameter] = true

		}
	}

	return nil
}

func (s *ServiceInfo) processApiKeyLocations() error {
	for _, rule := range s.ServiceConfig().GetSystemParameters().GetRules() {
		apiKeyLocationParameters := []*confpb.SystemParameter{}

		for _, parameter := range rule.GetParameters() {
			if parameter.GetName() == util.ApiKeyParameterName {
				apiKeyLocationParameters = append(apiKeyLocationParameters, parameter)
			}
		}

		method, err := s.getOrCreateMethod(rule.GetSelector())
		if err != nil {
			return err
		}

		s.extractApiKeyLocations(method, apiKeyLocationParameters)
	}

	for _, method := range s.Methods {
		// If any of method is not set with custom ApiKeyLocations, use the default
		// one and set the custom ApiKeyLocations in query parameter for transcoder
		// to ignore.
		if len(method.ApiKeyLocations) == 0 {
			s.AllTranscodingIgnoredQueryParams[util.DefaultApiKeyQueryParamKey] = true
			s.AllTranscodingIgnoredQueryParams[util.DefaultApiKeyQueryParamApiKey] = true
		}

	}

	return nil
}

func (s *ServiceInfo) extractApiKeyLocations(method *methodInfo, parameters []*confpb.SystemParameter) {
	var urlQueryNames, headerNames []*scpb.ApiKeyLocation
	for _, parameter := range parameters {
		if urlQueryName := parameter.GetUrlQueryParameter(); urlQueryName != "" {
			urlQueryNames = append(urlQueryNames, &scpb.ApiKeyLocation{
				Key: &scpb.ApiKeyLocation_Query{
					Query: urlQueryName,
				},
			})
			// set the custom ApiKeyLocation in query parameter for transcoder to ignore.\
			s.AllTranscodingIgnoredQueryParams[urlQueryName] = true
		}
		if headerName := parameter.GetHttpHeader(); headerName != "" {
			headerNames = append(headerNames, &scpb.ApiKeyLocation{
				Key: &scpb.ApiKeyLocation_Header{
					Header: headerName,
				},
			})
		}
	}
	method.ApiKeyLocations = append(method.ApiKeyLocations, urlQueryNames...)
	method.ApiKeyLocations = append(method.ApiKeyLocations, headerNames...)
}

func (s *ServiceInfo) processTypes() error {

	// Convert into map by type name for easy lookup.
	typesByTypeName := make(map[string]*typepb.Type)
	for _, t := range s.ServiceConfig().GetTypes() {
		typesByTypeName[t.Name] = t
	}

	// For each method, lookup the request type.
	for operation, mi := range s.Methods {
		requestTypeName := mi.RequestTypeName
		if requestTypeName == "" {
			glog.Warningf("for operation (%v): request type was malformed", operation)
			continue
		}

		requestType, ok := typesByTypeName[requestTypeName]
		if !ok {
			glog.Warningf("for operation (%v): could not find type with name (%v)", operation, requestTypeName)
			continue
		}

		// Create snake name to JSON name mapping for the request operation (and validate against duplicates).
		snakeToJson := make(SnakeToJsonSegments)
		for _, field := range requestType.GetFields() {

			if field.Name != field.JsonName {

				if prevJsonName, ok := snakeToJson[field.GetName()]; ok {
					if prevJsonName != field.GetJsonName() {
						// Duplicate snake name with mismatching JSON name.
						// This will cause an error in path matcher variable bindings.
						// Disallow it.
						return fmt.Errorf("for operation (%v): detected two types with same snake_name (%v) "+
							"but mistmatching json_name (%v, %v)", operation, field.GetName(), field.GetJsonName(), prevJsonName)
					}
				}

				// Unique entry.
				snakeToJson[field.GetName()] = field.GetJsonName()
			}
		}

		// Replace the snake name with the json name in url template
		if len(snakeToJson) > 0 {
			for _, httpRule := range mi.HttpRule {
				httpRule.UriTemplate = util.SnakeNamesToJsonNamesInPathParam(httpRule.UriTemplate, snakeToJson)
			}

		}
	}
	return nil
}

// get the methodInfo by full name, and create a new one if not exists.
// Ideally, all selector name in service config rules should exist in the api
// methods.
func (s *ServiceInfo) getOrCreateMethod(name string) (*methodInfo, error) {
	if s.Methods[name] == nil {
		names := strings.Split(name, ".")
		if len(names) <= 1 {
			return nil, fmt.Errorf("method %s should be in the format of apiName.methodShortName", name)
		}
		shortName := names[len(names)-1]
		s.Methods[name] = &methodInfo{
			ShortName: shortName,
			ApiName:   name[:len(name)-len(shortName)-1],
		}
		s.Operations = append(s.Operations, name)
	}
	return s.Methods[name], nil
}

func (s *ServiceInfo) LocalBackendClusterName() string {
	return util.BackendClusterName(fmt.Sprintf("%s_local", s.Name))
}

// If the backend address's scheme is grpc/grpcs, it should be changed it http or https.
func getJwtAudienceFromBackendAddr(scheme, hostname string) string {
	_, tls, _ := util.ParseBackendProtocol(scheme, "")
	if tls {
		return fmt.Sprintf("https://%s", hostname)
	}
	return fmt.Sprintf("http://%s", hostname)
}
