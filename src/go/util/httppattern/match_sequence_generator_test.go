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
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
)

func parsePattern(pattern string) (string, string) {
	s := strings.Split(pattern, " ")
	return s[0], s[1]
}

// Though lookup method is not exposed so far, this test helps validate the
// underlying implementation(trie insertion) of MatchSequenceGenerator.
func TestMatchSequenceGeneratorRequestMatch(t *testing.T) {
	testCases := []struct {
		desc                 string
		requestToMatchResult map[string]struct {
			httpPattern     string
			variableBinding string
		}
	}{
		{
			desc: "WildCardMatchesRoot",
			requestToMatchResult: map[string]struct {
				httpPattern     string
				variableBinding string
			}{
				"GET /": {
					httpPattern: "GET /**",
				},
				"GET /a": {
					httpPattern: "GET /**",
				},
				"GET /a/": {
					httpPattern: "GET /**",
				},
			},
		},
		{
			desc: "WildCardMatches",
			requestToMatchResult: map[string]struct {
				httpPattern     string
				variableBinding string
			}{
				"GET /a/b": {
					httpPattern: "GET /a/**",
				},
				"GET /a/b/c": {
					httpPattern: "GET /a/**",
				},
				"GET /b/c": {
					httpPattern: "GET /b/*",
				},
				"GET /a/b/c/d": {
					httpPattern: "GET /a/**/d",
				},
				"GET /": {
					httpPattern: "GET /",
				},
				"GET b/c/d": {
					httpPattern: "",
				},
				"GET /c/u/d/v": {
					httpPattern: "GET /c/*/d/**",
				},
				"GET /c/v/d/w/x": {
					httpPattern: "GET /c/*/d/**",
				},
				"GET /c/x/y/d/z": {
					httpPattern: "",
				},
				"GET /c//v/d/w/x": {
					httpPattern: "",
				},
				"GET /c/x/d/e": {
					httpPattern: "GET /c/*/d/**",
				},
				"GET /c/f/d/e": {
					httpPattern: "GET /c/*/d/**",
				},
			},
		},
		{
			desc: "WildCardMethodMatches",
			requestToMatchResult: map[string]struct {
				httpPattern     string
				variableBinding string
			}{
				"GET /a/b": {
					httpPattern: "* /a/**",
				},
				"GET /a/b/c": {
					httpPattern: "* /a/**",
				},
				"GET /b/c": {
					httpPattern: "* /b/*",
				},
				"GET /": {
					httpPattern: "* /",
				},
				"POST /a/b": {
					httpPattern: "* /a/**",
				},
				"POST /a/b/c": {
					httpPattern: "* /a/**",
				},
				"POST /b/c": {
					httpPattern: "* /b/*",
				},
				"POST /": {
					httpPattern: "* /",
				},
				"DELETE /a/b": {
					httpPattern: "* /a/**",
				},
				"DELETE /a/b/c": {
					httpPattern: "* /a/**",
				},
				"DELETE /b/c": {
					httpPattern: "* /b/*",
				},
				"DELETE /": {
					httpPattern: "* /",
				},
			},
		},
		{
			desc: "VariableVariableBindings",
			requestToMatchResult: map[string]struct {
				httpPattern     string
				variableBinding string
			}{
				"GET /a/book/c/d/e": {
					httpPattern: "GET /a/{x}/c/d/e",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "book"
  }
]
`,
				},
				"GET /a/hello/b/world/c": {
					httpPattern: "GET /{x=a/*}/b/{y=*}/c",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "a/hello"
  },
  {
    "FieldPath": [
      "y"
    ],
    "Value": "world"
  }
]
`,
				},
				"GET /a/b/zoo/d/animal/tiger": {
					httpPattern: "GET /a/{x=b/*}/{y=d/**}",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "b/zoo"
  },
  {
    "FieldPath": [
      "y"
    ],
    "Value": "d/animal/tiger"
  }
]
`,
				},
				"GET /alpha/dog/beta/eat/bones/gamma": {
					httpPattern: "GET /alpha/{x=*}/beta/{y=**}/gamma",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "dog"
  },
  {
    "FieldPath": [
      "y"
    ],
    "Value": "eat/bones"
  }
]
`,
				},
				"GET /foo/a": {
					httpPattern: "GET /{x=*}/a",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "foo"
  }
]
`,
				},
				"GET /foo/bar/a/b": {
					httpPattern: "GET /{x=**}/a/b",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "foo/bar"
  }
]
`,
				},
				"GET /a/b/foo": {
					httpPattern: "GET /a/b/{x=*}",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "foo"
  }
]
`,
				},
				"GET /a/b/c/foo/bar/baz": {
					httpPattern: "GET /a/b/c/{x=**}",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "foo/bar/baz"
  }
]
`,
				},
				"GET /foo/d/e/f/bar/baz": {
					httpPattern: "GET /{x=*}/d/e/f/{y=**}",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "foo"
  },
  {
    "FieldPath": [
      "y"
    ],
    "Value": "bar/baz"
  }
]
`,
				},
			},
		},
		{
			desc: "PercentEscapesUnescapedForSingleSegment",
			requestToMatchResult: map[string]struct {
				httpPattern     string
				variableBinding string
			}{
				"GET /a/p%20q%2Fr/c": {
					httpPattern: "GET /a/{x}/c",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "p%20q%2Fr"
  }
]
`,
				},
			},
		},
		{
			desc: "PercentEscapesNotUnescapedForMultiSegment2",
			requestToMatchResult: map[string]struct {
				httpPattern     string
				variableBinding string
			}{
				"GET /a/p/foo%20foo/q/bar%2Fbar/c": {
					httpPattern: "GET /a/{x=p/*/q/*}/c",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "p/foo%20foo/q/bar%2Fbar"
  }
]
`,
				},
			},
		},
		{
			desc: "OnlyUnreservedCharsAreUnescapedForMultiSegmentMatch",
			requestToMatchResult: map[string]struct {
				httpPattern     string
				variableBinding string
			}{
				"GET /a/%21%23%24%26%27%28%29%2A%2B%2C%2F%3A%3B%3D%3F%40%5B%5D/c": {
					httpPattern: "GET /a/{x=**}/c",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "%21%23%24%26%27%28%29%2A%2B%2C%2F%3A%3B%3D%3F%40%5B%5D"
  }
]
`,
				},
			},
		},
		{
			desc: "VariableVariableBindingsWithCustomVerb",
			requestToMatchResult: map[string]struct {
				httpPattern     string
				variableBinding string
			}{
				"GET /a/world:verb": {
					httpPattern: "GET /a/{y=*}:verb",
					variableBinding: `
[
  {
    "FieldPath": [
      "y"
    ],
    "Value": "world"
  }
]
`,
				},
				"GET /a/d/animal/tiger:verb": {
					httpPattern: "GET /a/{y=d/**}:verb",
					variableBinding: `
[
  {
    "FieldPath": [
      "y"
    ],
    "Value": "d/animal/tiger"
  }
]
`,
				},
				"GET /foo/a:verb": {
					httpPattern: "GET /{x=*}/a:verb",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "foo"
  }
]
`,
				},
				"GET /foo/bar/baz/b:verb": {
					httpPattern: "GET /{x=**}/b:verb",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "foo/bar/baz"
  }
]
`,
				},
				"GET /e/foo/f:verb": {
					httpPattern: "GET /e/{x=*}/f:verb",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "foo"
  }
]

