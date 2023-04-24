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

package clustergen_test

import (
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/clustergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/helpers"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/google/go-cmp/cmp"
	"github.com/imdario/mergo"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
	"google.golang.org/protobuf/testing/protocmp"
)

// SuccessOPTestCase is the shared struct to test a ClusterGenerator with
// One Platform config. It checks the factory and generator both succeed.
type SuccessOPTestCase struct {
	Desc string

	ServiceConfigIn *servicepb.Service

	// OptsIn is the input ESPv2 Options.
	// Will be merged with defaults.
	OptsIn options.ConfigGeneratorOptions

	// WantClusters is the expected output.
	WantClusters []*clusterpb.Cluster
}

// RunTest is a test helper to run the test.
func (tc *SuccessOPTestCase) RunTest(t *testing.T, factory clustergen.ClusterGeneratorOPFactory) {
	t.Helper()
	t.Run(tc.Desc, func(t *testing.T) {
		opts := options.DefaultConfigGeneratorOptions()
		if err := mergo.Merge(&opts, tc.OptsIn, mergo.WithOverride); err != nil {
			t.Fatalf("Merge() of test opts into default opts got err: %v", err)
		}

		gotGenerators, err := factory(tc.ServiceConfigIn, opts)
		if err != nil {
			t.Fatalf("NewXYZClusterFromOPConfig() got error: %v", err)
		}

		var gotClusters []*clusterpb.Cluster

		for i, gotGenerator := range gotGenerators {
			gotCluster, err := gotGenerator.GenConfig()
			if err != nil {
				t.Fatalf("GenConfig() at generator %d got error: %v", i, err)
			}
			gotClusters = append(gotClusters, gotCluster)
		}

		if diff := cmp.Diff(tc.WantClusters, gotClusters, protocmp.Transform()); diff != "" {
			t.Errorf("Cluster diff (-want +got):\n%s", diff)
		}
	})
}

// DisabledOPTestCase is the shared struct to test a ClusterGenerator with
// One Platform config. It checks that the factory function returns a nil
// ClusterGenerator.
type DisabledOPTestCase struct {
	Desc string

	ServiceConfigIn *servicepb.Service

	// OptsIn is the input ESPv2 Options.
	// Will be merged with defaults.
	OptsIn options.ConfigGeneratorOptions
}

// RunTest is a test helper to run the test.
func (tc *DisabledOPTestCase) RunTest(t *testing.T, factory clustergen.ClusterGeneratorOPFactory) {
	t.Helper()
	t.Run(tc.Desc, func(t *testing.T) {
		opts := options.DefaultConfigGeneratorOptions()
		if err := mergo.Merge(&opts, tc.OptsIn, mergo.WithOverride); err != nil {
			t.Fatalf("Merge() of test opts into default opts got err: %v", err)
		}

		gotGenerator, err := factory(tc.ServiceConfigIn, opts)
		if err != nil {
			t.Fatalf("NewXYZClusterFromOPConfig() got error: %v", err)
		}
		if gotGenerator != nil {
			t.Errorf("NewXYZClusterFromOPConfig() got generator, want no generator")
		}
	})
}

// DisabledOPTestCase is the shared struct to test a ClusterGenerator with
// One Platform config. It checks that the factory function returns a nil
// ClusterGenerator.
type FactoryErrorOPTestCase struct {
	Desc string

	ServiceConfigIn *servicepb.Service

	// OptsIn is the input ESPv2 Options.
	// Will be merged with defaults.
	OptsIn options.ConfigGeneratorOptions

	// WantFactoryError is the error that occurs when `NewXYZClusterFromOPConfig()`
	// is called.
	WantFactoryError string
}

// RunTest is a test helper to run the test.
func (tc *FactoryErrorOPTestCase) RunTest(t *testing.T, factory clustergen.ClusterGeneratorOPFactory) {
	t.Helper()
	t.Run(tc.Desc, func(t *testing.T) {
		opts := options.DefaultConfigGeneratorOptions()
		if err := mergo.Merge(&opts, tc.OptsIn, mergo.WithOverride); err != nil {
			t.Fatalf("Merge() of test opts into default opts got err: %v", err)
		}

		_, err := factory(tc.ServiceConfigIn, opts)
		if err == nil {
			t.Fatalf("NewXYZClusterFromOPConfig() got no error, want error")
		}
		if !strings.Contains(err.Error(), tc.WantFactoryError) {
			t.Errorf("NewXYZClusterFromOPConfig() got error %q, want error to contain %q", err.Error(), tc.WantFactoryError)
		}
	})
}

// CreateDefaultTLS is a helper function to create TLS config with default production
// settings.
func CreateDefaultTLS(t *testing.T, hostname string, isH2 bool) *corepb.TransportSocket {
	t.Helper()
	tls := helpers.ClusterTLSConfiger{
		RootCertsPath: util.DefaultRootCAPaths,
	}

	var alpn []string
	if isH2 {
		alpn = append(alpn, "h2")
	}

	transportSocket, err := tls.MakeTLSConfig(hostname, alpn)
	if err != nil {
		t.Fatalf("MakeTLSConfig got error: %v", err)
	}

	return transportSocket
}
