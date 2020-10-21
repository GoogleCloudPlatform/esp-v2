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
	"fmt"
	"strings"
	"testing"
)

func parsePattern(pattern string) (string, string) {
	s := strings.Split(pattern, " ")
	return s[0], s[1]
}

func TestSortErrorHttpPattern(t *testing.T) {
	testCases := []string{
		"GET /a{x=b/**}/bb/{y=*}",
		"GET /a{x=b/**}/{y=**}",
		"GET /a{x=b/**}/bb/{y=**}",
		"GET /a/**/*",
		"GET /a/**/foo/*",
		"GET /a/**/**",
		"GET /a/**/foo/**",
		"GET /**/**",
	}
	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			methods := &MethodSlice{}
			httpMethod, uriTemplate := parsePattern(tc)
			methods.AppendMethod(&Method{
				UriTemplate: uriTemplate,
				HttpMethod:  httpMethod,
			})

			if err := Sort(methods); err == nil || strings.Index(err.Error(), "invalid url template") == -1 {
				t.Errorf("expect failing to insert the template: %s but it succeed", tc)
			}
		})
	}
}

func TestSortDuplicateHttpPattern(t *testing.T) {
	testCases := []struct {
		desc         string
		httpPatterns []string
		wantError    string
	}{
		{
			desc: "duplicate in constant segments",
			httpPatterns: []string{
				"GET /foo/bar",
				"GET /foo/bar",
			},
			wantError: "duplicate http pattern `GET /foo/bar`",
		},
		{
			desc: "duplicate in variable segments",
			httpPatterns: []string{
				"GET /a/{id}",
				"GET /a/{name}",
			},
			wantError: "duplicate http pattern `GET /a/{name}`",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {

			methods := &MethodSlice{}
			for _, hp := range tc.httpPatterns {
				httpMethod, uriTemplate := parsePattern(hp)
				methods.AppendMethod(&Method{
					UriTemplate: uriTemplate,
					HttpMethod:  httpMethod,
				})
			}

			if err := Sort(methods); err != nil {
				if err.Error() != tc.wantError {
					t.Errorf("expect inserting http pattern error: %s, but get error: %v", tc.wantError, err)
				}
			} else {
				t.Errorf("expect inserting http pattern error: %s, but get success", tc.wantError)
			}
		})
	}
}

func TestSort(t *testing.T) {
	testCases := []struct {
		desc              string
		httpPatterns      []string
		sortedHttpPattern []string
	}{
		{
			desc: "wildcard prefix",
			httpPatterns: []string{
				"GET /**",
				"GET /**/a",
				"GET /**:verb",
			},
			sortedHttpPattern: []string{
				"GET /**/a",
				"GET /**:verb",
				"GET /**",
			},
		},
		{
			desc: "constant prefix",
			httpPatterns: []string{
				"GET /foo",
				"GET /foo/a",
				"GET /foo:verb",
			},
			sortedHttpPattern: []string{
				"GET /foo",
				"GET /foo/a",
				"GET /foo:verb",
			},
		},
		{
			desc: "root prefix",
			httpPatterns: []string{
				"GET /",
				"GET /**",
				"GET /foo",
			},
			sortedHttpPattern: []string{
				"GET /",
				"GET /foo",
				"GET /**",
			},
		},
		{
			desc: "constant with wildcard",
			httpPatterns: []string{
				"GET /a/**/b/c",
				"GET /a/**/b",
				"GET /a/**",
				"GET /a/*/b/c",
				"GET /a/*/b",
				"GET /a/*",
				"GET /a/x/b/c",
				"GET /a/x/b",
				"GET /a/x",
				"GET /a",
			},
			sortedHttpPattern: []string{
				"GET /a",
				"GET /a/x",
				"GET /a/x/b",
				"GET /a/x/b/c",
				"GET /a/*",
				"GET /a/*/b",
				"GET /a/*/b/c",
				"GET /a/**/b",
				"GET /a/**/b/c",
				"GET /a/**",
			},
		},
		{
			desc: "various variable bindings",
			httpPatterns: []string{
				"GET /a/{x}/c/d/e",
				"GET /{x=a/*}/b/{y=*}/c",
				"GET /a/{x=b/*}/{y=d/**}",
				"GET /alpha/{x=*}/beta/{y=**}/gamma",
				"GET /{x=*}/a",
				"GET /{x=**}/a/b",
				"GET /a/b/{x=*}",
				"GET /a/b/c/{x=**}",
				"GET /{x=*}/d/e/f/{y=**}",
			},
			sortedHttpPattern: []string{
				"GET /a/b/c/{x=**}",
				"GET /a/b/{x=*}",
				"GET /a/{x=b/*}/{y=d/**}",
				"GET /{x=a/*}/b/{y=*}/c",
				"GET /a/{x}/c/d/e",
				"GET /alpha/{x=*}/beta/{y=**}/gamma",
				"GET /{x=*}/a",
				"GET /{x=*}/d/e/f/{y=**}",
				"GET /{x=**}/a/b",
			},
		},
		{
			desc: "variable bindings with/without verb",
			httpPatterns: []string{
				"GET /a/{y=*}:verb",
				"GET /a/{y=d/**}:verb",
				"GET /{x=*}/a:verb",
				"GET /{x=**}/b:verb",
				"GET /g/{x=**}/h:verb",
				"GET /a/{y=*}",
				"GET /a/{y=d/**}",
				"GET /{x=*}/a",
				"GET /{x=**}/b",
				"GET /g/{x=**}/h",
			},
			sortedHttpPattern: []string{
				"GET /a/{y=d/**}:verb",
				"GET /a/{y=d/**}",
				"GET /a/{y=*}",
				"GET /a/{y=*}:verb",
				"GET /g/{x=**}/h",
				"GET /g/{x=**}/h:verb",
				"GET /{x=*}/a",
				"GET /{x=*}/a:verb",
				"GET /{x=**}/b",
				"GET /{x=**}/b:verb",
			},
		},
		{
			// This is not required. Only for unit test.
			desc: "deterministic order of http methods",
			httpPatterns: []string{
				"GET /a",
				"PUT /a",
				"DELETE /a",
				"POST /a",
				"* /a",
			},
			sortedHttpPattern: []string{
				"DELETE /a",
				"GET /a",
				"POST /a",
				"PUT /a",
				"* /a",
			},
		},
		{
			// This is not required. Only for unit test.
			desc: "deterministic order of exact match url template",
			httpPatterns: []string{
				"GET /b/c/a",
				"GET /b/a/c",
				"GET /a/b/c",
				"GET /a/c/b",
			},
			sortedHttpPattern: []string{
				"GET /a/b/c",
				"GET /a/c/b",
				"GET /b/a/c",
				"GET /b/c/a",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			methods := &MethodSlice{}
			for _, hp := range tc.httpPatterns {
				httpMethod, uriTemplate := parsePattern(hp)
				methods.AppendMethod(&Method{
					UriTemplate: uriTemplate,
					HttpMethod:  httpMethod,
				})
			}

			if err := Sort(methods); err != nil {
				t.Fatalf("fail to sort the methods with error: %v", err)
			}

			for idx, r := range *methods {
				if getHttpPattern := fmt.Sprintf("%s %s", r.HttpMethod, r.UriTemplate); getHttpPattern != tc.sortedHttpPattern[idx] {
					t.Errorf("expect http pattern: % s, get http pattern: %s", tc.sortedHttpPattern[idx], getHttpPattern)
				}
			}
		})
	}
}
