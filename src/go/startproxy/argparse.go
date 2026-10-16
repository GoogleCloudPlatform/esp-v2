// Copyright 2026 Google LLC
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

package startproxy

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// errHelpRequested is returned when -h or --help is passed to parseArgs.
var errHelpRequested = errors.New("help requested")

func resolveOption(defs []*flagDef, name string) (*flagDef, error) {
	if strings.HasPrefix(name, "--") {
		for _, d := range defs {
			for _, l := range d.longs {
				if l == name {
					return d, nil
				}
			}
		}
		// Unambiguous prefix abbreviation matching (argparse allow_abbrev=True).
		var matched []*flagDef
		for _, d := range defs {
			for _, l := range d.longs {
				if strings.HasPrefix(l, name) {
					matched = append(matched, d)
					break
				}
			}
		}
		if len(matched) == 1 {
			return matched[0], nil
		}
		if len(matched) > 1 {
			return nil, fmt.Errorf("ambiguous option: %s", name)
		}
		return nil, fmt.Errorf("unrecognized arguments: %s", name)
	}
	if strings.HasPrefix(name, "-") && len(name) >= 2 {
		for _, d := range defs {
			if d.short == name {
				return d, nil
			}
		}
		return nil, fmt.Errorf("unrecognized arguments: %s", name)
	}
	return nil, fmt.Errorf("unrecognized arguments: %s", name)
}

func isOptionToken(defs []*flagDef, tok string) bool {
	if !strings.HasPrefix(tok, "-") || tok == "-" {
		return false
	}
	optName := tok
	if idx := strings.IndexByte(tok, '='); idx != -1 {
		optName = tok[:idx]
	}
	if d, err := resolveOption(defs, optName); err == nil && d != nil {
		return true
	}
	// Short option with attached argument (e.g. -sfoo).
	if !strings.HasPrefix(tok, "--") && len(tok) > 2 {
		shortPrefix := tok[:2]
		for _, d := range defs {
			if d.short == shortPrefix {
				return true
			}
		}
	}
	return false
}

// printUsage writes the help message for startproxy to w.
func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: startproxy [-h] [options...]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "options:")
	fmt.Fprintln(w, "  -h, --help                               show this help message and exit")
	defs := buildFlagDefs()
	for _, d := range defs {
		var flagNames []string
		if d.short != "" {
			flagNames = append(flagNames, d.short)
		}
		flagNames = append(flagNames, d.longs...)
		flagsStr := strings.Join(flagNames, ", ")
		if len(d.choices) > 0 {
			flagsStr += fmt.Sprintf(" {%s}", strings.Join(d.choices, ","))
		}
		fmt.Fprintf(w, "  %-40s\n", flagsStr)
	}
}

// parseArgs parses command-line flags using argparse-compatible rules.
func parseArgs(argv []string) (*proxyArgs, error) {
	args := defaultArgs()
	defs := buildFlagDefs()

	for i := 0; i < len(argv); i++ {
		rawTok := argv[i]
		tok := rawTok
		if trimmed := strings.TrimSpace(rawTok); strings.HasPrefix(trimmed, "-") {
			tok = trimmed
		}
		if tok == "-h" || tok == "--help" {
			return nil, errHelpRequested
		}
		if !strings.HasPrefix(tok, "-") || tok == "-" {
			return nil, fmt.Errorf("unrecognized arguments: %s", rawTok)
		}

		var optName string
		var inlineVal string
		hasInline := false
		if idx := strings.IndexByte(tok, '='); idx != -1 {
			optName = tok[:idx]
			inlineVal = tok[idx+1:]
			hasInline = true
		} else if !strings.HasPrefix(tok, "--") && len(tok) > 2 {
			// Short flag with attached value, e.g. -sbar.
			shortCandidate := tok[:2]
			if d, err := resolveOption(defs, shortCandidate); err == nil && d != nil && d.kind != kindStoreTrue && d.kind != kindStoreFalse {
				optName = shortCandidate
				inlineVal = tok[2:]
				hasInline = true
			} else {
				optName = tok
			}
		} else {
			optName = tok
		}

		def, err := resolveOption(defs, optName)
		if err != nil {
			return nil, err
		}

		if def.kind == kindStoreTrue || def.kind == kindStoreFalse {
			if hasInline {
				return nil, fmt.Errorf("argument %s: ignored explicit argument %q", optName, inlineVal)
			}
			if err := def.apply(args, ""); err != nil {
				return nil, err
			}
			continue
		}

		var val string
		if hasInline {
			val = inlineVal
		} else {
			if i+1 >= len(argv) {
				return nil, fmt.Errorf("argument %s: expected one argument", optName)
			}
			nextTok := argv[i+1]
			if isOptionToken(defs, nextTok) {
				return nil, fmt.Errorf("argument %s: expected one argument", optName)
			}
			val = nextTok
			i++
		}

		if len(def.choices) > 0 {
			validChoice := false
			for _, c := range def.choices {
				if c == val {
					validChoice = true
					break
				}
			}
			if !validChoice {
				return nil, fmt.Errorf("argument %s: invalid choice: %q (choose from %v)", optName, val, def.choices)
			}
		}

		if err := def.apply(args, val); err != nil {
			return nil, fmt.Errorf("argument %s: %v", optName, err)
		}
	}

	return args, nil
}
