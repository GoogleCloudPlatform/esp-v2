// Copyright 2020 Google LLC
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

package util

import (
	"testing"
)

func TestJwtProviderClusterName(t *testing.T) {
	testCase := struct {
		address    string
		wantedName string
	}{
		address:    "localhost.com:8000",
		wantedName: "jwt-provider-cluster-localhost.com:8000",
	}

	if gotName := JwtProviderClusterName(testCase.address); gotName != testCase.wantedName {
		t.Errorf("fail to create jwt provider cluster name, expected: %s, got: %s", testCase.wantedName, gotName)
	}
}

func TestBackendClusterName(t *testing.T) {
	testCase := struct {
		address    string
		wantedName string
	}{
		address:    "localhost.com:8000",
		wantedName: "backend-cluster-localhost.com:8000",
	}

	if gotName := BackendClusterName(testCase.address); gotName != testCase.wantedName {
		t.Errorf("fail to create backend cluster name, expected: %s, got: %s", testCase.wantedName, gotName)
	}
}
