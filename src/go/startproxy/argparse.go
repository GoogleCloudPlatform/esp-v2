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

// This file ports the part of Python 3.11's argparse that start_proxy.py used:
// optional arguments only; store, store_true, append and BooleanOptionalAction
// actions; type=int; choices; prefix abbreviations (allow_abbrev); and the
// default help formatter with RawDescriptionHelpFormatter. Parsing follows
// argparse's algorithm step by step, so every command line keeps its meaning:
// the same lines are accepted, rejected or answered with help. Errors are
// reported like start_proxy.py's ArgumentParser.error() override: the full
// help and the message on stderr, and exit status 1.
//
// Known deviations, all outside realistic use: int options reject non-ASCII
// digits and values that overflow int, and help is wrapped to $COLUMNS (80 by
// default) rather than to the terminal width.

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// progName is the program name used in usage and error messages.
const progName = "startproxy"

// errHelpRequested is returned by parseArgs when -h/--help is given.
var errHelpRequested = errors.New("help requested")

// flagDef describes one option, like an argparse action.
type flagDef struct {
	// names are the option strings, e.g. {"-s", "--service"}.
	names []string
	value flagValue
	// metavar is the placeholder for the option's argument in help output.
	metavar string
	// choices, if set, are the only values the option accepts.
	choices []string
	help    string
	// hidden options (argparse.SUPPRESS) are accepted but not listed in help.
	hidden bool
}

// flagValue stores an option's value into proxyArgs.
type flagValue interface {
	// takesArg reports whether the option consumes an argument (argparse
	// nargs=None) rather than none (nargs=0).
	takesArg() bool
	// set stores arg. option is the option string used on the command line.
	set(option, arg string) error
}

// stringValue is action="store".
type stringValue struct{ p *string }

func (v stringValue) takesArg() bool { return true }
func (v stringValue) set(_, arg string) error {
	*v.p = arg
	return nil
}

// optStringValue is action="store" with default=None.
type optStringValue struct{ p **string }

func (v optStringValue) takesArg() bool { return true }
func (v optStringValue) set(_, arg string) error {
	*v.p = &arg
	return nil
}

// intValue is action="store", type=int.
type intValue struct{ p *int }

func (v intValue) takesArg() bool { return true }
func (v intValue) set(_, arg string) error {
	n, err := pyAtoi(arg)
	if err != nil {
		return err
	}
	*v.p = n
	return nil
}

// optIntValue is action="store", type=int with default=None.
type optIntValue struct{ p **int }

func (v optIntValue) takesArg() bool { return true }
func (v optIntValue) set(_, arg string) error {
	n, err := pyAtoi(arg)
	if err != nil {
		return err
	}
	*v.p = &n
	return nil
}

// appendValue is action="append".
type appendValue struct{ p *[]string }

func (v appendValue) takesArg() bool { return true }
func (v appendValue) set(_, arg string) error {
	*v.p = append(*v.p, arg)
	return nil
}

// trueValue is action="store_true".
type trueValue struct{ p *bool }

func (v trueValue) takesArg() bool { return false }
func (v trueValue) set(_, _ string) error {
	*v.p = true
	return nil
}

// boolOptionalValue is action=argparse.BooleanOptionalAction: --x sets true,
// --no-x sets false.
type boolOptionalValue struct{ p *bool }

func (v boolOptionalValue) takesArg() bool { return false }
func (v boolOptionalValue) set(option, _ string) error {
	*v.p = !strings.HasPrefix(option, "--no-")
	return nil
}

// helpValue is action="help".
type helpValue struct{}

func (helpValue) takesArg() bool        { return false }
func (helpValue) set(_, _ string) error { return errHelpRequested }

// argParser parses command lines like start_proxy.py's argparse parser.
type argParser struct {
	defs []*flagDef
	// options maps every option string to its definition.
	options map[string]*flagDef
	// order lists the option strings in definition order.
	order []string
}

func newArgParser(defs []*flagDef) *argParser {
	p := &argParser{defs: defs, options: make(map[string]*flagDef)}
	for _, d := range defs {
		for _, name := range d.names {
			p.options[name] = d
			p.order = append(p.order, name)
		}
	}
	return p
}

// parseArgs parses argv (without the program name) into proxyArgs. It returns
// errHelpRequested for -h/--help; other errors carry argparse's messages.
func parseArgs(argv []string) (*proxyArgs, error) {
	args := defaultArgs()
	if err := newArgParser(flagDefs(args)).parse(argv); err != nil {
		return nil, err
	}
	return args, nil
}

