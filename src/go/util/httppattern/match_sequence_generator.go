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
)

// MatchSequenceGenerator generate the match sequence of added methods based on their
// http pattern. The sequence should be same as the matching order of
// https://github.com/GoogleCloudPlatform/esp-v2/blob/641ce1d5c177401e424f2b27dd45de1bf797530b/src/api_proxy/path_matcher/path_matcher.h#L1
type MatchSequenceGenerator struct {
	RootPtr     *sorterNode
	CustomVerbs map[string]bool
	Methods     []*methodData
}

type Method struct {
	UriTemplate string
	HttpMethod  string
	Operation   string
}

func NewMatchSequenceGenerator() *MatchSequenceGenerator {
	return &MatchSequenceGenerator{
		RootPtr:     newMatchSequenceGeneratorNode(),
		CustomVerbs: make(map[string]bool),
	}
}

func (h *MatchSequenceGenerator) Register(method *Method) error {
	if method == nil {
		return fmt.Errorf("empty method")
	}

	uriTemplate := method.UriTemplate
	httpMethod := method.HttpMethod

	ht := Parse(uriTemplate)
	if ht == nil {
		return fmt.Errorf("invalid url template `%s`", uriTemplate)
	}

	pathInfo := transferFromUriTemplate(ht)
	methodData := &methodData{
		Method:   method,
		Variable: ht.Variables,
	}

	if !h.RootPtr.insertPath(pathInfo, httpMethod, methodData, true) {
		return fmt.Errorf("duplicate http pattern `%s %s`", httpMethod, uriTemplate)
	}
	h.Methods = append(h.Methods, methodData)
	if ht.Verb != "" {
		h.CustomVerbs[ht.Verb] = true
	}

	return nil
}

type MatchSequence []*Method

// Return a sorted slice of methods, used to match incoming request in sequence.
func (h *MatchSequenceGenerator) Generate() *MatchSequence {
	result := &MatchSequence{}
	h.RootPtr.traverse(result)
	return result

}

type variableBinding struct {
	FieldPath []string
	Value     string
}

type methodData struct {
	*Method
	Variable []*variable
}

type lookupResult struct {
	data       *methodData
	isMultiple bool
}

func (sr *MatchSequence) appendMethod(m *Method) {
	if m != nil {
		*sr = append(*sr, m)
	}
}

// This method is used for test only so far.
func (h *MatchSequenceGenerator) lookup(httpMethod string, path string) (*Method, []*variableBinding) {
	parts := extractRequestParts(path, h.CustomVerbs)

	if h.RootPtr == nil {
		return nil, nil
	}

	result := &lookupResult{}
	h.RootPtr.lookupPath(parts, 0, httpMethod, result)
	if result.data == nil {
		return nil, nil
	}

	return result.data.Method, extractBindingsFromPath(result.data.Variable, parts)
}

func extractRequestParts(path string, customVerbs map[string]bool) []string {
	// Remove query parameters.
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[0:idx]
	}
	// Replace last ':' with '/' to handle custom verb.
	// But not for /foo:bar/const.
	lastColonPos := strings.LastIndex(path, ":")
	lastSlashPos := strings.LastIndex(path, "/")

	if lastColonPos != -1 && lastColonPos > lastSlashPos {
		verb := path[lastColonPos+1:]

		if _, ok := customVerbs[verb]; ok {
			path = path[0:lastColonPos] + "/" + path[lastColonPos+1:]
		}
	}
	var result []string
	if path != "" {
		result = strings.Split(path[1:], "/")
	}

	// Removes all trailing empty parts caused by extra "/".
	for len(result) > 0 && result[len(result)-1] == "" {
		result = result[:len(result)-1]
	}

	return result
}

func extractBindingsFromPath(vars []*variable, parts []string) []*variableBinding {
	var bindings []*variableBinding
	for _, v := range vars {
		// Determine the subpath bound to the variable based on the
		// [start_segment, end_segment) segment range of the variable.
		//
		// In case of matching "**" - end_segment is negative and is relative to
		// the end such that end_segment = -1 will match all subsequent segments.
		binding := &variableBinding{
			FieldPath: v.FieldPath,
		}

		// Calculate the absolute index of the ending segment in case it's negative.
		endSegment := v.EndSegment
		if v.EndSegment < 0 {
			endSegment = len(parts) + v.EndSegment + 1
		}
		for i := v.StartSegment; i < endSegment; i += 1 {
			binding.Value = binding.Value + parts[i]
			if i < endSegment-1 {
				binding.Value = binding.Value + "/"
			}
		}
		bindings = append(bindings, binding)

	}

	return bindings
}

func transferFromUriTemplate(ht *UriTemplate) []string {
	var pathParts []string
	for _, segment := range ht.Segments {
		pathParts = append(pathParts, segment)
	}

	if ht.Verb != "" {
		pathParts = append(pathParts, ht.Verb)
	}

	return pathParts
}
