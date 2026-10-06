package cmlgo

import (
	"fmt"
	"slices"
	"strings"
)

type expression struct {
	Op   string        `json:"op"`
	Text string        `json:"text"`
	Mode string        `json:"mode"`
	Args []*expression `json:"args"`
}
type rule struct {
	Kind string      `json:"kind"`
	Expr *expression `json:"expr"`
}
type grammarState struct {
	rules    map[string]rule
	literals []string
	err      error
}

type capture struct {
	field string
	value ASTValue
}
type match struct {
	end      int
	ok       bool
	value    ASTValue
	captures []capture
}
type memoKey struct {
	name string
	pos  int
}
type parser struct {
	tokens   []token
	grammar  *grammarState
	memo     map[memoKey]match
	farthest int
	expected map[string]bool
	depth    int
	steps    int
	limited  bool
}

func (p *parser) failure(pos int, expected string) match {
	if pos > p.farthest {
		p.farthest = pos
		p.expected = map[string]bool{}
	}

	if pos == p.farthest {
		p.expected[expected] = true
	}

	return match{end: pos, ok: false, value: ASTValue{}, captures: nil}
}

//nolint:cyclop,funlen,gocognit,gocyclo // Ветви CML проверяются conformance и corpus-тестами.
func (p *parser) rule(name string, pos int) match {
	if name == symbolID || name == symbolSTRING || name == symbolINT {
		kind := map[string]string{symbolID: symbolId, symbolSTRING: symbolString, symbolINT: symbolInt}[name]
		if p.tokens[pos].kind == kind {
			return match{
				end:      pos + 1,
				ok:       true,
				value:    ASTValue{Text: p.tokens[pos].text, Position: p.tokens[pos].pos, Node: nil, RefType: ""},
				captures: nil,
			}
		}

		return p.failure(pos, name)
	}

	if name == symbolSL_COMMENT || name == symbolML_COMMENT {
		return match{end: pos, ok: false, value: ASTValue{}, captures: nil}
	}

	key := memoKey{name, pos}
	if m, ok := p.memo[key]; ok {
		return m
	}

	p.depth++

	defer func() { p.depth-- }()

	if p.depth > maxNesting {
		p.limited = true

		return p.failure(pos, "nesting limit")
	}

	r, ok := p.grammar.rules[key.name]
	if !ok {
		return p.failure(pos, name)
	}

	m := p.eval(r.Expr, key.pos)
	if m.ok {
		if r.Kind == symbolDatatype || r.Kind == symbolTerminal || r.Kind == symbolEnum {
			var b []byte

			for _, t := range p.tokens[pos:m.end] {
				b = append(b, t.text...)
			}

			m.value = ASTValue{Text: string(b), Position: p.tokens[pos].pos, Node: nil, RefType: ""}
			m.captures = nil
		} else if len(m.captures) > 0 || name == kindContextMappingModel || m.value.Node == nil {
			node := &Node{Kind: name, Fields: map[string][]ASTValue{}, Position: p.tokens[pos].pos, parent: nil}

			for _, c := range m.captures {
				node.Fields[c.field] = append(node.Fields[c.field], c.value)
			}

			m.value = ASTValue{Node: node, Position: node.Position, Text: "", RefType: ""}
			m.captures = nil
		}
	}

	p.memo[key] = m

	return m
}