// optionMatch is a token recognized as an option (argparse's option tuple).
type optionMatch struct {
	// def is nil for an unknown option.
	def *flagDef
	// option is the option string the token matched.
	option string
	// explicit is the argument attached to the token ("--x=1", "-x1"), if any.
	explicit *string
}

func (p *argParser) parse(argv []string) error {
	// Like argparse, first classify every token as an option or not;
	// everything after "--" is not an option.
	matches := make([]*optionMatch, len(argv))
	isArg := make([]bool, len(argv))
	for i := 0; i < len(argv); i++ {
		if argv[i] == "--" {
			for j := i + 1; j < len(argv); j++ {
				isArg[j] = true
			}
			break
		}
		m, err := p.parseOptional(argv[i])
		if err != nil {
			return err
		}
		matches[i] = m
		isArg[i] = m == nil
	}

	// Then consume the options from left to right. There are no positional
	// arguments, so anything an option does not consume is unrecognized.
	var extras []string
	for i := 0; i < len(argv); {
		if matches[i] == nil {
			extras = append(extras, argv[i])
			i++
			continue
		}
		next, err := p.consumeOptional(argv, isArg, i, matches[i], &extras)
		if err != nil {
			return err
		}
		i = next
	}
	if len(extras) > 0 {
		return fmt.Errorf("unrecognized arguments: %s", strings.Join(extras, " "))
	}
	return nil
}

// negativeNumber matches tokens argparse treats as negative numbers rather
// than as options.
var negativeNumber = regexp.MustCompile(`^-\p{Nd}+$|^-\p{Nd}*\.\p{Nd}+$`)

// parseOptional mirrors argparse's _parse_optional. It returns nil if arg is
// not an option.
func (p *argParser) parseOptional(arg string) (*optionMatch, error) {
	if arg == "" || arg[0] != '-' {
		return nil, nil
	}
	if d, ok := p.options[arg]; ok {
		return &optionMatch{def: d, option: arg}, nil
	}
	if len(arg) == 1 {
		return nil, nil
	}
	if eq := strings.IndexByte(arg, '='); eq >= 0 {
		if d, ok := p.options[arg[:eq]]; ok {
			explicit := arg[eq+1:]
			return &optionMatch{def: d, option: arg[:eq], explicit: &explicit}, nil
		}
	}
	matches := p.prefixMatches(arg)
	if len(matches) > 1 {
		options := make([]string, len(matches))
		for i, m := range matches {
			options[i] = m.option
		}
		return nil, fmt.Errorf("ambiguous option: %s could match %s", arg, strings.Join(options, ", "))
	}
	if len(matches) == 1 {
		return &matches[0], nil
	}
	if negativeNumber.MatchString(arg) || strings.Contains(arg, " ") {
		return nil, nil
	}
	// An option this parser doesn't know.
	return &optionMatch{option: arg}, nil
}

// prefixMatches mirrors argparse's _get_option_tuples: the options arg is an
// abbreviation of or, for a single dash, a short option with its value
// attached ("-sVALUE").
func (p *argParser) prefixMatches(arg string) []optionMatch {
	var result []optionMatch
	if arg[1] == '-' {
		prefix := arg
		var explicit *string
		if eq := strings.IndexByte(arg, '='); eq >= 0 {
			prefix = arg[:eq]
			e := arg[eq+1:]
			explicit = &e
		}
		for _, opt := range p.order {
			if strings.HasPrefix(opt, prefix) {
				result = append(result, optionMatch{def: p.options[opt], option: opt, explicit: explicit})
			}
		}
		return result
	}
	_, size := utf8.DecodeRuneInString(arg[1:])
	short, shortExplicit := arg[:1+size], arg[1+size:]
	for _, opt := range p.order {
		if opt == short {
			e := shortExplicit
			result = append(result, optionMatch{def: p.options[opt], option: opt, explicit: &e})
		} else if strings.HasPrefix(opt, arg) {
			result = append(result, optionMatch{def: p.options[opt], option: opt})
		}
	}
	return result
}

