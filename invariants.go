package cmlgo

import "strings"

//nolint:cyclop,gocognit,gocyclo // Scope, непустые правила и уникальные trace links проверяются независимо.
func (v *validator) invariants(n *Node) {
	if n.Kind != kindInvariant {
		return
	}

	if strings.TrimSpace(n.Text(fieldRule)) == "" {
		v.reportError(n, "invariant-rule", "Invariant rule must not be blank")
	}

	if n.parent != nil {
		for _, other := range n.parent.Children(fieldInvariants) {
			if other != n && other.Name() == n.Name() {
				v.reportError(n, "invariant-name", "Invariant name must be unique in its owner")

				break
			}
		}
	}

	for _, field := range []string{fieldTestedBy, fieldImplementedBy} {
		values := n.Fields[field]
		if len(values) == 0 {
			v.warn(n, "invariant-trace", "Invariant has no "+field+" trace link")
		}

		seen := map[string]bool{}

		for _, value := range values {
			text := strings.TrimSpace(value.Text)
			if text == "" {
				v.reportError(n, "invariant-link", "Invariant trace link must not be blank")
			}

			if seen[text] {
				v.reportError(n, "invariant-link", "Invariant trace links must be unique")
			}

			seen[text] = true
		}
	}
}

func invariantNotes(b *textBuffer, bc *Node, aliases map[*Node]string) {
	for _, n := range descendants(bc) {
		if n.Kind != kindInvariant || n.parent == nil {
			continue
		}

		b.formatf(
			"note as invariant_%s\n%s.%s\\n%s\nend note\n",
			aliases[n],
			pumlText(n.parent.Name()),
			pumlText(n.Name()),
			pumlText(n.Text(fieldRule)),
		)
	}
}
