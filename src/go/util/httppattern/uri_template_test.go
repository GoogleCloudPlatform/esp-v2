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

package httppattern

import (
	"encoding/json"
	"testing"
)

func TestReplaceVariableFieldInUriTemplateRebuild(t *testing.T) {
	testCases := []string{
		"/shelves/{shelf}/books/{book}",
		"/shelves/**",
		"/**",
		"/*",
		"/a:foo",
		"/a/b/c:foo",
		"/*/**",
		"/a/{a.b.c}",
		"/a/{a.b.c=*}",
		"/a/{b=*}",
		"/a/{b=**}",
		"/a/{b=c/*}",
		"/a/{b=c/*/d}",
		"/a/{b=c/**}",
		"/a/{b=c/**}/d/e",
		"/a/{b=c/*/d}/e",
		"/a/{b=c/*/d}/e:verb",
		"/*:verb",
		"/**:verb",
		"/{a}:verb",
		"/a/b/*:verb",
		"/a/b/**:verb",
		"/a/b/{a}:verb",
		"/{x}",
		"/{x.y.z}",
		"/{x=*}",
		"/{x=a/*}",
		"/{x.y.z=*/a/b}/c",
		"/{x=**}",
		"/{x.y.z=**}",
		"/{x.y.z=a/**/b}",
		"/{x.y.z=a/**/b}/c/d",
		"/{x}:verb",
		"/{x.y.z}:verb",
		"/{x.y.z=*/*}:verb",
		"/{x=**}:myverb",
		"/{x.y.z=**}:myverb",
		"/{x.y.z=a/**/b}:custom",
		"/{x.y.z=a/**/b}/c/d:custom",
		"/",
		"/a/*:verb",
		"/a/**:verb",
		"/a/{b=*}/**:verb",
	}

	uriTemplateStrEqual := func(get string, want string) bool {
		getUriTemplate := ParseUriTemplate(get)
		wantUriTemplate := ParseUriTemplate(want)
		getUriTemplateBytes, _ := json.Marshal(getUriTemplate)
		wantUriTemplateBytes, _ := json.Marshal(wantUriTemplate)
		return string(getUriTemplateBytes) == string(wantUriTemplateBytes)
	}

	for _, tc := range testCases {
		// Some uri templates are equal in syntax through not equal in string comparison.
		if getUriTemplate, _ := ReplaceVariableFieldInUriTemplate(tc, nil); !uriTemplateStrEqual(getUriTemplate, tc) {
			t.Errorf("fail to rebuild, wante uriTemplate: %s, get uriTemplate: %s", tc, getUriTemplate)
		}
	}
}

func TestReplaceVariableFieldInUriTemplate(t *testing.T) {
	testCases := []struct {
		desc            string
		uriTemplate     string
		varReplace      map[string]string
		wantUriTemplate string
	}{
		{
			desc:        "replace with {var} syntax",
			uriTemplate: "/shelves/{shelf}/books/{book}",
			varReplace: map[string]string{
				"shelf": "SHELF",
				"book":  "BOOK",
			},
			wantUriTemplate: "/shelves/{SHELF=*}/books/{BOOK=*}",
		},
		{
			desc:        "replace with {var=*} syntax",
			uriTemplate: "/a/{b=*}",
			varReplace: map[string]string{
				"a": "FOO",
				"b": "BAR",
			},
			wantUriTemplate: "/a/{BAR=*}",
		},
		{
			desc:        "replace with {a.b.c=*} syntax",
			uriTemplate: "/a/{a.b.c=*}",
			varReplace: map[string]string{
				"a": "FOO",
				"c": "BAR",
			},
			wantUriTemplate: "/a/{FOO.b.BAR=*}",
		},
		{
			desc:        "replace with {a.b.c=x/**} syntax",
			uriTemplate: "/a/{b.c=x/**}",
			varReplace: map[string]string{
				"b": "FOO",
				"c": "BAR",
			},
			wantUriTemplate: "/a/{FOO.BAR=x/**}",
		},
		{
			desc:        "replace with verb syntax",
			uriTemplate: "/a/{b=c/**}:verb",
			varReplace: map[string]string{
				"b": "BAR",
			},
			wantUriTemplate: "/a/{BAR=c/**}:verb",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			if getUriTemplate, _ := ReplaceVariableFieldInUriTemplate(tc.uriTemplate, tc.varReplace); getUriTemplate != tc.wantUriTemplate {
				t.Errorf("fail to replace variable field, wante uriTemplate: %s, get uriTemplate: %s", tc.wantUriTemplate, getUriTemplate)
			}
		})

	}
}
