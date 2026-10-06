package cmlgo_test

import (
	"io/fs"
	"os"
	"strings"
	"testing"

	cml "github.com/grespyrad/CMLGo"
)

// TestRecognizerParity проверяет одинаковые решения parser и compact recognizer.
//
//nolint:gocognit // Corpus сравнивает два независимых прохода для каждого CML fixture.
func TestRecognizerParity(t *testing.T) {
	t.Parallel()

	for _, directory := range []string{"testdata/upstream", "testdata/conformance"} {
		fixtureFS := os.DirFS(directory)

		err := fs.WalkDir(fixtureFS, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() || !strings.HasSuffix(path, ".cml") {
				return nil
			}

			t.Run(directory+"/"+path, func(t *testing.T) {
				t.Parallel()

				data, readErr := fs.ReadFile(fixtureFS, path)
				if readErr != nil {
					t.Fatal(readErr)
				}

				syntax := cml.CheckSyntax(path, data)
				ast := cml.Parse(path, data)

				if syntax.Valid != ast.Valid {
					t.Fatalf("syntax=%v ast=%v", syntax.Valid, ast.Valid)
				}
			})

			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
