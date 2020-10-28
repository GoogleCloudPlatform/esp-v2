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
	"testing"

	"github.com/GoogleCloudPlatform/esp-v2/src/go/util"
)

func TestUriTemplateParse(t *testing.T) {
	testCases := []struct {
		desc            string
		UriTemplate     string
		wantUriTemplate string
	}{
		{
			desc:        "ParseTest1",
			UriTemplate: "/shelves/{shelf}/books/{book}",
			wantUriTemplate: `
{
   "Origin":"/shelves/{shelf}/books/{book}",
   "Segments":[
      "shelves",
      "*",
      "books",
      "*"
   ],
   "Variables":[
      {
         "EndSegment":2,
         "FieldPath":[
            "shelf"
         ],
         "HasDoubleWildCard":false,
         "StartSegment":1
      },
      {
         "EndSegment":4,
         "FieldPath":[
            "book"
         ],
         "HasDoubleWildCard":false,
         "StartSegment":3
      }
   ],
   "Verb":""
}
`,
		},
		{
			desc:        "ParseTest2",
			UriTemplate: "/shelves/**",
			wantUriTemplate: `
{
   "Origin":"/shelves/**",
   "Segments":[
      "shelves",
      "**"
   ],
   "Variables":null,
   "Verb":""
}
`,
		},
		{
			desc:        "ParseTest3a",
			UriTemplate: "/**",
			wantUriTemplate: `
{
   "Origin":"/**",
   "Segments":[
      "**"
   ],
   "Variables":null,
   "Verb":""
}
`,
		},
		{
			desc:        "ParseTest3b",
			UriTemplate: "/*",
			wantUriTemplate: `
{
   "Origin":"/*",
   "Segments":[
      "*"
   ],
   "Variables":null,
   "Verb":""
}
`,
		},
		{
			desc:        "ParseTest4a",
			UriTemplate: "/a:foo",
			wantUriTemplate: `
{
   "Origin":"/a:foo",
   "Segments":[
      "a"
   ],
   "Variables":null,
   "Verb":"foo"
}
`,
		},
		{
			desc:        "ParseTest4b",
			UriTemplate: "/a/b/c:foo",
			wantUriTemplate: `
{
   "Origin":"/a/b/c:foo",
   "Segments":[
      "a",
      "b",
      "c"
   ],
   "Variables":null,
   "Verb":"foo"
} 
`,
		},
		{
			desc:        "ParseTest5",
			UriTemplate: "/*/**",
			wantUriTemplate: `
{
   "Origin":"/*/**",
   "Segments":[
      "*",
      "**"
   ],
   "Variables":null,
   "Verb":""
}
`,
		},
		{
			desc:        "ParseTest6",
			UriTemplate: "/*/a/**",
			wantUriTemplate: `
{
   "Origin":"/*/a/**",
   "Segments":[
      "*",
      "a",
      "**"
   ],
   "Variables":null,
   "Verb":""
}
`,
		},
		{
			desc:        "ParseTest7",
			UriTemplate: "/a/{a.b.c}",
			wantUriTemplate: `
{
   "Origin":"/a/{a.b.c}",
   "Segments":[
      "a",
      "*"
   ],
   "Variables":[
      {
         "EndSegment":2,
         "FieldPath":[
            "a",
            "b",
            "c"
         ],
         "HasDoubleWildCard":false,
         "StartSegment":1
      }
   ],
   "Verb":""
}
`,
		},
		{
			desc:        "ParseTest8",
			UriTemplate: "/a/{a.b.c=*}",
			wantUriTemplate: `
{
   "Origin":"/a/{a.b.c=*}",
   "Segments":[
      "a",
      "*"
   ],
   "Variables":[
      {
         "EndSegment":2,
         "FieldPath":[
            "a",
            "b",
            "c"
         ],
         "HasDoubleWildCard":false,
         "StartSegment":1
      }
   ],
   "Verb":""
}
`,
		},
		{
			desc:        "ParseTest9",
			UriTemplate: "/a/{b=*}",
			wantUriTemplate: `
{
   "Origin":"/a/{b=*}",
   "Segments":[
      "a",
      "*"
   ],
   "Variables":[
      {
         "EndSegment":2,
         "FieldPath":[
            "b"
         ],
         "HasDoubleWildCard":false,
         "StartSegment":1
      }
   ],
   "Verb":""
}
`,
		},
		{
			desc:        "ParseTest10",
			UriTemplate: "/a/{b=**}",
			wantUriTemplate: `
{
   "Origin":"/a/{b=**}",
   "Segments":[
      "a",
      "**"
   ],
   "Variables":[
      {
         "EndSegment":-1,
         "FieldPath":[
            "b"
         ],
         "HasDoubleWildCard":true,
         "StartSegment":1
      }
   ],
   "Verb":""
}
 `,
		},
		{
			desc:        "ParseTest11",
			UriTemplate: "/a/{b=c/*}",
			wantUriTemplate: `
{
   "Origin":"/a/{b=c/*}",
   "Segments":[
      "a",
      "c",
      "*"
   ],
   "Variables":[
      {
         "EndSegment":3,
         "FieldPath":[
            "b"
         ],
         "HasDoubleWildCard":false,
         "StartSegment":1
      }
   ],
   "Verb":""
}
 `,
		},
		{
			desc:        "ParseTest12",
			UriTemplate: "/a/{b=c/*/d}",
			wantUriTemplate: `
{
   "Origin":"/a/{b=c/*/d}",
   "Segments":[
      "a",
      "c",
      "*",
      "d"
   ],
   "Variables":[
      {
         "EndSegment":4,
         "FieldPath":[
            "b"
         ],
         "HasDoubleWildCard":false,
         "StartSegment":1
      }
   ],
   "Verb":""
}
 `,
		},
		{
			desc:        "ParseTest13",
			UriTemplate: "/a/{b=c/**}",
			wantUriTemplate: `
{
   "Origin":"/a/{b=c/**}",
   "Segments":[
      "a",
      "c",
      "**"
   ],
   "Variables":[
      {
         "EndSegment":-1,
         "FieldPath":[
            "b"
         ],
         "HasDoubleWildCard":true,
         "StartSegment":1
      }
   ],
   "Verb":""
}
`,
		},
		{
			desc:        "ParseTest14",
			UriTemplate: "/a/{b=c/**}/d/e",
			wantUriTemplate: `
{
   "Origin":"/a/{b=c/**}/d/e",
   "Segments":[
      "a",
      "c",
      "**",
      "d",
      "e"
   ],
   "Variables":[
      {
         "EndSegment":-3,
         "FieldPath":[
            "b"
         ],
         "HasDoubleWildCard":true,
         "StartSegment":1
      }
   ],
   "Verb":""
}
`,
		},
		{
			desc:        "ParseTest15",
			UriTemplate: "/a/{b=c/*/d}/e",
			wantUriTemplate: `
{
   "Origin":"/a/{b=c/*/d}/e",
   "Segments":[
      "a",
      "c",
      "*",
      "d",
      "e"
   ],
   "Variables":[
      {
         "EndSegment":4,
         "FieldPath":[
            "b"
         ],
         "HasDoubleWildCard":false,
         "StartSegment":1
      }
   ],
   "Verb":""
}
`,
		},
		{
			desc:        "ParseTest16",
			UriTemplate: "/a/{b=c/*/d}/e:verb",
			wantUriTemplate: `
{
   "Origin":"/a/{b=c/*/d}/e:verb",
   "Segments":[
      "a",
      "c",
      "*",
      "d",
      "e"
   ],
   "Variables":[
      {
         "EndSegment":4,
         "FieldPath":[
            "b"
         ],
         "HasDoubleWildCard":false,
         "StartSegment":1
      }
   ],
   "Verb":"verb"
}
`,
		},
		{
			desc:        "CustomVerbTests-1",
			UriTemplate: "/*:verb",
			wantUriTemplate: `
{
   "Origin":"/*:verb",
   "Segments":[
      "*"
   ],
   "Variables":null,
   "Verb":"verb"
}
`,
		},
		{
			desc:        "CustomVerbTests-2",
			UriTemplate: "/**:verb",
			wantUriTemplate: `
{
   "Origin":"/**:verb",
   "Segments":[
      "**"
   ],
   "Variables":null,
   "Verb":"verb"
}
`,
		},
		{
			desc:        "CustomVerbTests-3",
			UriTemplate: "/{a}:verb",
			wantUriTemplate: `
{
   "Origin":"/{a}:verb",
   "Segments":[
      "*"
   ],
   "Variables":[
      {
         "EndSegment":1,
         "FieldPath":[
            "a"
         ],
         "HasDoubleWildCard":false,
         "StartSegment":0
      }
   ],
   "Verb":"verb"
}
`,
		},
		{
			desc:        "CustomVerbTests-4",
			UriTemplate: "/a/b/*:verb",
			wantUriTemplate: `
{
   "Origin":"/a/b/*:verb",
   "Segments":[
      "a",
      "b",
      "*"
   ],
   "Variables":null,
   "Verb":"verb"
}
 `,
		},
		{
			desc:        "CustomVerbTests-5",
			UriTemplate: "/a/b/**:verb",
			wantUriTemplate: `
{
   "Origin":"/a/b/**:verb",
   "Segments":[
      "a",
      "b",
      "**"
   ],
   "Variables":null,
   "Verb":"verb"
}
 `,
		},
		{
			desc:        "CustomVerbTests-6",
			UriTemplate: "/a/b/{a}:verb",
			wantUriTemplate: `
{
   "Origin":"/a/b/{a}:verb",
   "Segments":[
      "a",
      "b",
      "*"
   ],
   "Variables":[
      {
         "EndSegment":3,
         "FieldPath":[
            "a"
         ],
         "HasDoubleWildCard":false,
         "StartSegment":2
      }
   ],
   "Verb":"verb"
}
`,
		},
		{
			desc:        "MoreVariableTests-1",
			UriTemplate: "/{x}",
			wantUriTemplate: `
{
   "Origin":"/{x}",
   "Segments":[
      "*"
   ],
   "Variables":[
      {
         "EndSegment":1,
         "FieldPath":[
            "x"
         ],
         "HasDoubleWildCard":false,
         "StartSegment":0
      }
   ],
   "Verb":""
}
`,
		},
		{
			desc:        "MoreVariableTests-2",
			UriTemplate: "/{x.y.z}",
			wantUriTemplate: `
{
   "Origin":"/{x.y.z}",
   "Segments":[
      "*"
   ],
   "Variables":[
      {
         "EndSegment":1,
         "FieldPath":[
            "x",
            "y",
            "z"
         ],
         "HasDoubleWildCard":false,
         "StartSegment":0
      }
   ],
   "Verb":""
}
`,
		},
		{
			desc:        "MoreVariableTests-3",
			UriTemplate: "/{x=*}",
			wantUriTemplate: `
{
   "Origin":"/{x=*}",
   "Segments":[
      "*"
   ],
   "Variables":[
      {
         "EndSegment":1,
         "FieldPath":[
            "x"
         ],
         "HasDoubleWildCard":false,
         "StartSegment":0
      }
   ],
   "Verb":""
}
`,
		},
		{
			desc:        "MoreVariableTests-4",
			UriTemplate: "/{x=a/*}",
			wantUriTemplate: `
{
   "Origin":"/{x=a/*}",
   "Segments":[
      "a",
      "*"
   ],
   "Variables":[
      {
         "EndSegment":2,
         "FieldPath":[
            "x"
         ],
         "HasDoubleWildCard":false,
         "StartSegment":0
      }
   ],
   "Verb":""
} 
`,
		},
		{
			desc:        "MoreVariableTests-5",
			UriTemplate: "/{x.y.z=*/a/b}/c",
			wantUriTemplate: `
{
   "Origin":"/{x.y.z=*/a/b}/c",
   "Segments":[
      "*",
      "a",
      "b",
      "c"
   ],
   "Variables":[
      {
         "EndSegment":3,
         "FieldPath":[
            "x",
            "y",
            "z"
         ],
         "HasDoubleWildCard":false,
         "StartSegment":0
      }
   ],
   "Verb":""
}
`,
		},
		{
			desc:        "MoreVariableTests-6",
			UriTemplate: "/{x=**}",
			wantUriTemplate: `
{
   "Origin":"/{x=**}",
   "Segments":[
      "**"
   ],
   "Variables":[
      {
         "EndSegment":-1,
         "FieldPath":[
            "x"
         ],
         "HasDoubleWildCard":true,
         "StartSegment":0
      }
   ],
   "Verb":""
}
`,
		},
		{
			desc:        "MoreVariableTests-7",
			UriTemplate: "/{x.y.z=**}",
			wantUriTemplate: `
{
   "Origin":"/{x.y.z=**}",
   "Segments":[
      "**"
   ],
   "Variables":[
      {
         "EndSegment":-1,
         "FieldPath":[
            "x",
            "y",
            "z"
         ],
         "HasDoubleWildCard":true,
         "StartSegment":0
      }
   ],
   "Verb":""
} 
`,
		},
		{
			desc:        "MoreVariableTests-8",
			UriTemplate: "/{x.y.z=a/**/b}",
			wantUriTemplate: `
{
   "Origin":"/{x.y.z=a/**/b}",
   "Segments":[
      "a",
      "**",
      "b"
   ],
   "Variables":[
      {
         "EndSegment":-1,
         "FieldPath":[
            "x",
            "y",
            "z"
         ],
         "HasDoubleWildCard":true,
         "StartSegment":0
      }
   ],
   "Verb":""
}
 `,
		},
		{
			desc:        "MoreVariableTests-9",
			UriTemplate: "/{x.y.z=a/**/b}/c/d",
			wantUriTemplate: `
{
   "Origin":"/{x.y.z=a/**/b}/c/d",
   "Segments":[
      "a",
      "**",
      "b",
      "c",
      "d"
   ],
   "Variables":[
      {
         "EndSegment":-3,
         "FieldPath":[
            "x",
            "y",
            "z"
         ],
         "HasDoubleWildCard":true,
         "StartSegment":0
      }
   ],
   "Verb":""
}
`,
		},
		{
			desc:        "VariableAndCustomVerbTests-1",
			UriTemplate: "/{x}:verb",
			wantUriTemplate: `
{
   "Origin":"/{x}:verb",
   "Segments":[
      "*"
   ],
   "Variables":[
      {
         "EndSegment":1,
         "FieldPath":[
            "x"
         ],
         "HasDoubleWildCard":false,
         "StartSegment":0
      }
   ],
   "Verb":"verb"
}
`,
		},
		{
			desc:        "VariableAndCustomVerbTests-2",
			UriTemplate: "/{x.y.z}:verb",
			wantUriTemplate: `
{
   "Origin":"/{x.y.z}:verb",
   "Segments":[
      "*"
   ],
   "Variables":[
      {
         "EndSegment":1,
         "FieldPath":[
            "x",
            "y",
            "z"
         ],
         "HasDoubleWildCard":false,
         "StartSegment":0
      }
   ],
   "Verb":"verb"
}
 `,
		},
		{
			desc:        "VariableAndCustomVerbTests-3",
			UriTemplate: "/{x.y.z=*/*}:verb",
			wantUriTemplate: `
{
   "Origin":"/{x.y.z=*/*}:verb",
   "Segments":[
      "*",
      "*"
   ],
   "Variables":[
      {
         "EndSegment":2,
         "FieldPath":[
            "x",
            "y",
            "z"
         ],
         "HasDoubleWildCard":false,
         "StartSegment":0
      }
   ],
   "Verb":"verb"
}
 `,
		},
		{
			desc:        "VariableAndCustomVerbTests-4",
			UriTemplate: "/{x=**}:myverb",
			wantUriTemplate: `
{
   "Origin":"/{x=**}:myverb",
   "Segments":[
      "**"
   ],
   "Variables":[
      {
         "EndSegment":-2,
         "FieldPath":[
            "x"
         ],
         "HasDoubleWildCard":true,
         "StartSegment":0
      }
   ],
   "Verb":"myverb"
}
`,
		},
		{
			desc:        "VariableAndCustomVerbTests-5",
			UriTemplate: "/{x.y.z=**}:myverb",
			wantUriTemplate: `
{
   "Origin":"/{x.y.z=**}:myverb",
   "Segments":[
      "**"
   ],
   "Variables":[
      {
         "EndSegment":-2,
         "FieldPath":[
            "x",
            "y",
            "z"
         ],
         "HasDoubleWildCard":true,
         "StartSegment":0
      }
   ],
   "Verb":"myverb"
}
`,
		},
		{
			desc:        "VariableAndCustomVerbTests-6",
			UriTemplate: "/{x.y.z=a/**/b}:custom",
			wantUriTemplate: `
{
   "Origin":"/{x.y.z=a/**/b}:custom",
   "Segments":[
      "a",
      "**",
      "b"
   ],
   "Variables":[
      {
         "EndSegment":-2,
         "FieldPath":[
            "x",
            "y",
            "z"
         ],
         "HasDoubleWildCard":true,
         "StartSegment":0
      }
   ],
   "Verb":"custom"
}
`,
		},
		{
			desc:        "VariableAndCustomVerbTests-7",
			UriTemplate: "/{x.y.z=a/**/b}/c/d:custom",
			wantUriTemplate: `
{
   "Origin":"/{x.y.z=a/**/b}/c/d:custom",
   "Segments":[
      "a",
      "**",
      "b",
      "c",
      "d"
   ],
   "Variables":[
      {
         "EndSegment":-4,
         "FieldPath":[
            "x",
            "y",
            "z"
         ],
         "HasDoubleWildCard":true,
         "StartSegment":0
      }
   ],
   "Verb":"custom"
}
`,
		},
		{
			desc:        "RootPath",
			UriTemplate: "/",
			wantUriTemplate: `
{
   "Origin":"/",
   "Segments":null,
   "Variables":null,
   "Verb":""
}
`,
		},
		{
			desc:        "ParseVerbTest2",
			UriTemplate: "/a/*:verb",
			wantUriTemplate: `
{
   "Origin":"/a/*:verb",
   "Segments":[
      "a",
      "*"
   ],
   "Variables":null,
   "Verb":"verb"
}
`,
		},
		{
			desc:        "ParseVerbTest3",
			UriTemplate: "/a/**:verb",
			wantUriTemplate: `
{
   "Origin":"/a/**:verb",
   "Segments":[
      "a",
      "**"
   ],
   "Variables":null,
   "Verb":"verb"
}
`,
		},
		{
			desc:        "ParseVerbTest4",
			UriTemplate: "/a/{b=*}/**:verb",
			wantUriTemplate: `
{
   "Origin":"/a/{b=*}/**:verb",
   "Segments":[
      "a",
      "*",
      "**"
   ],
   "Variables":[
      {
         "EndSegment":2,
         "FieldPath":[
            "b"
         ],
         "HasDoubleWildCard":false,
         "StartSegment":1
      }
   ],
   "Verb":"verb"
}
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ut := ParseUriTemplate(tc.UriTemplate)
			if ut == nil {
				t.Fatal("fail to generate UriTemplate")
			}

			if tc.wantUriTemplate != "" {
				bytes, _ := json.Marshal(ut)
				getUriTemplate := string(bytes)

				if err := util.JsonEqual(tc.wantUriTemplate, getUriTemplate); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func TestUriTemplateParseError(t *testing.T) {
	testCases := []string{
		"",
		"//",
		"/{}",
		"/a/",
		"/a//b",
		":verb",
		"/:verb",
		"/a/:verb",
		":",
		"/:",
		"/*:",
		"/**:",
		"/{{",
		"/{var}:",
		"/{var}::",
		"/{var/a",
		"/{{var}}",
		"/a/b/:",
		"/a/b/*:",
		"/a/b/**:",
		"/a/b/{var}:",
		"/a/{",
		"/a/{var",
		"/a/{var.",
		"/a/{x=var:verb}",
		"a",
		"{x}",
		"{x=/a}",
		"{x=/a/b}",
		"a/b",
		"a/b/{x}",
		"a/{x}/b",
		"a/{x}/b:verb",
		"/a/{var=/b}",
		"/{var=a/{nested=b}}",
		"/a{x}",
		"/{x}a",
		"/a{x}b",
		"/{x}a{y}",
		"/a/b{x}",
		"/a/{x}b",
		"/a/b{x}c",
		"/a/{x}b{y}",
		"/a/b{x}/s",
		"/a/{x}b/s",
		"/a/b{x}c/s",
		"/a/{x}b{y}/s",
		"/a/**/*",
		":",
		"/:",
		"/a/:",
		"/a/*:",
		"/a/**:",
		"/a/{b=*}/**:",
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("`%s`", tc), func(t *testing.T) {
			if ParseUriTemplate(tc) != nil {
				t.Fatalf("succeed parsing %s but expect to fail", tc)
			}
		})

	}
}
