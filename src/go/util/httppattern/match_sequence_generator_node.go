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
	"sort"
)

const HttpMethodWildCard = "*"

// Trie node to implement sequence generator.
type sorterNode struct {
	ResultMap map[string]*lookupResult
	Children  map[string]*sorterNode
	WildCard  bool
}

func newMatchSequenceGeneratorNode() *sorterNode {
	return &sorterNode{
		ResultMap: make(map[string]*lookupResult),
		Children:  make(map[string]*sorterNode),
	}
}

func (hn *sorterNode) insertTemplate(pathParts []string, pathPartsIdxCur int, httpMethod string, methodData *methodData, markDuplicate bool) bool {
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
		hn.Children[curSeg] = newMatchSequenceGeneratorNode()
	}

	child, _ := hn.Children[curSeg]
	if curSeg == WildCardPathKey {
		child.WildCard = true
	}

	return child.insertTemplate(pathParts, pathPartsIdxCur+1, httpMethod, methodData, markDuplicate)
}

func (hn *sorterNode) insertPath(pathParts []string, httpMethod string, methodData *methodData, markDuplicate bool) bool {
	return hn.insertTemplate(pathParts, 0, httpMethod, methodData, markDuplicate)
}

func (hn *sorterNode) lookupPath(pathParts []string, pathPartsIdxCur int, httpMethod string, result *lookupResult) {
	GetResultForHttpMethod := func(resultMap map[string]*lookupResult, m string, result *lookupResult) bool {
		if val, ok := resultMap[m]; ok {
			*result = *val
			return true
		}
		if val, ok := resultMap[HttpMethodWildCard]; ok {
			*result = *val
			return true
		}
		return false
	}

	lookupPathFromChild := func(childKey string, pathParts []string, pathPartsIdxCur int, httpMethod string, result *lookupResult) bool {
		if child, ok := hn.Children[childKey]; ok {
			child.lookupPath(pathParts, pathPartsIdxCur+1, httpMethod, result)
			if result != nil && result.data != nil {
				return true
			}
		}
		return false
	}

	if pathPartsIdxCur == len(pathParts) {
		if !GetResultForHttpMethod(hn.ResultMap, httpMethod, result) {
			// If we didn't find a wrapper graph at this node, check if we have one
			// in a wildcard (**) child. If we do, use it. This will ensure we match
			// the root with wildcard templates.
			if child, ok := hn.Children[WildCardPathKey]; ok {
				GetResultForHttpMethod(child.ResultMap, httpMethod, result)
			}
		}
		return
	}

	if lookupPathFromChild(pathParts[pathPartsIdxCur], pathParts, pathPartsIdxCur, httpMethod, result) {
		return
	}

	// For wild card node, keeps searching for next path segment until either
	// 1) reaching the end (/foo/** case), or 2) all remaining segments match
	// one of child branches (/foo/**/bar/xyz case).
	if hn.WildCard {
		hn.lookupPath(pathParts, pathPartsIdxCur+1, httpMethod, result)
		// Since only constant segments are allowed after wild card, no need to
		// search another wild card nodes from children, so bail out here.
		return
	}

	for _, childKey := range []string{SingleParameterKey, WildCardPathPartKey, WildCardPathKey} {
		if lookupPathFromChild(childKey, pathParts, pathPartsIdxCur, httpMethod, result) {
			return
		}
	}
}

// Traverse the sorter trie in matching order and add the visited method in result.
func (hn *sorterNode) traverse(result *MatchSequence) {
	appendMethodOnCurrentNode := func() {
		// Put the wildcard method in the end.
		var wildMethodResult *lookupResult
		if val, ok := hn.ResultMap[HttpMethodWildCard]; ok {
			wildMethodResult = val
			delete(hn.ResultMap, HttpMethodWildCard)
		}

		// Sort the method in alphabet order to generate deterministic sequence for better unit testing.
		var sortedKeys []string
		for key, _ := range hn.ResultMap {
			sortedKeys = append(sortedKeys, key)
		}
		sort.Strings(sortedKeys)

		for _, key := range sortedKeys {
			if val, ok := hn.ResultMap[key]; ok {
				result.appendMethod(val.data.Method)
			}
		}

		if wildMethodResult != nil {
			result.appendMethod(wildMethodResult.data.Method)
		}

	}

	traverseChildren := func() {
		var singleParameterChild *sorterNode
		var wildCardPathPartChild *sorterNode
		var wildCardPathChild *sorterNode
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
		for _, child := range []*sorterNode{singleParameterChild, wildCardPathPartChild, wildCardPathChild} {
			if child != nil {
				child.traverse(result)
			}
		}
	}

	// If the current node is wildcard(**), its children has higher priority.
	if hn.WildCard {
		traverseChildren()
		appendMethodOnCurrentNode()
	} else {
		appendMethodOnCurrentNode()
		traverseChildren()
	}
}
