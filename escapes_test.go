package cmlgo_test

import (
	"testing"

	cml "github.com/grespyrad/CMLGo"
)

// TestUnicodeEscapes проверяет UTF-16 escapes из исходного CML.
//
//nolint:gosmopolitan // Unicode контракт включает Han и emoji.
func TestUnicodeEscapes(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ source, want string }{
		{
			`BoundedContext ServiceIntegration { domainVisionStatement "\u0421\u0435\u0440\u0432\u0438\u0441" }`,
			"Сервис",
		},
		{`BoundedContext ServiceIntegration { domainVisionStatement "\uD83D\uDD27 文" }`, "🔧 文"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()

			var model cml.ContextMappingModel

			if err := cml.Unmarshal([]byte(tc.source), &model); err != nil {
				t.Fatal(err)
			}

			if got := model.BoundedContexts[0].DomainVisionStatement; got != tc.want {
				t.Fatalf("%q", got)
			}
		})
	}
}