//nolint:cyclop,funlen,gocognit,gocyclo // Ветви CML проверяются conformance и corpus-тестами.
func (p *parser) eval(e *expression, pos int) match {
	p.steps++
	if p.steps > maxParserWork {
		p.limited = true

		return match{end: pos, ok: false, value: ASTValue{}, captures: nil}
	}

	switch e.Op {
	case opLit:
		if matchesLiteral(p.tokens[pos], e.Text) {
			return match{
				end:      pos + 1,
				ok:       true,
				value:    ASTValue{Text: e.Text, Position: p.tokens[pos].pos, Node: nil, RefType: ""},
				captures: nil,
			}
		}

		return p.failure(pos, e.Text)
	case opCall:
		return p.rule(e.Text, pos)
	case opRef:
		m := p.rule(symbolID, pos)
		if m.ok {
			m.value.RefType = e.Text
		}

		return m
	case opAction:
		return match{end: pos, ok: true, value: ASTValue{}, captures: nil}
	case opAssign:
		m := p.eval(e.Args[0], pos)
		if m.ok {
			v := m.value

			if e.Mode == "?=" {
				v.Text = symbolTrue
			}

			m.captures = []capture{{field: e.Text, value: v}}
			m.value = ASTValue{}
		}

		return m
	case opSeq:
		return p.sequence(e.Args, pos)
	case opAlt:
		best := match{end: pos, ok: false, value: ASTValue{}, captures: nil}

		for _, a := range e.Args {
			m := p.eval(a, pos)
			if m.ok && (!best.ok || m.end > best.end) {
				best = m
			}
		}

		return best
	case opRepeat:
		out := match{end: pos, ok: true, value: ASTValue{}, captures: nil}
		count := 0

		for {
			m := p.eval(e.Args[0], out.end)
			if featureAttributeBoundary(e.Args[0], p.tokens, m) {
				break
			}

			if !m.ok || m.end == out.end {
				break
			}

			count++

			out.end = m.end
			out.captures = append(out.captures, m.captures...)
			out.value = m.value

			if e.Text == "?" {
				break
			}
		}

		if e.Text == "+" && count == 0 {
			return match{end: pos, ok: false, value: ASTValue{}, captures: nil}
		}

		return out
	case opUnordered:
		out := match{end: pos, ok: true, value: ASTValue{}, captures: nil}

		used := uint64(0)

		if len(e.Args) > maxUnorderedMembers {
			p.limited = true

			return p.failure(pos, "unordered group limit")
		}

		for {
			index := -1
			best := match{end: out.end, ok: false, value: ASTValue{}, captures: nil}

			for i, a := range e.Args {
				if used&(uint64(1)<<i) != 0 {
					continue
				}

				m := p.eval(a, out.end)
				if m.ok && m.end > best.end {
					index = i
					best = m
				}
			}

			if index < 0 || index >= maxUnorderedMembers {
				break
			}

			used |= uint64(1) << index

			out.end = best.end
			out.captures = append(out.captures, best.captures...)

			if best.value.Node != nil {
				out.value = best.value
			}
		}

		for i, a := range e.Args {
			if used&(uint64(1)<<i) == 0 {
				m := p.eval(a, out.end)
				if !m.ok {
					return m
				}

				out.captures = append(out.captures, m.captures...)
			}
		}

		return out
	}

	return p.failure(pos, "unknown grammar expression")
}

// sequence retries an optional element when consuming it prevents the next
// required element from matching (e.g. an operation without a return type).
func (p *parser) sequence(args []*expression, pos int) match {
	if len(args) == 0 {
		return match{end: pos, ok: true, value: ASTValue{}, captures: nil}
	}

	first := args[0]
	m := p.eval(first, pos)

	if m.ok {
		rest := p.sequence(args[1:], m.end)
		if rest.ok {
			captures := append([]capture(nil), m.captures...)

			captures = append(captures, rest.captures...)

			value := rest.value

			if value.Node == nil && value.Text == "" {
				value = m.value
			}

			return match{end: rest.end, ok: true, value: value, captures: captures}
		}
	}

	if first.Op == opRepeat && first.Text == "?" {
		return p.sequence(args[1:], pos)
	}

	return match{end: pos, ok: false, value: ASTValue{}, captures: nil}
}

// Parse parses one UTF-8 CML document. It performs syntax checks only; imports
// and semantic checks are provided by Validate and ValidateFile.
func Parse(file string, source []byte) Result {
	return parseDocument(file, source)
}

// CheckSyntax checks the complete grammar without building an AST or resolving imports.
func CheckSyntax(file string, source []byte) Result {
	return CheckSyntaxWithDialect(file, source, Dialect{Aliases: nil, Original: false})
}

