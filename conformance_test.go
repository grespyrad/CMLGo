package cmlgo_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	cml "github.com/grespyrad/CMLGo"
)

// TestOfficialConformance проверяет решения полного Xtext validator 6.12.0.
func TestOfficialConformance(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("testdata/conformance/expected.json")
	if err != nil {
		t.Fatal(err)
	}

	var corpus []struct {
		File  string `json:"file"`
		Valid bool   `json:"valid"`
	}

	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}

	for _, fixture := range corpus {
		t.Run(fixture.File, func(t *testing.T) {
			t.Parallel()

			result := cml.ValidateFile(filepath.Join("testdata", "conformance", fixture.File))
			if result.Valid != fixture.Valid {
				t.Fatalf("valid=%v expected=%v: %+v", result.Valid, fixture.Valid, result.Diagnostics)
			}
		})
	}
}

// TestOperationWithoutReturnType проверяет необходимый grammar lookahead.
func TestOperationWithoutReturnType(t *testing.T) {
	t.Parallel()

	source := []byte(
		"BoundedContext A { Service StoreService { store(); void ^delete(); @E get(); } " +
			"Aggregate Items { Entity E { aggregateRoot } } }",
	)
	if result := cml.Validate("test.cml", source); !result.Valid {
		t.Fatal(result.Diagnostics)
	}
}
