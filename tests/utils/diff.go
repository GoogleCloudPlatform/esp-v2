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

package utils

import (
	"strings"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

// ProtoDiff returns git diff style line-by-line diff between marshalled proto.
// Lines prefixed with '-' are missing in y and lines prefixed with '+' are
// extra in y.
func ProtoDiff(x, y proto.Message) string {
	if proto.Equal(x, y) {
		return ""
	}

	return StringDiff(prototext.Format(x), prototext.Format(y))
}

// StringDiff returns git diff style line-by-line diff between two strings.
// Lines prefixed with '-' are missing in y and lines prefixed with '+' are
// extra in y.
func StringDiff(x, y string) string {
	if x == y {
		return ""
	}

	var xs, ys []string
	if x != "" {
		xs = strings.Split(x, "\n")
	}
	if y != "" {
		ys = strings.Split(y, "\n")
	}

	lcs := longestCommonSuffix(xs, ys)
	buf := strings.Builder{}

	first := true
	var writeDiff = func(buf *strings.Builder, sign rune, str string) {
		if !first {
			buf.WriteRune('\n')
		} else {
			first = false
		}
		buf.WriteRune(sign)
		buf.WriteString(str)
	}

	xi := 0
	yi := 0
	for _, s := range lcs {
		for xi < len(xs) && xs[xi] != s {
			writeDiff(&buf, '-', xs[xi])
			xi++
		}
		xi++

		for yi < len(ys) && ys[yi] != s {
			writeDiff(&buf, '+', ys[yi])
			yi++
		}
		yi++
	}

	for xi < len(xs) {
		writeDiff(&buf, '-', xs[xi])
		xi++
	}

	for yi < len(ys) {
		writeDiff(&buf, '+', ys[yi])
		yi++
	}

	return buf.String()
}

func longestCommonSuffix(x, y []string) []string {
	type key struct {
		x, y int
	}
	m := make(map[key][]string)

	var longestCommonSuffixFn func(x, y []string) []string
	longestCommonSuffixFn = func(x, y []string) []string {
		k := key{len(x), len(y)}
		if v := m[k]; v != nil {
			return v
		}

		if len(x) == 0 || len(y) == 0 {
			return nil
		}

		xLast := x[len(x)-1]
		yLast := y[len(y)-1]
		if xLast == yLast {
			lcs := longestCommonSuffixFn(x[:len(x)-1], y[:len(y)-1])
			lcsLast := append(lcs, xLast)
			m[k] = lcsLast
			return lcsLast
		}

		lcsX := longestCommonSuffixFn(x, y[:len(y)-1])
		lcsY := longestCommonSuffixFn(x[:len(x)-1], y)
		if len(lcsX) >= len(lcsY) {
			m[k] = lcsX
			return lcsX
		}
		m[k] = lcsY
		return lcsY
	}

	return longestCommonSuffixFn(x, y)
}
