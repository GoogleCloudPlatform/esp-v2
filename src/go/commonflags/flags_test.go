// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commonflags

import (
	"flag"
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/options"
)

func TestDefaultCommonOptions(t *testing.T) {
	defaultOptions := options.DefaultCommonOptions()
	actualOptions := DefaultCommonOptionsFromFlags()

	if !reflect.DeepEqual(defaultOptions, actualOptions) {
		t.Fatalf("DefaultCommonOptions does not match DefaultCommonOptionsFromFlags:\nhave: %v\nwant: %v",
			defaultOptions, actualOptions)
	}
}

func TestServiceControlCredential(t *testing.T) {

	testData := []struct {
		desc                            string
		ServiceControlIamServiceAccount string
		ServiceControlIamDelegates      string
		BackendAuthIamServiceAccount    string
		BackendAuthIamDelegates         string
		wantedServiceControlCredentials *options.IAMCredentialsOptions
		wantedBackendAuthCredentials    *options.IAMCredentialsOptions
	}{
		{
			desc:                            "ServiceControlCredentials is created using ServiceControlIamServiceAccount and ServiceControlIamDelegates",
			ServiceControlIamServiceAccount: "ServiceControl@iam.com",
			ServiceControlIamDelegates:      "delegate_foo,delegate_bar",
			wantedServiceControlCredentials: &options.IAMCredentialsOptions{
				ServiceAccountEmail: "ServiceControl@iam.com",
				Delegates:           []string{"delegate_foo", "delegate_bar"},
			},
		},
		{
			desc:                            "ServiceControlCredentials is not set when ServiceControlIamServiceAccount is empty",
			ServiceControlIamDelegates:      "delegate_foo,delegate_bar",
			wantedServiceControlCredentials: nil,
		},
		{
			desc:                         "BackendAuthCredentials is created using BackendAuthIamServiceAccount and BackendAuthIamDelegates",
			BackendAuthIamServiceAccount: "BackendAuth@iam.com",
			BackendAuthIamDelegates:      "delegate_foo,delegate_bar",
			wantedBackendAuthCredentials: &options.IAMCredentialsOptions{
				ServiceAccountEmail: "BackendAuth@iam.com",
				Delegates:           []string{"delegate_foo", "delegate_bar"},
			},
		},
		{
			desc:                         "BackendAuthCredentials is not set when BackendAuthIamServiceAccount is empty",
			BackendAuthIamDelegates:      "delegate_foo,delegate_bar",
			wantedBackendAuthCredentials: nil,
		},
	}

	for _, tc := range testData {
		flag.Set("service_control_iam_service_account", tc.ServiceControlIamServiceAccount)
		flag.Set("service_control_iam_delegates", tc.ServiceControlIamDelegates)
		flag.Set("backend_auth_iam_service_account", tc.BackendAuthIamServiceAccount)
		flag.Set("backend_auth_iam_delegates", tc.BackendAuthIamDelegates)

		if gotServiceControlCredentials := DefaultCommonOptionsFromFlags().ServiceControlCredentials; !reflect.DeepEqual(gotServiceControlCredentials, tc.wantedServiceControlCredentials) {
			t.Errorf("ServiceControlCredential doesn't match:\nhave: %v\nwant: %v",
				gotServiceControlCredentials, tc.wantedServiceControlCredentials)
		}

	}
}

func TestResolveTracingProjectIdInCommonOptions(t *testing.T) {
	orig := *TracingProjectId
	defer flag.Set("tracing_project_id", orig)

	testCases := []struct {
		desc            string
		envResourceAttr string
		flagProjectId   string
		wantProjectId   string
	}{
		{
			desc:            "Default: no env var, flag is empty -> empty project ID",
			envResourceAttr: "",
			flagProjectId:   "",
			wantProjectId:   "",
		},
		{
			desc:            "OTEL_RESOURCE_ATTRIBUTES resolves project ID",
			envResourceAttr: "gcp.project.id=otel-project-id",
			flagProjectId:   "",
			wantProjectId:   "otel-project-id",
		},
		{
			desc:            "OTEL_RESOURCE_ATTRIBUTES overrides --tracing_project_id flag",
			envResourceAttr: "gcp.project.id=otel-override-project",
			flagProjectId:   "legacy-flag-project",
			wantProjectId:   "otel-override-project",
		},
		{
			desc:            "Fallback to --tracing_project_id when env var is unset",
			envResourceAttr: "",
			flagProjectId:   "legacy-flag-project",
			wantProjectId:   "legacy-flag-project",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Setenv("OTEL_RESOURCE_ATTRIBUTES", tc.envResourceAttr)
			flag.Set("tracing_project_id", tc.flagProjectId)

			opts := DefaultCommonOptionsFromFlags()
			if got := opts.TracingOptions.ProjectId; got != tc.wantProjectId {
				t.Errorf("DefaultCommonOptionsFromFlags().TracingOptions.ProjectId = %q, want %q", got, tc.wantProjectId)
			}
		})
	}
}