`,
				},
				"GET /g/foo/bar/h:verb": {
					httpPattern: "GET /g/{x=**}/h:verb",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "foo/bar"
  }
]

`,
				},
			},
		},
		{
			desc: "ConstantSuffixesWithVariable",
			requestToMatchResult: map[string]struct {
				httpPattern     string
				variableBinding string
			}{
				"GET /a/b/hello/world/c": {
					httpPattern: "GET /a/{x=b/**}",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "b/hello/world/c"
  }
]
`,
				},
				"GET /a/b/world/c/z": {
					httpPattern: "GET /a/{x=b/**}/z",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "b/world/c"
  }
]
`,
				},
				"GET /a/b/world/c/y/z": {
					httpPattern: "GET /a/{x=b/**}/y/z",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "b/world/c"
  }
]
`,
				},
				"GET /a/b/world/c:verb": {
					httpPattern: "GET /a/{x=b/**}:verb",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "b/world/c"
  }
]

`,
				},
				"GET /a/hello/b/world/c": {
					httpPattern: "GET /a/{x=**}",
					variableBinding: `[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "hello/b/world/c"
  }
]`,
				},
				"GET /c/hello/d/esp/world/e": {
					httpPattern: "GET /c/{x=*}/{y=d/**}/e",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "hello"
  },
  {
    "FieldPath": [
      "y"
    ],
    "Value": "d/esp/world"
  }
]
`,
				},
				"GET /c/hola/d/esp/mundo/e:verb": {
					httpPattern: "GET /c/{x=*}/{y=d/**}/e:verb",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "hola"
  },
  {
    "FieldPath": [
      "y"
    ],
    "Value": "d/esp/mundo"
  }
]
`,
				},
				"GET /f/foo/bar/baz/g": {
					httpPattern: "GET /f/{x=*}/{y=**}/g",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "foo"
  },
  {
    "FieldPath": [
      "y"
    ],
    "Value": "bar/baz"
  }
]
`,
				},
				"GET /f/foo/bar/baz/g:verb": {
					httpPattern: "GET /f/{x=*}/{y=**}/g:verb",
					variableBinding: `[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "foo"
  },
  {
    "FieldPath": [
      "y"
    ],
    "Value": "bar/baz"
  }
]
`,
				},
				"GET /a/b/foo/y/z/bar/baz/foo": {
					httpPattern: "GET /a/{x=b/*/y/z/**}/foo",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "b/foo/y/z/bar/baz"
  }
]
`,
				},
				"GET /a/b/foo/bar/baz/y/z/foo": {
					httpPattern: "GET /a/{x=b/*/**/y/z}/foo",
					variableBinding: `
[
  {
    "FieldPath": [
      "x"
    ],
    "Value": "b/foo/bar/baz/y/z"
  }
]
`,
				},
			},
		},
		{
			desc: "ConstantSuffixesWithVariable",
			requestToMatchResult: map[string]struct {
				httpPattern     string
				variableBinding string
			}{
				"GET /some/const:verb": {
					httpPattern: "GET /some/const:verb",
				},
				"GET /some/other:verb": {
					httpPattern: "GET /some/*:verb",
				},
				"GET /some/other:verb/": {
					httpPattern: "",
				},
				"GET /some/bar/foo:verb": {
					httpPattern: "GET /some/*/foo:verb",
				},
				"GET /some/foo1/foo2/foo:verb": {
					httpPattern: "",
				},
				"GET /some/foo/bar:verb": {
					httpPattern: "",
				},
				"GET /other/bar/foo:verb": {
					httpPattern: "GET /other/**:verb",
				},
				"GET /other/bar/foo/const:verb": {
					httpPattern: "GET /other/**/const:verb",
				},
			},
		},
		{
			desc: "CustomVerbMatch2",
			requestToMatchResult: map[string]struct {
				httpPattern     string
				variableBinding string
			}{
				"GET /some:verb/const:verb": {
					httpPattern: "GET /*/*:verb",
				},
			},
		},
		{
			desc: "CustomVerbMatch3",
			requestToMatchResult: map[string]struct {
				httpPattern     string
				variableBinding string
			}{
				// This is not custom verb since it was not configured.
				"GET /foo/other:verb": {
					httpPattern: "GET /foo/*",
				},
			},
		},
		{
			desc: "CustomVerbMatch4",
			requestToMatchResult: map[string]struct {
				httpPattern     string
				variableBinding string
			}{
				// last slash is before last colon.
				"GET /foo/other:verb/hello": {
					httpPattern: "GET /foo/*/hello",
				},
			},
		},
		{
			desc: "RejectPartialMatches",
			requestToMatchResult: map[string]struct {
				httpPattern     string
				variableBinding string
			}{
				"GET /prefix/middle/suffix": {
					httpPattern: "GET /prefix/middle/suffix",
				},
				"GET /prefix/middle": {
					httpPattern: "GET /prefix/middle",
				},
				"GET /prefix": {
					httpPattern: "GET /prefix",
				},
				"GET /prefix/middle/suffix/other": {
					httpPattern: "",
				},
				"GET /prefix/middle/other": {
					httpPattern: "",
				},
				"GET /prefix/other": {
					httpPattern: "",
				},
				"GET /other": {
					httpPattern: "",
				},
			},
		},
	}

	for _, tc := range testCases {
		sg := NewMatchSequenceGenerator()
		insertedTemplateToMethod := make(map[string]*Method)
		methodToUrlTemplate := make(map[*Method]string)
		for _, result := range tc.requestToMatchResult {
			if result.httpPattern == "" {
				continue
			}
			httpMethod, uriTemplate := parsePattern(result.httpPattern)
			if _, ok := insertedTemplateToMethod[result.httpPattern]; !ok {
				methodObj := &Method{
					HttpMethod:  httpMethod,
					UriTemplate: uriTemplate,
				}
				insertedTemplateToMethod[result.httpPattern] = methodObj
				methodToUrlTemplate[methodObj] = result.httpPattern
				if err := sg.Register(methodObj); err != nil {
					t.Fatalf("fail to insert %s: %v", result.httpPattern, err)
				}
			}
		}

		for req, result := range tc.requestToMatchResult {
			t.Run(tc.desc+" "+req, func(t *testing.T) {
				httpMethod, path := parsePattern(req)
				var getVariableBindings []*variableBinding
				var method *Method

				if method, getVariableBindings = sg.lookup(httpMethod, path); result.httpPattern == "" && method != nil || method != insertedTemplateToMethod[result.httpPattern] {
					t.Errorf("fail, request: %s, expect template: %s, get template: %s", req, result.httpPattern, methodToUrlTemplate[method])
				}

				if result.variableBinding != "" {
					bytes, _ := json.Marshal(getVariableBindings)
					getVariableBindingsStr := string(bytes)
					if err := util.JsonEqualWithNormalizer(result.variableBinding, getVariableBindingsStr, util.NormalizeJsonList); err != nil {
						t.Error(err)
					}
				}
			})
		}

	}
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
		pathToTemplate map[string]string
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
			pathToTemplate: map[string]string{
				"GET /foo/bar": "GET /foo/bar",
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
			pathToTemplate: map[string]string{
				"GET /a/x": "GET /a/{id}",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			sg := NewMatchSequenceGenerator()
			templateToMethod := map[string]*Method{}
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
				if r.wantRegisterPatternError == "" {
					templateToMethod[r.httpPattern] = method
				}
			}

			for req, template := range tc.pathToTemplate {
				httpMethod, path := parsePattern(req)
				if getMethod, _ := sg.lookup(httpMethod, path); getMethod != templateToMethod[template] {
					t.Errorf("expect matched template %s", template)
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
