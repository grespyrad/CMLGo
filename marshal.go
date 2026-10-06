package cmlgo

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"
)

type emitted struct {
	tokens []string
	used   map[string]int
	ok     bool
}

func cloneUsed(used map[string]int) map[string]int {
	out := make(map[string]int, len(used))
	maps.Copy(out, used)

	return out
}

func consumed(used map[string]int) int {
	n := 0

	for _, count := range used {
		n += count
	}

	return n
}

func marshalNode(node *Node) ([]byte, error) {
	result := emitRule(kindContextMappingModel, node)
	if !result.ok {
		return nil, fmt.Errorf("serialize grammar fields: %w", ErrInvalidModel)
	}

	data := []byte(strings.Join(result.tokens, " ") + "\n")
	if result := Parse("serialized.cml", data); !result.Valid {
		return nil, fmt.Errorf(
			"serialized CML failed syntax validation: %s: %w",
			result.Diagnostics[0].Message,
			ErrInvalidModel,
		)
	}

	return data, nil
}

func emitRule(name string, node *Node) emitted {
	rule, ok := loadGrammar().rules[name]
	if !ok || !acceptsKind(name, node.Kind, map[string]bool{}) {
		return emitted{}
	}

	result := emitExpr(rule.Expr, node, map[string]int{})
	if result.ok {
		for field, values := range node.Fields {
			if result.used[field] != len(values) {
				return emitted{}
			}
		}
	}

	return result
}

//nolint:cyclop,funlen,gocognit,gocyclo // Ветви CML проверяются conformance и corpus-тестами.
func emitExpr(e *expression, node *Node, used map[string]int) emitted {
	fail := emitted{used: used, tokens: nil, ok: false}

	switch e.Op {
	case opLit:
		return emitted{tokens: []string{e.Text}, used: used, ok: true}
	case opAction:
		return emitted{used: used, ok: true, tokens: nil}
	case opCall:
		if e.Text == symbolSL_COMMENT || e.Text == symbolML_COMMENT {
			return fail
		}

		rule, ok := loadGrammar().rules[e.Text]
		if !ok {
			return fail
		}

		if rule.Kind == symbolTerminal {
			return emitExpr(rule.Expr, node, used)
		}

		result := emitRule(e.Text, node)
		if result.ok {
			result.used = cloneUsed(used)

			for field, count := range node.Fields {
				result.used[field] = len(count)
			}
		}

		return result
	case opAssign:
		index := used[e.Text]
		values := node.Fields[e.Text]

		if index >= len(values) {
			return fail
		}

		value := values[index]

		var result emitted

		switch {
		case e.Mode == "?=":
			result = emitFlag(e.Args[0])
		case value.Node != nil:
			result = emitNodeValue(e.Args[0], value.Node)
		default:
			result = emitScalar(e.Args[0], value)
		}

		if !result.ok {
			return fail
		}

		result.used = cloneUsed(used)
		result.used[e.Text] = index + 1

		return result
	case opSeq, opUnordered:
		result := emitted{used: cloneUsed(used), ok: true, tokens: nil}

		for _, arg := range e.Args {
			part := emitExpr(arg, node, result.used)
			if !part.ok {
				return fail
			}

			result.tokens = append(result.tokens, part.tokens...)
			result.used = part.used
		}

		return result
	case opAlt:
		best := fail

		for _, arg := range e.Args {
			result := emitExpr(arg, node, cloneUsed(used))
			if result.ok && (!best.ok || consumed(result.used) > consumed(best.used)) {
				best = result
			}
		}

		return best
	case opRepeat:
		result := emitted{used: cloneUsed(used), ok: true, tokens: nil}
		count := 0

		for {
			part := emitExpr(e.Args[0], node, cloneUsed(result.used))
			if !part.ok || consumed(part.used) <= consumed(result.used) {
				break
			}

			result.tokens = append(result.tokens, part.tokens...)
			result.used = part.used
			count++

			if e.Text == "?" {
				break
			}
		}

		if e.Text == "+" && count == 0 {
			return fail
		}

		return result
	}

	return fail
}

