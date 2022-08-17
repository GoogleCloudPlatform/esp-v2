// Copyright 2022 Google LLC
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

package filterconfig

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/tests/utils"
	ahpb "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/encoding/prototext"
)

func TestPreserveDefaultHttpBinding(t *testing.T) {
	testData := []struct {
		desc         string
		originalDesc string
		wantDesc     string
	}{
		{
			// Add the default http binding if it's not present.
			desc: "default http binding is not present",
			originalDesc: `
				selector: "package.name.Service.Method"
				post: "/v1/Service/Method"
			`,
			wantDesc: `
				selector: "package.name.Service.Method"
				post: "/v1/Service/Method"
				additional_bindings: {
					post: "/package.name.Service/Method"
					body: "*"
				}
			`,
		},
		{
			// Do not add the default binding if it's identitical to the primary
			// binding. Difference in selector and additional_bindings is ignored.
			desc: "default http binding is not present",
			originalDesc: `
				selector: "package.name.Service.Method"
				post: "/package.name.Service/Method"
				body: "*"
				additional_bindings: {
					post: "/v1/Service/Method"
				}
			`,
			wantDesc: `
				selector: "package.name.Service.Method"
				post: "/package.name.Service/Method"
				body: "*"
				additional_bindings: {
					post: "/v1/Service/Method"
				}
			`,
		},
		{
			// Do not add the default binding if it's identitical to any existing
			// additional binding.
			desc: "default http binding is not present",
			originalDesc: `
				selector: "package.name.Service.Method"
				post: "/package.name.Service/Method"
				body: "*"
				additional_bindings: {
					post: "/v1/Service/Method"
				}
			`,
			wantDesc: `
				selector: "package.name.Service.Method"
				post: "/package.name.Service/Method"
				body: "*"
				additional_bindings: {
					post: "/v1/Service/Method"
				}
			`,
		},
	}

	for _, tc := range testData {
		got := &ahpb.HttpRule{}
		if err := prototext.Unmarshal([]byte(tc.originalDesc), got); err != nil {
			fmt.Println("failed to unmarshal originalDesc: ", err)
		}

		preserveDefaultHttpBinding(got, "/package.name.Service/Method")
		want := &ahpb.HttpRule{}
		if err := prototext.Unmarshal([]byte(tc.wantDesc), want); err != nil {
			fmt.Println("failed to unmarshal wantDesc: ", err)
		}

		if diff := utils.ProtoDiff(want, got); diff != "" {
			t.Errorf("Result is not the same: diff (-want +got):\n%v", diff)
		}
	}
}
