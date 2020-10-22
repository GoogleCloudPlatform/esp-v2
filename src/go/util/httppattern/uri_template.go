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
)

// Use null char to denote coming into invalid char.
const (
	InvalidChar = byte(0)
)

// UriTemplate is used to syntax pairse uri templates. It is based on the grammar
// on https://github.com/googleapis/googleapis/blob/e5211c547d63632963f9125e2b333185d57ff8f6/google/api/http.proto#L224.
type UriTemplate struct {
	Segments  []string
	Verb      string
	Variables []*variable
}

func Parse(input string) *UriTemplate {
	if input == "/" {
		return &UriTemplate{}
	}

	p := parser{
		input: input,
	}
	if !p.parse() || !p.validateParts() {
		return nil
	}

	return &UriTemplate{
		Segments:  p.segments,
		Verb:      p.verb,
		Variables: p.variables,
	}
}

func ReplaceVariableFieldName(input string, fieldNameMapping map[string]string) (string, error) {
	uriTemplate := Parse(input)
	if uriTemplate == nil {
		return "", fmt.Errorf("invalid uri template `%s`", input)
	}

	return serializeUriTemplate(uriTemplate, fieldNameMapping), nil
}
