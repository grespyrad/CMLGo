package cmlgo

import "strings"

func emitDatatype(name, value string) string {
	if name == kindType && containsLiteral(loadGrammar().rules[kindType].Expr, value) {
		return value
	}

	separators := "."

	switch name {
	case kindChannelIdentifier:
		separators = "./:"
	case kindThrowsIdentifier:
		separators = ".,"
	default:
	}

	parts := make([]string, 0, 2*len(value)+1)

	start := 0

	for i, r := range value {
		if !strings.ContainsRune(separators, r) {
			continue
		}

		part := emitID(value[start:i])
		if part == "" {
			return ""
		}

		parts = append(parts, part, string(r))
		start = i + 1
	}

	last := emitID(value[start:])
	if last == "" {
		return ""
	}

	parts = append(parts, last)

	return strings.Join(parts, "")
}

func containsLiteral(e *expression, text string) bool {
	if e.Op == opLit && e.Text == text {
		return true
	}

	for _, arg := range e.Args {
		if containsLiteral(arg, text) {
			return true
		}
	}

	return false
}
