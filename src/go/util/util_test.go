// Copyright 2019 Google LLC
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

func TestMaybeTruncateSpanName(t *testing.T) {
	spaNameWith128Bytes := ""
	for i := 0; i < 129; i += 1 {
		spaNameWith128Bytes += "x"
	}

	testCases := []struct {
		desc         string
		spanName     string
		wantSpanName string
	}{
		{
			desc:         "spanName <= 128 bytes not needed to truncate",
			spanName:     spaNameWith128Bytes[:128],
			wantSpanName: spaNameWith128Bytes[:128],
		},
		{
			desc:         "spanName > 128 byte will be truncated and appended with ...",
			spanName:     spaNameWith128Bytes[:128+1],
			wantSpanName: spaNameWith128Bytes[:125] + "...",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			getSpanName := MaybeTruncateSpanName(tc.spanName)
			if getSpanName != tc.wantSpanName {
				t.Errorf("expect %s, get: %s", tc.wantSpanName, getSpanName)
			}
		})

	}
}
