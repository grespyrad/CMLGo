package cmlgo_test

import (
	"os"
	"strings"
	"testing"

	cml "github.com/grespyrad/CMLGo"
)

// TestPublishedExamples проверяет модели из руководства как внешний потребитель API.
func TestPublishedExamples(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"service-ru", "service-en", "etalon-service", "etalon-service-neutral"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assertPublishedModel(t, "examples/"+name+".cml")
		})
	}
}

func assertExampleDiagrams(t *testing.T, result cml.Result, want int) {
	t.Helper()

	diagrams, err := cml.PlantUML(result)
	if err != nil {
		t.Fatal(err)
	}

	notes := 0

	for _, diagram := range diagrams {
		notes += strings.Count(diagram, "note as invariant_")
	}

	if notes < want {
		t.Fatalf("missing invariant scopes in diagrams: %d", notes)
	}
}

func assertPublishedModel(t *testing.T, path string) {
	t.Helper()

	//nolint:gosec // Имя выбирается из фиксированного списка собственных примеров.
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	result := cml.Validate(path, source)
	if !result.Valid || len(result.Diagnostics) != 0 {
		t.Fatalf("%+v", result.Diagnostics)
	}

	assertRoundTrip(t, source)

	if cml.CheckSyntaxWithDialect(path, source, cml.Dialect{Original: true, Aliases: nil}).Valid {
		t.Fatal("Original accepted Invariant")
	}

	notes := 4

	if strings.Contains(path, "etalon-") {
		notes = 3
	}

	assertExampleDiagrams(t, result, notes)
}
