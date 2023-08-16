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

// Package routegentest contains test helpers to test route generators.
package routegentest

import (
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/routegen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/imdario/mergo"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// SuccessOPTestCase is the shared struct to test a RouteGenerator with
// One Platform config. It checks the factory and generator both succeed.
type SuccessOPTestCase struct {
	Desc string

	ServiceConfigIn *servicepb.Service

	// OptsIn is the input ESPv2 Options.
	// Will be merged with defaults.
	OptsIn options.ConfigGeneratorOptions

	// OptsMergeBehavior is implementation detail on how opts are merged into
	// default ESPv2 opts. Useful to manually set when dealing with opts
	// that default to true, but you want to set to false (empty value).
	//
	// See https://github.com/imdario/mergo/issues/129 for an example.
	//
	// If specified, make sure `OptsIn` contains ALL options you need, as defaults
	// will most likely be ignored.
	OptsMergeBehavior func(*mergo.Config)

	// WantHostConfig is the expected virtual host config, in JSON.
	// Only the routes are populated and verified against the generated routes.
	WantHostConfig string
}

// RunTest is a test helper to run the test.
func (tc *SuccessOPTestCase) RunTest(t *testing.T, factory routegen.RouteGeneratorOPFactory) {
	t.Helper()

	if tc.OptsMergeBehavior == nil {
		tc.OptsMergeBehavior = mergo.WithOverride
	}
	t.Run(tc.Desc, func(t *testing.T) {
		opts := options.DefaultConfigGeneratorOptions()
		if err := mergo.Merge(&opts, tc.OptsIn, tc.OptsMergeBehavior); err != nil {
			t.Fatalf("Merge() of test opts into default opts got err: %v", err)
		}

		gotGenerators, err := factory(tc.ServiceConfigIn, opts)
		if err != nil {
			t.Fatalf("NewXYZRouteGensFromOPConfig() got error: %v", err)
		}

		gotHost := &routepb.VirtualHost{}
		for i, gotGenerator := range gotGenerators {
			gotRoutes, err := gotGenerator.GenRouteConfig()
			if err != nil {
				t.Fatalf("GenRouteConfig() at generator %d got error: %v", i, err)
			}

			gotHost.Routes = append(gotHost.Routes, gotRoutes...)
		}

		gotJson, err := util.ProtoToJson(gotHost)
		if err != nil {
			t.Fatalf("Fail to convert generated virtual host config proto to JSON: %v", err)
		}

		if err := util.JsonEqual(tc.WantHostConfig, gotJson); err != nil {
			t.Errorf("Fail during route config JSON comparison of generated virtual host \n %v", err)
		}
	})
}

// GenConfigErrorOPTestCase is the shared struct to test a RouteGenerator with
// One Platform config. It checks that the factory is successful, but the route
// generation returns an error.
type GenConfigErrorOPTestCase struct {
	Desc string

	ServiceConfigIn *servicepb.Service

	// OptsIn is the input ESPv2 Options.
	// Will be merged with defaults.
	OptsIn options.ConfigGeneratorOptions

	// OptsMergeBehavior is implementation detail on how opts are merged into
	// default ESPv2 opts. Useful to manually set when dealing with opts
	// that default to true, but you want to set to false (empty value).
	//
	// See https://github.com/imdario/mergo/issues/129 for an example.
	//
	// If specified, make sure `OptsIn` contains ALL options you need, as defaults
	// will most likely be ignored.
	OptsMergeBehavior func(*mergo.Config)

	// WantGenErrors are the expected errors from each generator.
	WantGenErrors []string
}

// RunTest is a test helper to run the test.
func (tc *GenConfigErrorOPTestCase) RunTest(t *testing.T, factory routegen.RouteGeneratorOPFactory) {
	t.Helper()

	if tc.OptsMergeBehavior == nil {
		tc.OptsMergeBehavior = mergo.WithOverride
	}
	t.Run(tc.Desc, func(t *testing.T) {
		opts := options.DefaultConfigGeneratorOptions()
		if err := mergo.Merge(&opts, tc.OptsIn, tc.OptsMergeBehavior); err != nil {
			t.Fatalf("Merge() of test opts into default opts got err: %v", err)
		}

		gotGenerators, err := factory(tc.ServiceConfigIn, opts)
		if err != nil {
			t.Fatalf("NewXYZRouteGensFromOPConfig() got error: %v", err)
		}

		for i, gotGenerator := range gotGenerators {
			_, err := gotGenerator.GenRouteConfig()
			if err == nil {
				t.Fatalf("GenRouteConfig() for generator %d got no error, want error", i)
			}
			if !strings.Contains(err.Error(), tc.WantGenErrors[i]) {
				t.Errorf("GenRouteConfig() for generator %d got error %q, want error to contain %q", i, err.Error(), tc.WantGenErrors[i])
			}
		}
	})
}
