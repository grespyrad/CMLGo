package cmlgo_test

import (
	"strings"
	"testing"

	cml "github.com/grespyrad/CMLGo"
)

// TestPlantUMLReference проверяет контракт CMLGo.
func TestPlantUMLReference(t *testing.T) {
	t.Parallel()

	r := cml.ValidateFile("testdata/reference/service.cml")

	files, err := cml.PlantUML(r)
	if err != nil {
		t.Fatal(err)
	}

	all := make([]string, 0, len(files))

	for name, source := range files {
		if !strings.HasSuffix(name, ".puml") || !strings.HasPrefix(source, "@startuml\n") ||
			!strings.HasSuffix(source, "@enduml\n") {
			t.Fatalf("invalid diagram %s", name)
		}

		all = append(all, source)
	}

	for _, want := range []string{
		"ServiceIntegration", "ExternalRecord", "Command", "DomainEvent",
		"Request", "Observation", "Обрабатывать запросы",
	} {
		if !strings.Contains(strings.Join(all, ""), want) {
			t.Fatalf("missing %q", want)
		}
	}
}

// TestPlantUMLRejectsInvalid проверяет контракт CMLGo.
func TestPlantUMLRejectsInvalid(t *testing.T) {
	t.Parallel()

	if _, err := cml.PlantUML(cml.Validate("bad.cml", []byte("bad"))); err == nil {
		t.Fatal("invalid model generated")
	}
}
