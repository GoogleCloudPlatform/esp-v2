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
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/api-proxy/src/go/options"
)

func TestDefaultCommonOptions(t *testing.T) {
	defaultOptions := options.DefaultCommonOptions()
	actualOptions := DefaultCommonOptionsFromFlags()

	if !reflect.DeepEqual(defaultOptions, actualOptions) {
		t.Fatalf("DefaultCommonOptions does not match DefaultCommonOptionsFromFlags:\nhave: %v\nwant: %v",
			defaultOptions, actualOptions)
	}
}
