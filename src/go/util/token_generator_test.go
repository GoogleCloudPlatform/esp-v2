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

package util

import "testing"

func TestGenerateAccessToken(t *testing.T) {
	token, duration, err := GenerateAccessToken("testdata/key.json")
	if token == "" || duration == 0 || err != nil {
		t.Errorf("Test : Fail to make access token, got token: %s, duration: %v, err: %v", token, duration, err)
	}
}
