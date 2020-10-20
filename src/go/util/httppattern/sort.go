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

type Method struct {
	UriTemplate string
	HttpMethod  string
	Operation   string
}

type MethodSlice []*Method

// Sort the slice of methods, based on the http patterns.
// It will raise errors:
//   - methods with duplicate http pattern
//   - invalid uri template
// The time complexity is O(W * L), where W is the size of slice
// and L is the size of uri template segments
func Sort(methods *MethodSlice) error {
	s := newHttpPatternTrie()
	for _, m := range *methods {
		if err := s.register(m); err != nil {
			return err
		}
	}

	*methods = nil
	s.RootPtr.traverse(methods)

	return nil
}