// CheckSyntaxWithDialect проверяет грамматику выбранного диалекта без AST.
func CheckSyntaxWithDialect(file string, source []byte, dialect Dialect) Result {
	g := grammarForDialect(dialect)
	tokens, issue := lexWithDialect(file, source, g.literals, false, dialect)

	if issue != nil {
		return Result{Diagnostics: []Diagnostic{*issue}, Valid: false, Documents: nil, Model: nil}
	}

	p := recognizer{
		tokens:   tokens,
		grammar:  g,
		memo:     map[memoKey]syntaxMatch{},
		depth:    0,
		steps:    0,
		farthest: 0,
		limited:  false,
	}
	m := p.rule(kindContextMappingModel, 0)

	if m.ok && m.end == len(tokens)-1 && !p.limited {
		return Result{Valid: true, Diagnostics: []Diagnostic{}, Documents: nil, Model: nil}
	}

	at := p.farthest
	if at >= len(tokens) {
		at = len(tokens) - 1
	}

	code := symbolSyntax

	if p.limited {
		code = symbolLimit
	}

	return Result{
		Diagnostics: []Diagnostic{
			{Severity: symbolError, Code: code, Message: "Invalid CML syntax", Position: tokens[at].pos},
		}, Valid: false, Documents: nil, Model: nil,
	}
}

func parseDocument(file string, source []byte) Result {
	return ParseWithDialect(file, source, Dialect{Aliases: nil, Original: false})
}

// ParseWithDialect разбирает документ с aliases или строгой исходной грамматикой.
//
//nolint:cyclop,funlen,gocognit,gocyclo,nestif // Ветви CML проверяются conformance и corpus-тестами.
func ParseWithDialect(file string, source []byte, dialect Dialect) Result {
	g := grammarForDialect(dialect)
	if g.err != nil {
		return Result{
			Diagnostics: []Diagnostic{
				{
					Severity: symbolError,
					Code:     "grammar",
					Message:  g.err.Error(),
					Position: Position{File: file, Line: 1, Column: 1, Offset: 0},
				},
			}, Valid: false, Documents: nil, Model: nil,
		}
	}

	tokens, issue := lexWithDialect(file, source, g.literals, false, dialect)
	if issue != nil {
		return Result{Diagnostics: []Diagnostic{*issue}, Valid: false, Documents: nil, Model: nil}
	}

	p := parser{
		tokens:   tokens,
		grammar:  g,
		memo:     map[memoKey]match{},
		expected: map[string]bool{}, farthest: 0, depth: 0, steps: 0, limited: false,
	}
	m := p.rule(kindContextMappingModel, 0)

	if !m.ok || m.end != len(tokens)-1 || p.limited {
		at := p.farthest
		if m.ok && m.end > at {
			at = m.end
		}

		if at >= len(tokens) {
			at = len(tokens) - 1
		}

		var expected []string

		for s := range p.expected {
			expected = append(expected, s)
		}

		slices.Sort(expected)

		if len(expected) > maxExpectedTokens {
			expected = expected[:maxExpectedTokens]
		}

		msg := fmt.Sprintf("Unexpected %q; expected %s", tokens[at].raw, strings.Join(expected, ", "))
		code := symbolSyntax

		if p.limited {
			code = symbolLimit
			msg = "Parser nesting or work limit exceeded"
		}

		return Result{
			Diagnostics: []Diagnostic{
				{Severity: symbolError, Code: code, Message: msg, Position: tokens[at].pos},
			},
			Valid:     false,
			Documents: nil,
			Model:     nil,
		}
	}

	attachParents(m.value.Node, nil)

	return Result{Valid: true, Diagnostics: []Diagnostic{}, Model: m.value.Node, Documents: nil}
}

func attachParents(n, parent *Node) {
	if n == nil {
		return
	}

	n.parent = parent
	for _, values := range n.Fields {
		for _, v := range values {
			if v.Node != nil {
				attachParents(v.Node, n)
			}
		}
	}
}
