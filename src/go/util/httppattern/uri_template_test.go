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
		return getUriTemplate.Equal(wantUriTemplate)
	}

	for _, tc := range testCases {
		// Some uri templates are equal in syntax through not equal in string comparison.
		if getUriTemplate := ParseUriTemplate(tc).String(); !uriTemplateStrEqual(getUriTemplate, tc) {
			t.Errorf("fail to rebuild, want uriTemplate: %s, get uriTemplate: %s", tc, getUriTemplate)
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
			uriTemplate := ParseUriTemplate(tc.uriTemplate)
			uriTemplate.ReplaceVariableField(tc.varReplace)
			if getUriTemplate := uriTemplate.String(); getUriTemplate != tc.wantUriTemplate {
				t.Errorf("fail to replace variable field, wante uriTemplate: %s, get uriTemplate: %s", tc.wantUriTemplate, getUriTemplate)
			}
		})

	}
}

func TestUriTemplateRegex(t *testing.T) {
	testData := []struct {
		desc        string
		uri         string
		wantMatcher string
		wantError   string
	}{
		{
			desc:        "No path params",
			uri:         "/shelves",
			wantMatcher: `^/shelves$`,
		},
		{
			desc:        "Path params with fieldpath-only bindings",
			uri:         "/shelves/{shelf_id}/books/{book.id}",
			wantMatcher: `^/shelves/[^\/]+/books/[^\/]+$`,
		},
		{
			desc:        "Path params with fieldpath-only bindings and verb",
			uri:         "/shelves/{shelf_id}/books/{book.id}:checkout",
			wantMatcher: `^/shelves/[^\/]+/books/[^\/]+:checkout$`,
		},
		{
			desc:        "Path param with wildcard segments",
			uri:         "/test/*/test/**",
			wantMatcher: `^/test/[^\/]+/test/.*$`,
		},
		{
			desc:        "Path param with wildcard segments and verb",
			uri:         "/test/*/test/**:upload",
			wantMatcher: `^/test/[^\/]+/test/.*:upload$`,
		},
		{
			desc:        "Path param with wildcard in segment binding",
			uri:         "/test/{x=*}/test/{y=**}",
			wantMatcher: `^/test/[^\/]+/test/.*$`,
		},
		{
			desc:        "Path param with mixed wildcards",
			uri:         "/test/{name=*}/test/**",
			wantMatcher: `^/test/[^\/]+/test/.*$`,
		},
		{
			desc:        "Path params with full segment binding",
			uri:         "/v1/{name=books/*}",
			wantMatcher: `^/v1/books/[^\/]+$`,
		},
		{
			desc:        "Path params with multiple field path segment bindings",
			uri:         "/v1/{test=a/b/*}/route/{resource_id=shelves/*/books/**}:upload",
			wantMatcher: `^/v1/a/b/[^\/]+/route/shelves/[^\/]+/books/.*:upload$`,
		},
	}

	for _, tc := range testData {
		t.Run(tc.desc, func(t *testing.T) {
			uriTemplate := ParseUriTemplate(tc.uri)
			if uriTemplate == nil {
				t.Fatalf("fail to parse uri template %s", tc.uri)
			}

			if got := uriTemplate.Regex(); tc.wantMatcher != got {
				t.Errorf("Test (%v): \n got %v \nwant %v", tc.desc, got, tc.wantMatcher)
			}
		})
	}
}