func emitNodeValue(e *expression, node *Node) emitted {
	if e.Op == opCall {
		return emitRule(e.Text, node)
	}

	if e.Op == opAlt {
		for _, arg := range e.Args {
			result := emitNodeValue(arg, node)
			if result.ok {
				return result
			}
		}
	}

	return emitted{}
}

func emitFlag(e *expression) emitted {
	if e.Op == opLit {
		return emitted{tokens: []string{e.Text}, used: nil, ok: true}
	}

	if e.Op == opCall {
		if r, ok := loadGrammar().rules[e.Text]; ok {
			return emitFlag(r.Expr)
		}
	}

	if e.Op == opAlt {
		return emitFlag(e.Args[0])
	}

	return emitted{}
}

//nolint:cyclop,gocognit,gocyclo // Ветви CML проверяются conformance и corpus-тестами.
func emitScalar(e *expression, value ASTValue) emitted {
	var text string

	switch e.Op {
	case opLit:
		if value.Text != e.Text {
			return emitted{}
		}

		text = e.Text
	case opRef:
		text = emitID(value.Text)
	case opCall:
		switch e.Text {
		case symbolID:
			text = emitID(value.Text)
		case symbolSTRING:
			if !utf8.ValidString(value.Text) {
				return emitted{}
			}

			text = quoteString(value.Text)
		case symbolINT:
			text = value.Text
		default:
			r, ok := loadGrammar().rules[e.Text]
			if !ok {
				return emitted{}
			}

			if r.Kind == symbolEnum || e.Text == kindUserActivityDefaultVerb {
				return emitScalar(r.Expr, value)
			}

			if r.Kind == symbolDatatype {
				text = emitDatatype(e.Text, value.Text)
			} else {
				text = value.Text
			}
		}
	case opAlt:
		for _, arg := range e.Args {
			result := emitScalar(arg, value)
			if result.ok {
				return result
			}
		}

		return emitted{}
	default:
		return emitted{}
	}

	if text == "" {
		return emitted{}
	}

	return emitted{tokens: []string{text}, used: nil, ok: true}
}

func emitID(value string) string {
	if !validIdentifier(value) {
		return ""
	}

	if slices.Contains(loadGrammar().literals, value) || RussianAliases()[value] != "" {
		return "^" + value
	}

	return value
}

func quoteString(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"\"", "\\\"",
		"\n", "\\n",
		"\r", "\\r",
		"\t", "\\t",
		"\b", "\\b",
		"\f", "\\f",
	)

	return "\"" + replacer.Replace(value) + "\""
}

func acceptsKind(name, kind string, seen map[string]bool) bool {
	if name == kind {
		return true
	}

	if seen[name] {
		return false
	}

	seen[name] = true

	r, ok := loadGrammar().rules[name]

	if !ok || r.Kind != fieldRule {
		return false
	}

	return dispatchesKind(r.Expr, kind, seen)
}

func hasAssignment(e *expression) bool {
	if e.Op == opAssign {
		return true
	}

	return slices.ContainsFunc(e.Args, hasAssignment)
}

//nolint:gocognit // Ветви CML проверяются conformance и corpus-тестами.
func dispatchesKind(e *expression, kind string, seen map[string]bool) bool {
	if e.Op == opAlt {
		for _, a := range e.Args {
			if dispatchesKind(a, kind, cloneSeen(seen)) {
				return true
			}
		}

		return false
	}

	if hasAssignment(e) {
		return false
	}

	if e.Op == opCall {
		return acceptsKind(e.Text, kind, seen)
	}

	for _, a := range e.Args {
		if dispatchesKind(a, kind, cloneSeen(seen)) {
			return true
		}
	}

	return false
}

func cloneSeen(seen map[string]bool) map[string]bool {
	result := make(map[string]bool, len(seen))
	maps.Copy(result, seen)

	return result
}
