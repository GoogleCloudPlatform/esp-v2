// Copyright 2023 Google LLC
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

package clustergen

import (
	"fmt"

	helpers2 "github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"github.com/golang/glog"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// RemoteBackendCluster is an Envoy cluster to communicate with remote backends
// via dynamic routing. Primarily for API Gateway use case.
type RemoteBackendCluster struct {
	BackendCluster *helpers2.BaseBackendCluster
}

// NewRemoteBackendClustersFromOPConfig creates all RemoteBackendCluster from
// OP service config + descriptor + ESPv2 options. It is a ClusterGeneratorOPFactory.
//
// Generates multiple clusters, 1 per remote backend. Automatically de-duplicates
// multiple clusters with the same remote socket address.
func NewRemoteBackendClustersFromOPConfig(serviceConfig *servicepb.Service, opts options.ConfigGeneratorOptions) ([]ClusterGenerator, error) {
	var gens []ClusterGenerator
	dedupClusterNames := make(map[string]bool)

	for _, rule := range serviceConfig.GetBackend().GetRules() {
		if util.ShouldSkipOPDiscoveryAPI(rule.GetSelector(), opts.AllowDiscoveryAPIs) {
			glog.Warningf("Skip backend rule %q because discovery API is not supported.", rule.GetSelector())
			continue
		}

		gen, err := backendRuleToCluster(rule, opts, false)
		if err != nil {
			return nil, fmt.Errorf("fail to create RemoteBackendCluster for selector %q: %v", rule.GetSelector(), err)
		}
		gens = dedupAndAddGenerator(gen, gens, dedupClusterNames)

		httpBackendGen, err := httpBackendRuleToCluster(rule, opts)
		if err != nil {
			return nil, fmt.Errorf("fail to create HTTP RemoteBackendCluster for selector %q: %v", rule.GetSelector(), err)
		}
		gens = dedupAndAddGenerator(httpBackendGen, gens, dedupClusterNames)
	}

	return gens, nil
}

// httpBackendRuleToCluster creates a RemoteBackendCluster for non-OpenAPI HTTP backend support.
// This is not used by ESPv2.
func httpBackendRuleToCluster(rule *servicepb.BackendRule, opts options.ConfigGeneratorOptions) (*RemoteBackendCluster, error) {
	httpBackendRule := rule.GetOverridesByRequestProtocol()[util.HTTPBackendProtocolKey]
	if httpBackendRule == nil {
		return nil, nil
	}

	// TODO(yangshuo): remove this after the API compiler ensures it.
	httpBackendRule.Selector = rule.GetSelector()

	glog.Infof("Selector %q has HTTP backend rule", rule.GetSelector())
	return backendRuleToCluster(httpBackendRule, opts, true)
}

// backendRuleToCluster is a shared helper to translate a BackendRule into a RemoteBackendCluster.
func backendRuleToCluster(rule *servicepb.BackendRule, opts options.ConfigGeneratorOptions, isHTTPBackend bool) (*RemoteBackendCluster, error) {
	if opts.EnableBackendAddressOverride {
		glog.Infof("Skipping create remote cluster from backend rule %q because backend address override is enabled.", rule.GetSelector())
		return nil, nil
	}

	if rule.GetAddress() == "" {
		glog.Infof("Skip backend rule %q because it does not have dynamic routing address.", rule.GetSelector())
		return nil, nil
	}

	scheme, hostname, port, _, err := util.ParseURI(rule.GetAddress())
	if err != nil {
		return nil, fmt.Errorf("error parsing remote backend rule's address for operation %q, %v", rule.GetSelector(), err)
	}

	// Create a cluster for the remote backend.
	protocol, useTLS, err := util.ParseBackendProtocol(scheme, rule.GetProtocol())
	if err != nil {
		return nil, fmt.Errorf("error parsing remote backend rule's protocol for operation %q, %v", rule.GetSelector(), err)
	}
	if protocol == util.GRPC {
		if isHTTPBackend {
			return nil, fmt.Errorf("gRPC protocol conflicted with http backend; this is an API compiler bug")
		}
	}

	var tls *helpers2.ClusterTLSConfiger
	if useTLS {
		tls = helpers2.NewClusterTLSConfigerFromOPConfig(opts, true)
	}

	address := fmt.Sprintf("%v:%v", hostname, port)
	cluster := &RemoteBackendCluster{
		BackendCluster: &helpers2.BaseBackendCluster{
			ClusterName:            fmt.Sprintf("backend-cluster-%s", address),
			Hostname:               hostname,
			Port:                   port,
			Protocol:               protocol,
			ClusterConnectTimeout:  opts.ClusterConnectTimeout,
			MaxRequestsThreshold:   opts.BackendClusterMaxRequests,
			BackendDnsLookupFamily: opts.BackendDnsLookupFamily,
			DNS:                    helpers2.NewClusterDNSConfigerFromOPConfig(opts),
			TLS:                    tls,
		},
	}
	return cluster, nil
}

// dedupAndAddGenerator is a helper to update tracking variables when adding a new ClusterGenerator to the output.
func dedupAndAddGenerator(gen *RemoteBackendCluster, gens []ClusterGenerator, dedupClusterNames map[string]bool) []ClusterGenerator {
	if gen == nil {
		return gens
	}

	if _, exist := dedupClusterNames[gen.GetName()]; !exist {
		dedupClusterNames[gen.GetName()] = true
		gens = append(gens, gen)
	}
	return gens
}

// GetName implements the ClusterGenerator interface.
func (c *RemoteBackendCluster) GetName() string {
	return c.BackendCluster.ClusterName
}

// GenConfig implements the ClusterGenerator interface.
func (c *RemoteBackendCluster) GenConfig() (*clusterpb.Cluster, error) {
	return c.BackendCluster.GenBaseConfig()
}
