package cmlgo_test

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"testing"

	cml "github.com/grespyrad/CMLGo"
)

// TestTypedCorpusRoundTrip проверяет контракт CMLGo.
func TestTypedCorpusRoundTrip(t *testing.T) {
	t.Parallel()

	err := fs.WalkDir(os.DirFS("testdata/upstream"), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		t.Run(path, func(t *testing.T) {
			t.Parallel()

			//nolint:gosec // path получен из собственного fixture corpus, а не из внешнего ввода.
			source, e := os.ReadFile("testdata/upstream/" + path)
			if e != nil {
				t.Fatal(e)
			}

			assertRoundTrip(t, source)
		})

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestTokenizePreservesUnicodeAndComments проверяет контракт CMLGo.
//
//nolint:gosmopolitan // Unicode контракт требует Han и emoji в fixture.
func TestTokenizePreservesUnicodeAndComments(t *testing.T) {
	t.Parallel()

	source := []byte(
		"// Сервис 文 🔧\nBoundedContext ServiceIntegration { domainVisionStatement \"Сервис\\n文 🔧\" /* Keep\n  me */ }",
	)

	tokens, err := cml.Tokenize("test.cml", source)
	if err != nil {
		t.Fatal(err)
	}

	if tokens[0].Text != "// Сервис 文 🔧" || tokens[6].Text != "/* Keep\n  me */" {
		t.Fatalf("%+v", tokens)
	}
}

func assertRoundTrip(t *testing.T, source []byte) {
	t.Helper()

	var model cml.ContextMappingModel

	if decodeErr := cml.Unmarshal(source, &model); decodeErr != nil {
		t.Fatal(decodeErr)
	}

	data, e := cml.Marshal(&model)
	if e != nil {
		t.Fatal(e)
	}

	var again cml.ContextMappingModel

	if decodeErr := cml.Unmarshal(data, &again); decodeErr != nil {
		t.Fatal(decodeErr)
	}

	first, e := json.Marshal(model)
	if e != nil {
		t.Fatal(e)
	}

	second, e := json.Marshal(again)
	if e != nil {
		t.Fatal(e)
	}

	if !bytes.Equal(first, second) {
		t.Fatal("round-trip data differ")
	}
}
