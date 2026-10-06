//nolint:cyclop // Средняя сложность corpus и negative scenarios отражает независимые проверки контрактов.
package cmlgo_test

import (
	"os"
	"testing"

	cml "github.com/grespyrad/CMLGo"
)

// TestReferenceModels проверяет контракт CMLGo.
func TestReferenceModels(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"service", "service-before"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := cml.ValidateFile("testdata/reference/" + name + ".cml")
			if !result.Valid {
				t.Fatalf("%+v", result.Diagnostics)
			}
		})
	}
}

// TestValidationContract проверяет контракт CMLGo.
//
//nolint:gosmopolitan // Unicode контракт требует Han и emoji в fixture.
func TestValidationContract(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name, source string
		valid        bool
	}{
		{"empty", "", true},
		{
			"unicode",
			"// Сервис 文 🔧\nBoundedContext ServiceIntegration { domainVisionStatement = \"Сервис 🔧 文\" }",
			true,
		},
		{"escaped identifier", "BoundedContext ^type", true},
		{"missing brace", "BoundedContext ServiceIntegration {", false},
		{"unknown syntax", "BoundedContext ServiceIntegration { invariant X }", false},
		{"trailing garbage", "BoundedContext ServiceIntegration ???", false},
		{"unterminated string", "BoundedContext ServiceIntegration { domainVisionStatement \"Сервис }", false},
		{"missing context", "ContextMap { contains Missing }", false},
		{"duplicate context", "BoundedContext ServiceIntegration BoundedContext ServiceIntegration", false},
		{
			"multiple roots",
			"BoundedContext ServiceIntegration { Aggregate A { " +
				"Entity One { aggregateRoot } Entity Two { aggregateRoot } } }",
			false,
		},
		{
			"missing type",
			"BoundedContext ServiceIntegration { Aggregate A { Entity One { - Missing missing } } }",
			false,
		},
		{"self relationship", "ContextMap { contains A -> A } BoundedContext", false},
		{"map membership", "ContextMap { contains A -> B } BoundedContext A BoundedContext", false},
		{"roles", "ContextMap { contains A,B A [S,OHS] -> [C] B } BoundedContext A BoundedContext B", false},
		{"team owner", "BoundedContext A { Aggregate X { owner A } }", false},
		{"invalid utf8", string([]byte{0xff}), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := cml.Validate("test.cml", []byte(tc.source))
			if result.Valid != tc.valid {
				t.Fatalf("valid=%v want %v: %+v", result.Valid, tc.valid, result.Diagnostics)
			}

			if !tc.valid && len(result.Diagnostics) == 0 {
				t.Fatal("missing diagnostic")
			}
		})
	}
}

// TestImports проверяет контракт CMLGo.
func TestImports(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	for name, source := range map[string]string{
		"map.cml":      "import \"contexts.cml\" ContextMap { contains A,B A -> B }",
		"contexts.cml": "import \"map.cml\" BoundedContext A BoundedContext B",
	} {
		if err := os.WriteFile(dir+"/"+name, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	if r := cml.ValidateFile(dir + "/map.cml"); !r.Valid {
		t.Fatalf("%+v", r.Diagnostics)
	}

	if err := os.Remove(dir + "/contexts.cml"); err != nil {
		t.Fatal(err)
	}

	if r := cml.ValidateFile(dir + "/map.cml"); r.Valid {
		t.Fatal("missing import accepted")
	}
}

// FuzzParse проверяет контракт CMLGo.
//
//nolint:gosmopolitan // Unicode контракт требует Han и emoji в fixture.
func FuzzParse(f *testing.F) {
	f.Add("BoundedContext ServiceIntegration { Aggregate A { Entity One { aggregateRoot String id } } }")
	f.Add("Сервис 文 🔧")
	f.Fuzz(func(t *testing.T, source string) {
		if len(source) < 65536 {
			cml.Validate("fuzz.cml", []byte(source))
		}
	})
}

// BenchmarkValidateServiceIntegration проверяет контракт CMLGo.
func BenchmarkValidateServiceIntegration(b *testing.B) {
	data, err := os.ReadFile("testdata/reference/service.cml")
	if err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()

	for b.Loop() {
		if r := cml.Validate("service.cml", data); !r.Valid {
			b.Fatal(r.Diagnostics)
		}
	}
}
