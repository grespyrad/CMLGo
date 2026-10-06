package cmlgo

import (
	"cmp"
	"encoding/json"
	"slices"
	"strings"
	"sync"

	"github.com/grespyrad/CMLGo/internal/grammar"
)

//nolint:gochecknoglobals // sync.Once публикует неизменяемый grammar cache.
var (
	grammarOnce      sync.Once
	compiled         grammarState
	originalOnce     sync.Once
	originalCompiled grammarState
)

//nolint:gocognit // Ветви CML проверяются conformance и corpus-тестами.
func loadGrammar() *grammarState {
	grammarOnce.Do(func() {
		compiled.err = json.Unmarshal(grammar.JSON, &compiled.rules)

		set := map[string]bool{}

		var walk func(*expression)

		walk = func(e *expression) {
			if e.Op == opLit {
				set[e.Text] = true
			}

			for _, a := range e.Args {
				walk(a)
			}
		}

		for _, r := range compiled.rules {
			walk(r.Expr)
		}

		for s := range set {
			if !extensionKeyword(s) {
				compiled.literals = append(compiled.literals, s)
			}
		}

		slices.SortFunc(compiled.literals, func(a, b string) int {
			if len(a) != len(b) {
				return cmp.Compare(len(b), len(a))
			}

			return strings.Compare(a, b)
		})
	})

	return &compiled
}

func grammarForDialect(dialect Dialect) *grammarState {
	if !dialect.Original {
		return loadGrammar()
	}

	originalOnce.Do(func() {
		originalCompiled.err = json.Unmarshal(grammar.OriginalJSON, &originalCompiled.rules)
		originalCompiled.literals = slices.Clone(loadGrammar().literals)
	})

	return &originalCompiled
}
