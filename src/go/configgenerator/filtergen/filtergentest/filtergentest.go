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

// Package filtergentest contains test helpers to test filter generators.
package filtergentest

import (
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/configgenerator/filtergen"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
	"github.com/imdario/mergo"
	servicepb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

// SuccessOPTestCase is the shared struct to test a FilterGenerator with
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

	// OnlyCheckFilterConfig indicates the WantFilterConfigs only represents
	// the filter config proto, not the surrounding HTTP filter.
	OnlyCheckFilterConfig bool

	// WantFilterConfigs is the expected filter config message per generator.
	// It is an ordered slice. Each element is the JSON representation of the
	// filter config.
	WantFilterConfigs []string
}

// RunTest is a test helper to run the test.
func (tc *SuccessOPTestCase) RunTest(t *testing.T, factory filtergen.FilterGeneratorOPFactory) {
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
			t.Fatalf("NewXYZFilterGensFromOPConfig() got error: %v", err)
		}

		if len(gotGenerators) != len(tc.WantFilterConfigs) {
			t.Fatalf("Invalid number of filter generators, got %d, want %d", len(gotGenerators), len(tc.WantFilterConfigs))
		}

		for i, gotGenerator := range gotGenerators {
			gotConfig, err := gotGenerator.GenFilterConfig()
			if err != nil {
				t.Fatalf("GenFilterConfig() at generator %d got error: %v", i, err)
			}

			gotHTTPFilter := gotConfig
			if !tc.OnlyCheckFilterConfig {
				gotHTTPFilter, err = filtergen.FilterConfigToHTTPFilter(gotConfig, gotGenerator.FilterName())
				if err != nil {
					t.Fatalf("Fail to convert filter config to HTTP filter for generator %d: %v", i, err)
				}
			}

			gotJson, err := util.ProtoToJson(gotHTTPFilter)
			if err != nil {
				t.Fatalf("Fail to convert HTTP filter config proto at generator %d to JSON: %v", i, err)
			}

			if err := util.JsonEqual(tc.WantFilterConfigs[i], gotJson); err != nil {
				t.Errorf("Fail during filter config JSON comparison at generator %d \n %v", i, err)
			}
		}
	})
}

// FactoryErrorOPTestCase is the shared struct to test a FilterGenerator with
// One Platform config. It checks that the factory returns an error.
type FactoryErrorOPTestCase struct {
	Desc string

	ServiceConfigIn *servicepb.Service

	// OptsIn is the input ESPv2 Options.
	// Will be merged with defaults.
	OptsIn options.ConfigGeneratorOptions

	// WantFactoryError is the error that occurs when `NewXYZFilterGensFromOPConfig()`
	// is called.
	WantFactoryError string
}

// RunTest is a test helper to run the test.
func (tc *FactoryErrorOPTestCase) RunTest(t *testing.T, factory filtergen.FilterGeneratorOPFactory) {
	t.Helper()
	t.Run(tc.Desc, func(t *testing.T) {
		opts := options.DefaultConfigGeneratorOptions()
		if err := mergo.Merge(&opts, tc.OptsIn, mergo.WithOverride); err != nil {
			t.Fatalf("Merge() of test opts into default opts got err: %v", err)
		}

		_, err := factory(tc.ServiceConfigIn, opts)
		if err == nil {
			t.Fatalf("NewXYZFilterGensFromOPConfig() got no error, want error")
		}
		if !strings.Contains(err.Error(), tc.WantFactoryError) {
			t.Errorf("NewXYZFilterGensFromOPConfig() got error %q, want error to contain %q", err.Error(), tc.WantFactoryError)
		}
	})
}

// GenConfigErrorOPTestCase is the shared struct to test a FilterGenerator with
// One Platform config. It checks that the factory is successful, but the filter
// generation returns an error.
type GenConfigErrorOPTestCase struct {
	Desc string

	ServiceConfigIn *servicepb.Service

	// OptsIn is the input ESPv2 Options.
	// Will be merged with defaults.
	OptsIn options.ConfigGeneratorOptions

	// WantGenErrors are the expected errors from each generator.
	WantGenErrors []string
}

// RunTest is a test helper to run the test.
func (tc *GenConfigErrorOPTestCase) RunTest(t *testing.T, factory filtergen.FilterGeneratorOPFactory) {
	t.Helper()
	t.Run(tc.Desc, func(t *testing.T) {
		opts := options.DefaultConfigGeneratorOptions()
		if err := mergo.Merge(&opts, tc.OptsIn, mergo.WithOverride); err != nil {
			t.Fatalf("Merge() of test opts into default opts got err: %v", err)
		}

		gotGenerators, err := factory(tc.ServiceConfigIn, opts)
		if err != nil {
			t.Fatalf("NewXYZFilterGensFromOPConfig() got error: %v", err)
		}

		for i, gotGenerator := range gotGenerators {
			_, err := gotGenerator.GenFilterConfig()
			if err == nil {
				t.Fatalf("GenFilterConfig() for generator %d got no error, want error", i)
			}
			if !strings.Contains(err.Error(), tc.WantGenErrors[i]) {
				t.Errorf("GenFilterConfig() for generator %d got error %q, want error to contain %q", i, err.Error(), tc.WantGenErrors[i])
			}
		}
	})
}
