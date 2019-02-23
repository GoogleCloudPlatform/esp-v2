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

package utils

import (
	"strings"
	"testing"
)

func TestDiffStrings(t *testing.T) {
	testData := []struct {
		x    string
		y    string
		want string
	}{
		{
			x:    "foo",
			y:    "foo",
			want: "",
		},
		{
			x:    "",
			y:    "bar",
			want: "+bar",
		},
		{
			x:    "foo\nbar",
			y:    "",
			want: "-foo\n-bar",
		},
		{
			x:    strings.Join([]string{"a", "b", "c"}, "\n"),
			y:    strings.Join([]string{"a", "b"}, "\n"),
			want: strings.Join([]string{"-c"}, "\n"),
		},
		{
			x:    strings.Join([]string{"1", "a", "b", "c"}, "\n"),
			y:    strings.Join([]string{"a", "b", "c", "A", "B"}, "\n"),
			want: strings.Join([]string{"-1", "+A", "+B"}, "\n"),
		},
		{
			x:    strings.Join([]string{"1", "2", "a", "b", "3", "c"}, "\n"),
			y:    strings.Join([]string{"a", "b", "c", "A", "B"}, "\n"),
			want: strings.Join([]string{"-1", "-2", "-3", "+A", "+B"}, "\n"),
		},
	}

	for _, tc := range testData {
		got := StringDiff(tc.x, tc.y)
		if got != tc.want {
			t.Errorf("got:\n%v\nwant:\n%v", got, tc.want)
		}
	}
}
