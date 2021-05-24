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
	"fmt"
	"strings"
	"testing"
)

func TestValidateRegexProgramSize(t *testing.T) {
	testData := []struct {
		desc        string
		regex       string
		programSize int
		wantError   error
	}{
		{
			desc:        "oversize regex",
			regex:       "/**",
			programSize: 1,
			wantError:   fmt.Errorf("regex program size"),
		},
		{
			desc:        "invalid regex",
			regex:       "[",
			programSize: 1000,
			wantError:   fmt.Errorf("error parsing regexp: missing closing ]: `[`"),
		},
	}

	for _, tc := range testData {
		err := ValidateRegexProgramSize(tc.regex, tc.programSize)
		if err != nil {
			if tc.wantError == nil || !strings.Contains(err.Error(), tc.wantError.Error()) {
				t.Errorf("Test (%v): \n got %v \nwant %v", tc.desc, err, tc.wantError)
			}
		} else if tc.wantError != nil {
			t.Errorf("Test (%v): \n got %v \nwant %v", tc.desc, err, tc.wantError)
		}
	}
}
