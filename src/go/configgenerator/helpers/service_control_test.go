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

package helpers

import (
	"net/url"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
	"github.com/google/go-cmp/cmp"
	confpb "google.golang.org/genproto/googleapis/api/serviceconfig"
)

func TestParseServiceControlURLFromOPConfig(t *testing.T) {
	testData := []struct {
		desc            string
		serviceConfigIn *confpb.Service
		optsIn          options.ConfigGeneratorOptions
		wantURI         url.URL
	}{
		{
			desc: "URL from service config by default",
			serviceConfigIn: &confpb.Service{
				Control: &confpb.Control{
					Environment: "https://staging-servicecontrol.sandbox.googleapis.com",
				},
			},
			wantURI: url.URL{
				Scheme: "https",
				Host:   "staging-servicecontrol.sandbox.googleapis.com:443",
			},
		},
		{
			desc: "option overrides service config",
			serviceConfigIn: &confpb.Service{
				Control: &confpb.Control{
					// not used due to non-empty option
					Environment: "https://staging-servicecontrol.sandbox.googleapis.com",
				},
			},
			optsIn: options.ConfigGeneratorOptions{
				ServiceControlURL: "https://servicecontrol.googleapis.com",
			},
			wantURI: url.URL{
				Scheme: "https",
				Host:   "servicecontrol.googleapis.com:443",
			},
		},
		{
			desc:            "Empty inputs results in empty URL",
			serviceConfigIn: &confpb.Service{},
			wantURI:         url.URL{},
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			gotURI, err := ParseServiceControlURLFromOPConfig(tc.serviceConfigIn, tc.optsIn)
			if err != nil {
				t.Fatalf("ParseServiceControlURLFromOPConfig(...) has wrong error, got: %v, want no error", err)
			}

			if diff := cmp.Diff(tc.wantURI, gotURI); diff != "" {
				t.Errorf("processServiceControlURL(...) has unexpected diff for ServiceControlURI (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseServiceControlURLFromOPConfig_BadInput(t *testing.T) {
	testData := []struct {
		desc            string
		serviceConfigIn *confpb.Service
		optsIn          options.ConfigGeneratorOptions
		wantErr         string
	}{
		{
			desc: "url parsing fails",
			serviceConfigIn: &confpb.Service{
				Control: &confpb.Control{
					Environment: "https://[::1:80",
				},
			},
			wantErr: `parse "https://[::1:80": missing ']' in host`,
		},
		{
			desc: "url should not have path segment",
			serviceConfigIn: &confpb.Service{
				Control: &confpb.Control{
					Environment: "https://servicecontrol.googleapis.com/v1/services",
				},
			},
			wantErr: `should not have path part: /v1/services`,
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			_, err := ParseServiceControlURLFromOPConfig(tc.serviceConfigIn, tc.optsIn)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("ParseServiceControlURLFromOPConfig(...) has wrong error, got: %v, want: %q", err, tc.wantErr)
			}
		})
	}
}
