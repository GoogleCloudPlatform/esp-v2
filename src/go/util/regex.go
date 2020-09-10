// Copyright 2020 Google LLC
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

package util

import (
	"fmt"
	"regexp/syntax"
)

func ValidateRegexProgramSize(regex string, maxProgramSize int) error {
	regParse, err := syntax.Parse(regex, 0)
	if err != nil {
		return err
	}

	prog, err := syntax.Compile(regParse)
	if err != nil {
		return err
	}

	if len(prog.Inst) > maxProgramSize {
		return fmt.Errorf("regex program size(%v) is larger than the max expected(%v): %s", len(prog.Inst), maxProgramSize, regex)
	}

	return nil
}
