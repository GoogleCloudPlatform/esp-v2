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
	"sort"
)

const HttpMethodWildCard = "*"

// httpPatternTrie store the methods based on the http patterns.
// The implementation is based on
// https://github.com/GoogleCloudPlatform/esp-v2/blob/641ce1d5c177401e424f2b27dd45de1bf797530b/src/api_proxy/path_matcher/path_matcher.h#L1
type httpPatternTrie struct {
	RootPtr     *httpPatternTrieNode
	CustomVerbs map[string]bool
}

type httpPatternTrieNode struct {
	ResultMap map[string]*lookupResult
	Children  map[string]*httpPatternTrieNode
	WildCard  bool
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

func newHttpPatternTrie() *httpPatternTrie {
	return &httpPatternTrie{
		RootPtr:     newHttpPatternTrieNode(),
		CustomVerbs: make(map[string]bool),
	}
}

func (h *httpPatternTrie) register(method *Method) error {
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

	if ht.Verb != "" {
		h.CustomVerbs[ht.Verb] = true
	}

	return nil
}

func (h *httpPatternTrie) sort() *MethodSlice {
	result := &MethodSlice{}
	h.RootPtr.traverse(result)
	return result

}

func newHttpPatternTrieNode() *httpPatternTrieNode {
	return &httpPatternTrieNode{
		ResultMap: make(map[string]*lookupResult),
		Children:  make(map[string]*httpPatternTrieNode),
	}
}

func (sr *MethodSlice) appendMethod(m *Method) {
	if m != nil {
		*sr = append(*sr, m)
	}
}

func extractBindingsFromPath(vars []*variable, parts []string) []*variableBinding {
	var bindings []*variableBinding
	for _, v := range vars {
		// Determine the subpath bound to the variable based on the
		// [start_segment, end_segment) segment range of the variable.
		//
		// In case of matching "**" - end_segment is negative and is relative to
		// the end such that end_segment = -1 will match all subsequent segmenth.
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

func (hn *httpPatternTrieNode) insertTemplate(pathParts []string, pathPartsIdxCur int, httpMethod string, methodData *methodData, markDuplicate bool) bool {
	if pathPartsIdxCur == len(pathParts) {
		if val, ok := hn.ResultMap[httpMethod]; ok {
			if markDuplicate {
				val.isMultiple = true
			}
			return false
		}
		hn.ResultMap[httpMethod] = &lookupResult{
			data:       methodData,
			isMultiple: false,
		}
		return true
	}

	curSeg := pathParts[pathPartsIdxCur]

	if _, ok := hn.Children[curSeg]; !ok {
		hn.Children[curSeg] = newHttpPatternTrieNode()
	}

	child, _ := hn.Children[curSeg]
	if curSeg == WildCardPathKey {
		child.WildCard = true
	}

	return child.insertTemplate(pathParts, pathPartsIdxCur+1, httpMethod, methodData, markDuplicate)
}

func (hn *httpPatternTrieNode) insertPath(pathParts []string, httpMethod string, methodData *methodData, markDuplicate bool) bool {
	return hn.insertTemplate(pathParts, 0, httpMethod, methodData, markDuplicate)
}

// Traverse the httpPatternTrie in matching order and add the visited method in result.
func (hn *httpPatternTrieNode) traverse(result *MethodSlice) {
	appendMethodOnCurrentNode := func() {

		// Sort the method in alphabet order to generate deterministic sequence for better unit testing.
		var sortedKeys []string
		// Put the wildcard method in the end.
		var wildMethodResult *lookupResult
		for key, val := range hn.ResultMap {
			if key == HttpMethodWildCard {
				wildMethodResult = val
				continue
			}
			sortedKeys = append(sortedKeys, key)
		}
		sort.Strings(sortedKeys)

		for _, key := range sortedKeys {
			if val, ok := hn.ResultMap[key]; ok {
				result.appendMethod(val.data.Method)
			}
		}

		// Put the wildcard method in the end.
		if wildMethodResult != nil {
			result.appendMethod(wildMethodResult.data.Method)
		}

	}

	traverseChildren := func() {
		var singleParameterChild *httpPatternTrieNode
		var wildCardPathPartChild *httpPatternTrieNode
		var wildCardPathChild *httpPatternTrieNode
		var exactMatchChildKey []string
		for key, child := range hn.Children {
			switch key {
			case SingleParameterKey:
				singleParameterChild = child
			case WildCardPathPartKey:
				wildCardPathPartChild = child
			case WildCardPathKey:
				wildCardPathChild = child
			default:
				exactMatchChildKey = append(exactMatchChildKey, key)
			}
		}

		// Visit exact match children first.
		// Sort the child keys to generate deterministic sequence for better unit testing.
		sort.Strings(exactMatchChildKey)
		for _, key := range exactMatchChildKey {
			if child, ok := hn.Children[key]; ok {
				child.traverse(result)
			}
		}

		// Visit vague match children after.
		for _, child := range []*httpPatternTrieNode{singleParameterChild, wildCardPathPartChild, wildCardPathChild} {
			if child != nil {
				child.traverse(result)
			}
		}
	}

	// If the current node is wildcard(**), its children has higher priority.
	if hn.WildCard {
		// Pre-order traverse.
		traverseChildren()
		// Post-order traverse.
		appendMethodOnCurrentNode()
	} else {
		appendMethodOnCurrentNode()
		traverseChildren()
	}
}
