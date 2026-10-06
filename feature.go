package cmlgo

//nolint:cyclop,gocyclo // Lookahead отличает comma attribute list от следующего Feature.
func featureAttributeBoundary(e *expression, tokens []token, m match) bool {
	if !m.ok || m.end >= len(tokens) || e.Op != opSeq || len(e.Args) != 2 {
		return false
	}

	if e.Args[0].Op != opLit || e.Args[0].Text != "," || e.Args[1].Op != opAssign ||
		e.Args[1].Text != fieldEntityAttributes {
		return false
	}

	next := tokens[m.end]

	return next.kind == symbolString || next.kind == opLit && inSet(next.text, "a an the")
}