// consumeOptional mirrors argparse's consume_optional for the option at
// argv[i]. It returns the index of the next unconsumed token.
func (p *argParser) consumeOptional(argv []string, isArg []bool, i int, m *optionMatch, extras *[]string) (int, error) {
	type action struct {
		def         *flagDef
		option, arg string
	}
	var actions []action
	def, option, explicit := m.def, m.option, m.explicit
	stop := i + 1
	for {
		if def == nil {
			*extras = append(*extras, argv[i])
			return i + 1, nil
		}
		if explicit != nil {
			if !def.value.takesArg() && option[1] != '-' && *explicit != "" {
				// A single-dash option without an argument followed by more
				// characters: those are further single-dash options ("-tN").
				actions = append(actions, action{def, option, ""})
				r, size := utf8.DecodeRuneInString(*explicit)
				option = option[:1] + string(r)
				d, ok := p.options[option]
				if !ok {
					return 0, argError(def, "ignored explicit argument %s", pyRepr(*explicit))
				}
				def = d
				if rest := (*explicit)[size:]; rest != "" {
					explicit = &rest
				} else {
					explicit = nil
				}
				continue
			}
			if !def.value.takesArg() {
				return 0, argError(def, "ignored explicit argument %s", pyRepr(*explicit))
			}
			actions = append(actions, action{def, option, *explicit})
		} else if def.value.takesArg() {
			if i+1 >= len(argv) || !isArg[i+1] {
				return 0, argError(def, "expected one argument")
			}
			actions = append(actions, action{def, option, argv[i+1]})
			stop = i + 2
		} else {
			actions = append(actions, action{def, option, ""})
		}
		break
	}
	for _, a := range actions {
		if err := takeAction(a.def, a.option, a.arg); err != nil {
			return 0, err
		}
	}
	return stop, nil
}

func takeAction(def *flagDef, option, arg string) error {
	if len(def.choices) > 0 && !contains(def.choices, arg) {
		quoted := make([]string, len(def.choices))
		for i, c := range def.choices {
			quoted[i] = pyRepr(c)
		}
		return argError(def, "invalid choice: %s (choose from %s)", pyRepr(arg), strings.Join(quoted, ", "))
	}
	if err := def.value.set(option, arg); err != nil {
		if errors.Is(err, errHelpRequested) {
			return err
		}
		return argError(def, "%v", err)
	}
	return nil
}

