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

func TestMatchSequenceGeneratorRegisterErrorHttpPattern(t *testing.T) {
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
			sg := NewMatchSequenceGenerator()
			httpMethod, uriTemplate := parsePattern(tc)
			if err := sg.Register(&Method{HttpMethod: httpMethod, UriTemplate: uriTemplate}); err == nil || strings.Index(err.Error(), "invalid url template") == -1 {
				t.Errorf("expect failing to register the template: %s but it succeed", tc)
			}
		})
	}
}

func TestMatchSequenceGeneratorDuplicateHttpPattern(t *testing.T) {
	testCases := []struct {
		desc     string
		register []struct {
			httpPattern              string
			wantRegisterPatternError string
		}
	}{
		{
			desc: "duplicate in constant segments",
			register: []struct {
				httpPattern              string
				wantRegisterPatternError string
			}{
				{
					httpPattern: "GET /foo/bar",
				},
				{
					httpPattern:              "GET /foo/bar",
					wantRegisterPatternError: "duplicate http pattern `GET /foo/bar`",
				},
			},
		},
		{
			desc: "duplicate in variable segments",
			register: []struct {
				httpPattern              string
				wantRegisterPatternError string
			}{
				{
					httpPattern: "GET /a/{id}",
				},
				{
					httpPattern:              "GET /a/{name}",
					wantRegisterPatternError: "duplicate http pattern `GET /a/{name}`",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			sg := NewMatchSequenceGenerator()
			for _, r := range tc.register {
				httpMethod, uriTemplate := parsePattern(r.httpPattern)
				method := &Method{
					UriTemplate: uriTemplate,
					HttpMethod:  httpMethod,
				}
				if err := sg.Register(method); err == nil && r.wantRegisterPatternError != "" {
					t.Errorf("expect registering http pattern error: %s, but get success", r.wantRegisterPatternError)
				} else if err != nil && err.Error() != r.wantRegisterPatternError {
					if r.wantRegisterPatternError == "" {
						t.Errorf("expect registering http pattern error: %s, but get error: %v", r.wantRegisterPatternError, err)
					} else {
						t.Errorf("expect succeful registering http pattern,b ut get error: %v", err)
					}

				}
			}
		})
	}
}

func TestMatchSequenceGeneratorSort(t *testing.T) {
	testCases := []struct {
		desc              string
		httpPatterns      []string
		sortedHttpPattern []string
	}{
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
			sg := NewMatchSequenceGenerator()
			for _, hp := range tc.httpPatterns {
				httpMethod, uriTemplate := parsePattern(hp)
				if err := sg.Register(&Method{
					UriTemplate: uriTemplate,
					HttpMethod:  httpMethod,
				}); err != nil {
					t.Errorf("fail to register httpPatternL `%s`: %v", hp, err)
				}
			}
			res := sg.Generate()
			if len(*res) != len(tc.sortedHttpPattern) {
				t.Fatalf("different size of http pattern, expect: %v, get: %v", len(tc.sortedHttpPattern), len(*res))
			}
			for idx, r := range *res {
				if getHttpPattern := fmt.Sprintf("%s %s", r.HttpMethod, r.UriTemplate); getHttpPattern != tc.sortedHttpPattern[idx] {
					t.Errorf("expect http pattern: % s, get http pattern: %s", tc.sortedHttpPattern[idx], getHttpPattern)
				}
			}
		})
	}
}
