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
	"bytes"
	"fmt"

	"github.com/google/go-cmp/cmp"
)

// Use null char to denote coming into invalid char.
const (
	InvalidChar = byte(0)
)

// Pattern Corresponds espv2.api.envoy.v10.http.common.Pattern and it holds the
// syntax parsing result for uri template.
type Pattern struct {
	HttpMethod string
	*UriTemplate
}

// UriTemplate keeps information of the uri template string.
// It follows the grammar:
// https://github.com/googleapis/googleapis/blob/e5211c547d63632963f9125e2b333185d57ff8f6/google/api/http.proto#L224.
type UriTemplate struct {
	Segments  []string
	Verb      string
	Variables []*variable
	// The original uri template string before parsing.
	// It is ignored when calling `String` or `Equal`
	Origin string
}

// The info about a variable binding {variable=subpath} in the template.
type variable struct {
	// Specifies the range of segments [start_segment, end_segment) the
	// variable binds to. Both start_segment and end_segment are 0 based.
	// end_segment can also be negative, which means that the position is
	// specified relative to the end such that -1 corresponds to the end
	// of the path.
	StartSegment int
	EndSegment   int

	// The path of the protobuf field the variable binds to.
	FieldPath []string

	// Do we have a ** in the variable template?
	HasDoubleWildCard bool
}

func (u *UriTemplate) ExactMatchString(acceptTrailingBackslash bool) string {
	if len(u.Segments) == 0 {
		return "/"
	}

	startSegmentToVariable := make(map[int]*variable)
	for _, v := range u.Variables {
		startSegmentToVariable[v.StartSegment] = v

		// The opposite processing for EndSegment against `postProcessVariables()`
		// Recover EndSegment from negative index for positive index for doubleWildCard
		if v.EndSegment < 0 && v.HasDoubleWildCard {
			if u.Verb != "" {
				v.EndSegment += 1
			}
			v.EndSegment = v.EndSegment + len(u.Segments) + 1
		}
	}

	buff := bytes.Buffer{}
	nextIdx := 0
	for idx, seg := range u.Segments {
		//  The current segment has been visited included in variable.
		if idx < nextIdx {
			continue
		}
		nextIdx = idx + 1

		// Add variable syntax.
		if v, ok := startSegmentToVariable[idx]; ok {
			buff.WriteString(generateVariableBindingSyntax(u.Segments, v))
			nextIdx = v.EndSegment
			continue
		}

		// Add path field.
		buff.WriteString(fmt.Sprintf("/%s", seg))
	}

	if acceptTrailingBackslash {
		buff.WriteString("/")
	}

	if u.Verb != "" {
		buff.WriteString(fmt.Sprintf(":%s", u.Verb))
	}

	return buff.String()
}

// Output the string representation with defaults.
func (u *UriTemplate) String() string {
	return u.ExactMatchString(false)
}

// Check if two uriTemplates are equal. Ignore `Origin`
func (u *UriTemplate) Equal(v *UriTemplate) bool {
	return cmp.Equal(u.Segments, v.Segments) && cmp.Equal(u.Variables, v.Variables) && cmp.Equal(u.Verb, v.Verb)
}

// Replace all the variable fields found in the input map.
func (u *UriTemplate) ReplaceVariableField(fieldMapping map[string]string) {
	for _, v := range u.Variables {
		var newFieldPath []string

		for _, field := range v.FieldPath {
			if newField, ok := fieldMapping[field]; ok {
				newFieldPath = append(newFieldPath, newField)
			} else {
				newFieldPath = append(newFieldPath, field)
			}
		}
		v.FieldPath = newFieldPath
	}
}

func (u *UriTemplate) IsExactMatch() bool {
	for _, seg := range u.Segments {
		if seg == SingleWildCardKey || seg == DoubleWildCardKey {
			return false
		}
	}
	return true
}

// Generate regular expression of the current uri template.
func (u *UriTemplate) Regex(ExcludeColonInUrlWildcardPathSegment bool) string {
	regex := bytes.Buffer{}
	for _, segment := range u.Segments {
		regex.WriteByte('/')
		switch segment {
		case SingleWildCardKey:
			regex.WriteString(singleWildcardReplacementRegex(ExcludeColonInUrlWildcardPathSegment))
		case DoubleWildCardKey:
			regex.WriteString(doubleWildcardReplacementRegex(ExcludeColonInUrlWildcardPathSegment))
		default:
			regex.WriteString(segment)
		}
	}
	regex.WriteString(optionalTrailingSlashRegex)

	if u.Verb != "" {
		regex.WriteString(":" + u.Verb)
	}

	return "^" + regex.String() + "$"
}

// `generateVariableBindingSyntax` tries to recover the following syntax with
// replacement of fieldPathName.
//    Variable = "{" FieldPath [ "=" Segments ] "}" ;
func generateVariableBindingSyntax(segments []string, v *variable) string {
	pathVar := bytes.Buffer{}
	for i := v.StartSegment; i < v.EndSegment; i += 1 {
		pathVar.WriteString(segments[i])
		if i != v.EndSegment-1 {
			pathVar.WriteString("/")
		}
	}

	varName := bytes.Buffer{}
	for idx, field := range v.FieldPath {
		varName.WriteString(field)
		if idx != len(v.FieldPath)-1 {
			varName.WriteByte('.')
		}
	}

	return fmt.Sprintf("/{%s=%s}", varName.String(), pathVar.String())
}
