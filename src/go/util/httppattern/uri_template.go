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
	"bytes"
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

// The info about a variable binding {variable=subpath} in the template.
type variable struct {
	// Specifies the range of segments [start_segment, end_segment) the
	// variable binds to. Both start_segment and end_segment are 0 based.
	// end_segment can also be negative, which means that the position is
	// specified relative to the end such that -1 corresponds to the end
	// of the path.
	StartSegment int
	EndSegment   int

	// The path of the protobuf field the variable binds to.
	FieldPath []string

	// Do we have a ** in the variable template?
	HasWildcardPath bool
}

// HTTP Template Grammar:
// Questions:
//
// Template = "/" | "/" Segments [ Verb ] ;
// Segments = Segment { "/" Segment } ;
// Segment  = "*" | "**" | LITERAL | Variable ;
// Variable = "{" FieldPath [ "=" Segments ] "}" ;
// FieldPath = IDENT { "." IDENT } ;
// Verb     = ":" LITERAL ;
type parser struct {
	input      string
	tb         int
	te         int
	inVariable bool
	segments   []string
	verb       string
	variables  []*variable
}

func Parse(ht string) *UriTemplate {
	if ht == "/" {
		return &UriTemplate{}
	}

	p := parser{
		input: ht,
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

func (p *parser) parse() bool {
	if !p.parseTemplate() || !p.consumeAllInput() {
		return false
	}

	p.postProcessVariables()
	return true
}

// only constant path segments are allowed after '**'.
func (p *parser) validateParts() bool {
	foundWildCard := false
	for i := 0; i < len(p.segments); i += 1 {
		if !foundWildCard {
			if p.segments[i] == DoubleWildCardKey {
				foundWildCard = true
			}
		} else if p.segments[i] == SingleParameterKey || p.segments[i] == SingleWildCardKey || p.segments[i] == DoubleWildCardKey {
			return false
		}
	}

	return true
}

// Template = "/" Segments [ Verb ] ;
func (p *parser) parseTemplate() bool {
	// Expected '/'
	if !p.consume('/') {
		return false
	}

	if !p.parseSegments() {
		return false
	}

	if p.ensureCurrent() && p.currentChar() == ':' {
		if !p.parseVerb() {
			return false
		}
	}

	return true
}

// Segments = Segment { "/" Segment } ;
func (p *parser) parseSegments() bool {
	if !p.parseSegment() {
		return false
	}

	for {
		if !p.consume('/') {
			break
		}
		if !p.parseSegment() {
			return false
		}
	}

	return true
}

// Segment  = "*" | "**" | LITERAL | Variable ;
func (p *parser) parseSegment() bool {
	markVariableHasWildcardPath := func() bool {
		if p.inVariable && len(p.variables) > 0 {
			p.currentVariable().HasWildcardPath = true
			return true
		}
		// something's wrong we're not in a variable
		return false
	}

	if !p.ensureCurrent() {
		return false
	}
	switch p.currentChar() {
	case '*':
		p.consume('*')
		if p.consume('*') {
			// **
			p.segments = append(p.segments, "**")
			if p.inVariable {
				return markVariableHasWildcardPath()
			}
			return true
		} else {
			p.segments = append(p.segments, "*")
			return true
		}
	case '{':
		return p.parseVariable()
	default:
		return p.parseLiteralSegment()
	}
}

// Variable = "{" FieldPath [ "=" Segments ] "}" ;
func (p *parser) parseVariable() bool {
	if !p.consume('{') {
		return false
	}
	if !p.startVariable() {
		return false
	}
	if !p.parseFieldPath() {
		return false
	}
	if p.consume('=') {
		if !p.parseSegments() {
			return false
		}
	} else {
		p.segments = append(p.segments, "*")
	}

	if !p.endVariable() {
		return false
	}
	if !p.consume('}') {
		return false
	}

	return true
}

// FieldPath = IDENT { "." IDENT } ;
func (p *parser) parseFieldPath() bool {
	if !p.parseIdentifier() {
		return false
	}

	for p.consume('.') {
		if !p.parseIdentifier() {
			return false
		}
	}
	return true
}

// Verb     = ":" LITERAL ;
func (p *parser) parseVerb() bool {
	if !p.consume(':') {
		return false
	}
	verb, result := p.parseLiteral()
	if !result {
		return false
	}

	p.verb = verb
	return true
}

func (p *parser) parseIdentifier() bool {
	addFieldIdentifier := func(id string) bool {
		if p.inVariable && len(p.variables) > 0 {
			p.currentVariable().FieldPath = append(p.currentVariable().FieldPath, id)
			return true
		} else {
			// something's wrong we're not in a variable
			return false
		}
	}

	var idf bytes.Buffer
	// Initialize to false to handle empty literal.
	result := false
	for p.nextChar() {
		switch c := p.currentChar(); c {
		case '.':
			fallthrough
		case '}':
			fallthrough
		case '=':
			return result && addFieldIdentifier(idf.String())
		default:
			p.consume(c)
			idf.WriteByte(c)
			break
		}
		result = true
	}

	return result && addFieldIdentifier(idf.String())
}

func (p *parser) parseLiteral() (string, bool) {
	var buffer bytes.Buffer
	if !p.ensureCurrent() {
		return "", false
	}

	// Initialize to false in case we encounter an empty literal.
	result := false

	for {
		switch c := p.currentChar(); c {
		case '/':
			fallthrough
		case ':':
			fallthrough
		case '}':
			return buffer.String(), result
		default:
			p.consume(c)
			buffer.WriteByte(c)
			break
		}
		result = true

		if !p.nextChar() {
			break
		}
	}

	return buffer.String(), result
}

func (p *parser) parseLiteralSegment() bool {
	l, result := p.parseLiteral()
	if !result {
		return false
	}

	p.segments = append(p.segments, l)
	return true
}

func (p *parser) consume(c byte) bool {
	if p.tb >= p.te && !p.nextChar() {
		return false
	}

	if p.currentChar() != c {
		return false
	}

	p.tb += 1
	return true
}

func (p *parser) consumeAllInput() bool {
	return p.tb >= len(p.input)
}

func (p *parser) currentChar() byte {
	if p.tb < p.te && p.te <= len(p.input) {
		return p.input[p.te-1]
	}

	return InvalidChar
}

func (p *parser) ensureCurrent() bool {
	return p.tb < p.te || p.nextChar()
}

func (p *parser) nextChar() bool {
	if p.te < len(p.input) {
		p.te += 1
		return true
	}
	return false
}

func (p *parser) currentVariable() *variable {
	if len(p.variables) == 0 {
		return nil
	}
	return p.variables[len(p.variables)-1]
}

func (p *parser) startVariable() bool {
	if !p.inVariable {
		p.variables = append(p.variables, &variable{
			StartSegment:    len(p.segments),
			HasWildcardPath: false,
		})
		p.inVariable = true
		return true
	}

	// nested variables are not allowed
	return false
}

func (p *parser) endVariable() bool {
	var validateVariable = func(v *variable) bool {
		return len(v.FieldPath) > 0 && v.StartSegment < v.EndSegment && v.EndSegment <= len(p.segments)
	}

	if p.inVariable && len(p.variables) > 0 {
		p.currentVariable().EndSegment = len(p.segments)
		p.inVariable = false
		return validateVariable(p.currentVariable())
	}

	// something's wrong we're not in a variable
	return false
}

func (p *parser) postProcessVariables() {
	for _, v := range p.variables {
		if v.HasWildcardPath {
			// if the variable contains a '**', store the end_position
			// relative to the end, such that -1 corresponds to the end
			// of the path. As we only support fixed path after '**',
			// this will allow the matcher code to reconstruct the variable
			// value based on the url segments.
			v.EndSegment = v.EndSegment - len(p.segments) - 1

			if p.verb != "" {
				// a custom verb will add an additional segment, so
				// the end_position needs a -1
				v.EndSegment -= 1
			}
		}
	}
}