func argError(def *flagDef, format string, a ...interface{}) error {
	return fmt.Errorf("argument %s: %s", strings.Join(def.names, "/"), fmt.Sprintf(format, a...))
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// pyAtoi converts s like Python's int(s): surrounding whitespace (exactly the
// characters unicode.IsSpace accepts), a sign and single underscores between
// digits are allowed.
func pyAtoi(s string) (int, error) {
	invalid := fmt.Errorf("invalid int value: %s", pyRepr(s))
	t := strings.TrimFunc(s, unicode.IsSpace)
	digits := t
	if digits != "" && (digits[0] == '+' || digits[0] == '-') {
		digits = digits[1:]
	}
	if digits == "" || digits[0] == '_' || digits[len(digits)-1] == '_' || strings.Contains(digits, "__") {
		return 0, invalid
	}
	for _, r := range digits {
		if r != '_' && (r < '0' || r > '9') {
			return 0, invalid
		}
	}
	n, err := strconv.Atoi(strings.ReplaceAll(t, "_", ""))
	if err != nil {
		return 0, invalid
	}
	return n, nil
}

// pyRepr formats s like Python's repr() of a str, which argparse uses to quote
// values in error messages.
func pyRepr(s string) string {
	quote := '\''
	if strings.ContainsRune(s, '\'') && !strings.ContainsRune(s, '"') {
		quote = '"'
	}
	var b strings.Builder
	b.WriteRune(quote)
	for _, r := range s {
		switch {
		case r == quote || r == '\\':
			b.WriteRune('\\')
			b.WriteRune(r)
		case r == '\t':
			b.WriteString(`\t`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&b, `\x%02x`, r)
		case r < 0x80 || unicode.IsPrint(r):
			b.WriteRune(r)
		case r <= 0xff:
			fmt.Fprintf(&b, `\x%02x`, r)
		case r <= 0xffff:
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			fmt.Fprintf(&b, `\U%08x`, r)
		}
	}
	b.WriteRune(quote)
	return b.String()
}

// usage is the option's entry in the usage message, e.g. "-s SERVICE".
func (d *flagDef) usage() string {
	if d.value.takesArg() {
		return d.names[0] + " " + d.metavar
	}
	if _, ok := d.value.(boolOptionalValue); ok {
		return strings.Join(d.names, " | ")
	}
	return d.names[0]
}

// invocation is the option's heading in the help output, e.g.
// "-s SERVICE, --service SERVICE".
func (d *flagDef) invocation() string {
	if !d.value.takesArg() {
		return strings.Join(d.names, ", ")
	}
	parts := make([]string, len(d.names))
	for i, name := range d.names {
		parts[i] = name + " " + d.metavar
	}
	return strings.Join(parts, ", ")
}

// helpWidth is the text width argparse's HelpFormatter uses: $COLUMNS (80 by
// default) minus 2.
func helpWidth() int {
	columns, err := pyAtoi(os.Getenv("COLUMNS"))
	if err != nil || columns <= 0 {
		columns = 80
	}
	return columns - 2
}

// usageBlock mirrors HelpFormatter._format_usage.
func (p *argParser) usageBlock(width int) string {
	const prefix = "usage: "
	parts := []string{progName}
	for _, d := range p.defs {
		if !d.hidden {
			parts = append(parts, "["+d.usage()+"]")
		}
	}
	usage := strings.Join(parts, " ")
	if runeLen(prefix)+runeLen(usage) > width {
		var lines []string
		if float64(runeLen(prefix)+runeLen(progName)) <= 0.75*float64(width) {
			indent := strings.Repeat(" ", runeLen(prefix)+runeLen(progName)+1)
			lines = usageLines(parts, indent, prefix, width)
		} else {
			// A long program name goes on its own line.
			lines = append([]string{progName}, usageLines(parts[1:], strings.Repeat(" ", runeLen(prefix)), "", width)...)
		}
		usage = strings.Join(lines, "\n")
	}
	return prefix + usage + "\n\n"
}

// usageLines mirrors get_lines in HelpFormatter._format_usage.
func usageLines(parts []string, indent, prefix string, width int) []string {
	var lines, line []string
	lineLen := runeLen(indent) - 1
	if prefix != "" {
		lineLen = runeLen(prefix) - 1
	}
	for _, part := range parts {
		if lineLen+1+runeLen(part) > width && len(line) > 0 {
			lines = append(lines, indent+strings.Join(line, " "))
			line = nil
			lineLen = runeLen(indent) - 1
		}
		line = append(line, part)
		lineLen += runeLen(part) + 1
	}
	if len(line) > 0 {
		lines = append(lines, indent+strings.Join(line, " "))
	}
	if prefix != "" && len(lines) > 0 {
		lines[0] = lines[0][len(indent):]
	}
	return lines
}

var longBreak = regexp.MustCompile(`\n\n\n+`)

// formatHelp renders the --help output (argparse's format_help).
func (p *argParser) formatHelp(width int) string {
	const indent = 2
	maxLen := 0
	for _, d := range p.defs {
		if n := runeLen(d.invocation()) + indent; !d.hidden && n > maxLen {
			maxLen = n
		}
	}
	helpPosition := min(maxLen+2, min(24, max(width-20, 2*indent)))
	textWidth := max(width-helpPosition, 11)
	actionWidth := helpPosition - indent - 2

	var items strings.Builder
	for _, d := range p.defs {
		if d.hidden {
			continue
		}
		inv := d.invocation()
		if d.help == "" {
			fmt.Fprintf(&items, "%*s%s\n", indent, "", inv)
			continue
		}
		firstIndent := helpPosition
		if runeLen(inv) <= actionWidth {
			fmt.Fprintf(&items, "%*s%s%*s  ", indent, "", inv, actionWidth-runeLen(inv), "")
			firstIndent = 0
		} else {
			fmt.Fprintf(&items, "%*s%s\n", indent, "", inv)
		}
		for i, line := range wrapText(d.help, textWidth) {
			pad := helpPosition
			if i == 0 {
				pad = firstIndent
			}
			fmt.Fprintf(&items, "%*s%s\n", pad, "", line)
		}
	}

	help := p.usageBlock(width) + helpDescription + "\n\n" + "\noptions:\n" + items.String() + "\n"
	return strings.Trim(longBreak.ReplaceAllString(help, "\n\n"), "\n") + "\n"
}

func runeLen(s string) int { return utf8.RuneCountInString(s) }

// wrapText wraps text like Python's textwrap.wrap(text, width) with default
// options, which argparse uses for option help.
func wrapText(text string, width int) []string {
	chunks := splitChunks([]rune(text))
	// Like textwrap, use the chunks as a stack with the next chunk on top.
	for i, j := 0, len(chunks)-1; i < j; i, j = i+1, j-1 {
		chunks[i], chunks[j] = chunks[j], chunks[i]
	}
	top := func() []rune { return chunks[len(chunks)-1] }
	pop := func() []rune {
		c := top()
		chunks = chunks[:len(chunks)-1]
		return c
	}
	isBlank := func(c []rune) bool { return strings.TrimSpace(string(c)) == "" }

	var lines []string
	for len(chunks) > 0 {
		var line [][]rune
		lineLen := 0
		if isBlank(top()) && len(lines) > 0 {
			pop()
		}
		for len(chunks) > 0 && lineLen+len(top()) <= width {
			lineLen += len(top())
			line = append(line, pop())
		}
		if len(chunks) > 0 && len(top()) > width {
			// A chunk too long for any line: textwrap's _handle_long_word
			// breaks it, after a hyphen if possible.
			spaceLeft := width - lineLen
			if width < 1 {
				spaceLeft = 1
			}
			chunk := top()
			end := spaceLeft
			if hyphen := lastIndexRune(chunk[:spaceLeft], '-'); hyphen > 0 && strings.Trim(string(chunk[:hyphen]), "-") != "" {
				end = hyphen + 1
			}
			line = append(line, chunk[:end])
			chunks[len(chunks)-1] = chunk[end:]
		}
		if len(line) > 0 && isBlank(line[len(line)-1]) {
			line = line[:len(line)-1]
		}
		if len(line) > 0 {
			var b strings.Builder
			for _, c := range line {
				b.WriteString(string(c))
			}
			lines = append(lines, b.String())
		}
	}
	return lines
}

func lastIndexRune(rs []rune, r rune) int {
	for i := len(rs) - 1; i >= 0; i-- {
		if rs[i] == r {
			return i
		}
	}
	return -1
}

// splitChunks splits text into textwrap's chunks (TextWrapper.wordsep_re):
// whitespace runs, em-dashes, and words, with hyphenated words split after
// their hyphens.
func splitChunks(t []rune) [][]rune {
	var chunks [][]rune
	for pos := 0; pos < len(t); {
		var end int
		switch {
		case isWrapSpace(t[pos]):
			end = pos
			for end < len(t) && isWrapSpace(t[end]) {
				end++
			}
		case t[pos] == '-' && pos > 0 && isWordPunct(t[pos-1]) && emDashEnd(t, pos) > 0:
			end = emDashEnd(t, pos)
		default:
			end = wordEnd(t, pos)
		}
		chunks = append(chunks, t[pos:end])
		pos = end
	}
	return chunks
}

// emDashEnd returns the end of an em-dash (two or more hyphens followed by a
// word character) starting at i, or 0 if there is none.
func emDashEnd(t []rune, i int) int {
	j := i
	for j < len(t) && t[j] == '-' {
		j++
	}
	if j-i >= 2 && j < len(t) && isWordChar(t[j]) {
		return j
	}
	return 0
}

// wordEnd returns the end of the shortest word chunk starting at pos.
func wordEnd(t []rune, pos int) int {
	at := func(i int) rune {
		if i < 0 || i >= len(t) {
			return -1
		}
		return t[i]
	}
	for j := pos + 1; j < len(t); j++ {
		// Hyphenated word: break after a hyphen that follows two letters (or
		// letter, hyphen, letter) and precedes letter, optional hyphen, letter.
		if t[j] == '-' {
			before := (isLetter(at(j-2)) && isLetter(at(j-1))) ||
				(isLetter(at(j-3)) && at(j-2) == '-' && isLetter(at(j-1)))
			after := isLetter(at(j+1)) && (isLetter(at(j+2)) || (at(j+2) == '-' && isLetter(at(j+3))))
			if before && after {
				return j + 1
			}
		}
		// End of the word.
		if isWrapSpace(t[j]) {
			return j
		}
		// Right before an em-dash.
		if isWordPunct(t[j-1]) && emDashEnd(t, j) > 0 {
			return j
		}
	}
	return len(t)
}

// Character classes of textwrap's regular expressions, with Python's
// Unicode semantics.
func isWrapSpace(r rune) bool { return strings.ContainsRune("\t\n\v\f\r ", r) }
func isWordChar(r rune) bool  { return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r) }
func isLetter(r rune) bool    { return isWordChar(r) && !unicode.Is(unicode.Nd, r) }
func isWordPunct(r rune) bool { return isWordChar(r) || strings.ContainsRune(`!"'&.,?`, r) }
