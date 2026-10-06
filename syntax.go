package cmlgo

// recognizer shares grammar and lexer with Parse but does not allocate ASTs.
type syntaxMatch struct {
	end int
	ok  bool
}
type recognizer struct {
	tokens                 []token
	grammar                *grammarState
	memo                   map[memoKey]syntaxMatch
	depth, steps, farthest int
	limited                bool
}

//nolint:cyclop // Ветви CML проверяются conformance и corpus-тестами.
func (p *recognizer) rule(name string, pos int) syntaxMatch {
	kind := ""

	switch name {
	case symbolID:
		kind = symbolId
	case symbolSTRING:
		kind = symbolString
	case symbolINT:
		kind = symbolInt
	case symbolSL_COMMENT, symbolML_COMMENT:
		return syntaxMatch{end: pos, ok: false}
	default:
		// Другие правила не требуют этой проверки.
	}

	if kind != "" {
		if p.tokens[pos].kind == kind {
			return syntaxMatch{pos + 1, true}
		}

		return syntaxMatch{end: pos, ok: false}
	}

	key := memoKey{name, pos}
	if m, ok := p.memo[key]; ok {
		return m
	}

	p.depth++

	defer func() { p.depth-- }()

	if p.depth > maxNesting {
		p.limited = true

		return syntaxMatch{end: pos, ok: false}
	}

	r, ok := p.grammar.rules[key.name]
	if !ok {
		return syntaxMatch{end: pos, ok: false}
	}

	m := p.eval(r.Expr, key.pos)

	p.memo[key] = m

	return m
}

//nolint:cyclop,funlen,gocognit,gocyclo // Ветви CML проверяются conformance и corpus-тестами.
func (p *recognizer) eval(e *expression, pos int) syntaxMatch {
	if pos > p.farthest {
		p.farthest = pos
	}

	p.steps++
	if p.steps > maxParserWork {
		p.limited = true

		return syntaxMatch{end: pos, ok: false}
	}

	switch e.Op {
	case opLit:
		if matchesLiteral(p.tokens[pos], e.Text) {
			return syntaxMatch{pos + 1, true}
		}
	case opCall:
		return p.rule(e.Text, pos)
	case opRef:
		return p.rule(symbolID, pos)
	case opAction:
		return syntaxMatch{pos, true}
	case opAssign:
		return p.eval(e.Args[0], pos)
	case opSeq:
		return p.sequence(e.Args, pos)

	case opAlt:
		best := syntaxMatch{end: pos, ok: false}

		for _, a := range e.Args {
			m := p.eval(a, pos)
			if m.ok && (!best.ok || m.end > best.end) {
				best = m
			}
		}

		return best
	case opRepeat:
		at, count := pos, 0

		for {
			m := p.eval(e.Args[0], at)
			if featureAttributeBoundary(
				e.Args[0],
				p.tokens,
				match{end: m.end, ok: m.ok, value: ASTValue{}, captures: nil},
			) {
				break
			}

			if !m.ok || m.end == at {
				break
			}

			at = m.end
			count++

			if e.Text == "?" {
				break
			}
		}

		return syntaxMatch{at, e.Text != "+" || count > 0}
	case opUnordered:
		at := pos

		used := uint64(0)

		if len(e.Args) > maxUnorderedMembers {
			p.limited = true

			return syntaxMatch{end: pos, ok: false}
		}

		for {
			index := -1
			best := syntaxMatch{end: at, ok: false}

			for i, a := range e.Args {
				if used&(uint64(1)<<i) != 0 {
					continue
				}

				m := p.eval(a, at)
				if m.ok && m.end > best.end {
					index = i
					best = m
				}
			}

			if index < 0 || index >= maxUnorderedMembers {
				break
			}

			used |= uint64(1) << index

			at = best.end
		}

		for i, a := range e.Args {
			if used&(uint64(1)<<i) == 0 {
				m := p.eval(a, at)
				if !m.ok {
					return m
				}
			}
		}

		return syntaxMatch{at, true}
	default:
		// Другие правила не требуют этой проверки.
	}

	return syntaxMatch{end: pos, ok: false}
}

func (p *recognizer) sequence(args []*expression, pos int) syntaxMatch {
	if len(args) == 0 {
		return syntaxMatch{end: pos, ok: true}
	}

	first := args[0]
	m := p.eval(first, pos)

	if m.ok {
		rest := p.sequence(args[1:], m.end)
		if rest.ok {
			return rest
		}
	}

	if first.Op == opRepeat && first.Text == "?" {
		return p.sequence(args[1:], pos)
	}

	return syntaxMatch{end: pos, ok: false}
}
