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

// Trie node to implement match uence generator.
type matchSequenceGeneratorNode struct {
	ResultMap map[string]*lookupResult
	Children  map[string]*matchSequenceGeneratorNode
	WildCard  bool
}

func newMatchSequenceGeneratorNode() *matchSequenceGeneratorNode {
	return &matchSequenceGeneratorNode{
		ResultMap: make(map[string]*lookupResult),
		Children:  make(map[string]*matchSequenceGeneratorNode),
	}
}

func (mn *matchSequenceGeneratorNode) insertTemplate(pathParts []string, pathPartsIdxCur int, httpMethod string, methodData *methodData, markDuplicate bool) bool {
	if pathPartsIdxCur == len(pathParts) {
		if val, ok := mn.ResultMap[httpMethod]; ok {
			if markDuplicate {
				val.isMultiple = true
			}
			return false
		}
		mn.ResultMap[httpMethod] = &lookupResult{
			data:       methodData,
			isMultiple: false,
		}
		return true
	}

	curSeg := pathParts[pathPartsIdxCur]

	if _, ok := mn.Children[curSeg]; !ok {
		mn.Children[curSeg] = newMatchSequenceGeneratorNode()
	}

	child, _ := mn.Children[curSeg]
	if curSeg == WildCardPathKey {
		child.WildCard = true
	}

	return child.insertTemplate(pathParts, pathPartsIdxCur+1, httpMethod, methodData, markDuplicate)
}

func (mn *matchSequenceGeneratorNode) insertPath(pathParts []string, httpMethod string, methodData *methodData, markDuplicate bool) bool {
	return mn.insertTemplate(pathParts, 0, httpMethod, methodData, markDuplicate)
}

// Traverse the sorter trie in matching order and add the visited method in result.
func (mn *matchSequenceGeneratorNode) traverse(result *MatchSequence) {
	appendMethodOnCurrentNode := func() {

		// Sort the method in alphabet order to generate deterministic sequence for better unit testing.
		var sortedKeys []string
		// Put the wildcard method in the end.
		var wildMethodResult *lookupResult
		for key, val := range mn.ResultMap {
			if key == HttpMethodWildCard {
				wildMethodResult = val
				continue
			}
			sortedKeys = append(sortedKeys, key)
		}
		sort.Strings(sortedKeys)

		for _, key := range sortedKeys {
			if val, ok := mn.ResultMap[key]; ok {
				result.appendMethod(val.data.Method)
			}
		}

		// Put the wildcard method in the end.
		if wildMethodResult != nil {
			result.appendMethod(wildMethodResult.data.Method)
		}

	}

	traverseChildren := func() {
		var singleParameterChild *matchSequenceGeneratorNode
		var wildCardPathPartChild *matchSequenceGeneratorNode
		var wildCardPathChild *matchSequenceGeneratorNode
		var exactMatchChildKey []string
		for key, child := range mn.Children {
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
			if child, ok := mn.Children[key]; ok {
				child.traverse(result)
			}
		}

		// Visit vague match children after.
		for _, child := range []*matchSequenceGeneratorNode{singleParameterChild, wildCardPathPartChild, wildCardPathChild} {
			if child != nil {
				child.traverse(result)
			}
		}
	}

	// If the current node is wildcard(**), its children has higher priority.
	if mn.WildCard {
		// Pre-order traverse.
		traverseChildren()
		// Post-order traverse.
		appendMethodOnCurrentNode()
	} else {
		appendMethodOnCurrentNode()
		traverseChildren()
	}
}
