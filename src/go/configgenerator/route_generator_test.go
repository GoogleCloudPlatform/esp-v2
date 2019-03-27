// Copyright 2019 Google Cloud Platform Proxy Authors
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

package configgenerator

import (
	"flag"
	"reflect"
	"strconv"
	"strings"
	"testing"

	sc "cloudesf.googlesource.com/gcpproxy/src/go/configinfo"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/gogo/protobuf/types"
)

func TestMakeRouteConfig(t *testing.T) {
	testData := []struct {
		desc string
		// Test parameters, in the order of "cors_preset", "cors_allow_origin"
		// "cors_allow_origin_regex", "cors_allow_methods", "cors_allow_headers"
		// "cors_expose_headers"
		params           []string
		allowCredentials bool
		wantedError      string
		wantRoute        *route.CorsPolicy
	}{
		{
			desc:      "No Cors",
			wantRoute: nil,
		},
		{
			desc:        "Incorrect configured basic Cors",
			params:      []string{"basic", "", `^https?://.+\\.example\\.com$`, "", "", ""},
			wantedError: "cors_allow_origin cannot be empty when cors_preset=basic",
		},
		{
			desc:        "Incorrect configured  Cors",
			params:      []string{"", "", "", "GET", "", ""},
			wantedError: "cors_preset must be set in order to enable CORS support",
		},
		{
			desc:        "Incorrect configured regex Cors",
			params:      []string{"cors_with_regexs", "", `^https?://.+\\.example\\.com$`, "", "", ""},
			wantedError: `cors_preset must be either "basic" or "cors_with_regex"`,
		},
		{
			desc:   "Correct configured basic Cors, with allow methods",
			params: []string{"basic", "http://example.com", "", "GET,POST,PUT,OPTIONS", "", ""},
			wantRoute: &route.CorsPolicy{
				AllowOrigin:      []string{"http://example.com"},
				AllowMethods:     "GET,POST,PUT,OPTIONS",
				AllowCredentials: &types.BoolValue{Value: false},
			},
		},
		{
			desc:   "Correct configured regex Cors, with allow headers",
			params: []string{"cors_with_regex", "", `^https?://.+\\.example\\.com$`, "", "Origin,Content-Type,Accept", ""},
			wantRoute: &route.CorsPolicy{
				AllowOriginRegex: []string{`^https?://.+\\.example\\.com$`},
				AllowHeaders:     "Origin,Content-Type,Accept",
				AllowCredentials: &types.BoolValue{Value: false},
			},
		},
		{
			desc:             "Correct configured regex Cors, with expose headers",
			params:           []string{"cors_with_regex", "", `^https?://.+\\.example\\.com$`, "", "", "Content-Length"},
			allowCredentials: true,
			wantRoute: &route.CorsPolicy{
				AllowOriginRegex: []string{`^https?://.+\\.example\\.com$`},
				ExposeHeaders:    "Content-Length",
				AllowCredentials: &types.BoolValue{Value: true},
			},
		},
	}

	for _, tc := range testData {
		// Initial flags
		if tc.params != nil {
			flag.Set("cors_preset", tc.params[0])
			flag.Set("cors_allow_origin", tc.params[1])
			flag.Set("cors_allow_origin_regex", tc.params[2])
			flag.Set("cors_allow_methods", tc.params[3])
			flag.Set("cors_allow_headers", tc.params[4])
			flag.Set("cors_expose_headers", tc.params[5])
		}
		flag.Set("cors_allow_credentials", strconv.FormatBool(tc.allowCredentials))

		gotRoute, err := MakeRouteConfig(&sc.ServiceInfo{Name: "test-api"})
		if tc.wantedError != "" {
			if err == nil || !strings.Contains(err.Error(), tc.wantedError) {
				t.Errorf("Test (%s): expected err: %v, got: %v", tc.desc, tc.wantedError, err)
			}
			continue
		}

		gotHost := gotRoute.GetVirtualHosts()
		if len(gotHost) != 1 {
			t.Errorf("Test (%s): got expected number of virtual host", tc.desc)
		}
		gotCors := gotHost[0].GetCors()
		if !reflect.DeepEqual(gotCors, tc.wantRoute) {
			t.Errorf("Test (%s): makeRouteConfig failed, got Cors: %s, want: %s", tc.desc, gotCors, tc.wantRoute)
		}
	}
}
