//nolint:cyclop // Последовательный round-trip и negative scenarios проверяют независимые ошибки.
package cmlgo_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	cml "github.com/grespyrad/CMLGo"
)

//nolint:gosmopolitan // Fixture включает CJK для проверки Unicode rule.
const invariantSource = `BoundedContext ServiceIntegration {
 Aggregate Requests {
  Invariant SingleOwner {
   rule "Один writer 🔧 文"
   rationale "Согласованность"
   testedBy "example.contracts.TestOwner", "example.contracts.TestRestore"
   implementedBy "example.requests.Handle"
  }
  Entity Request { aggregateRoot int value
   Invariant ValueRange {
    rule "0 <= value <= 100"
    testedBy "example.contracts.TestValue"
    implementedBy "example.requests.SetValue"
   }
  }
 }
}`

// TestInvariantRoundTrip проверяет codec, recognition и diagrams расширения.
//
//nolint:cyclop,gocognit,gocyclo,gosmopolitan // Последовательный typed round-trip с Unicode и двумя областями.
func TestInvariantRoundTrip(t *testing.T) {
	t.Parallel()

	result := cml.Validate("invariants.cml", []byte(invariantSource))
	if !result.Valid {
		t.Fatalf("%+v", result.Diagnostics)
	}

	if !cml.CheckSyntax("invariants.cml", []byte(invariantSource)).Valid {
		t.Fatal("recognizer differs")
	}

	var model cml.ContextMappingModel

	if err := cml.Unmarshal([]byte(invariantSource), &model); err != nil {
		t.Fatal(err)
	}

	invariant := model.BoundedContexts[0].Aggregates[0].Invariants[0]
	if invariant.Name != "SingleOwner" || len(invariant.TestedBy) != 2 || invariant.Rule != "Один writer 🔧 文" {
		t.Fatalf("%+v", invariant)
	}

	encoded, err := cml.Marshal(&model)
	if err != nil {
		t.Fatal(err)
	}

	var decoded cml.ContextMappingModel

	if decodeErr := cml.Unmarshal(encoded, &decoded); decodeErr != nil {
		t.Fatal(decodeErr)
	}

	before, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}

	after, err := json.Marshal(decoded)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(before, after) {
		t.Fatal("round trip changed model")
	}

	diagrams, err := cml.PlantUML(result)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(diagrams["ServiceIntegration.puml"], "SingleOwner") {
		t.Fatal("missing invariant diagram")
	}
}

// TestInvariantInvalid проверяет scope и trace validation.
func TestInvariantInvalid(t *testing.T) {
	t.Parallel()

	for name, body := range map[string]string{
		"blank":          `Invariant A {rule " "}`,
		"duplicate":      `Invariant A {rule "a"} Invariant A {rule "b"}`,
		"blank-link":     `Invariant A {rule "a" testedBy ""}`,
		"duplicate-link": `Invariant A {rule "a" testedBy "T", "T"}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if cml.Validate(name, []byte("BoundedContext B {Aggregate A {"+body+"}}")).Valid {
				t.Fatal("invalid invariant accepted")
			}
		})
	}

	result := cml.Validate(
		"warning",
		[]byte(`BoundedContext B {Invariant A {rule "r"} Aggregate A {Invariant A {rule "r"}}}`),
	)
	if !result.Valid || len(result.Diagnostics) != 4 {
		t.Fatalf("%+v", result)
	}
}

// TestDialect проверяет aliases, Unicode names и original grammar.
//
//nolint:cyclop,gocognit,gocyclo // Один сценарий проверяет model round-trip и независимость dialect конфигурации.
func TestDialect(t *testing.T) {
	t.Parallel()

	source := []byte(
		`ОграниченныйКонтекст Сервис {
 domainVisionStatement "Сервис 🔧"
 Агрегат Запросы { Сущность Запрос { aggregateRoot int значение
  Invariant Значение { rule "0..100" testedBy "example.contracts.TestValue" implementedBy "example.requests.SetValue" }
 } }
}`,
	)

	var model cml.ContextMappingModel

	if err := cml.Unmarshal(source, &model); err != nil {
		t.Fatal(err)
	}

	if !cml.Validate("ru.cml", source).Valid {
		t.Fatal("Russian model invalid")
	}

	encoded, err := cml.MarshalWithDialect(&model, cml.Dialect{Aliases: cml.RussianAliases(), Original: false})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(encoded), "ОграниченныйКонтекст") {
		t.Fatal("not translated")
	}

	var decoded cml.ContextMappingModel

	if decodeErr := cml.Unmarshal(encoded, &decoded); decodeErr != nil {
		t.Fatal(decodeErr)
	}

	before, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}

	after, err := json.Marshal(decoded)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(before, after) {
		t.Fatal("Russian round trip changed data")
	}

	dialect := cml.Dialect{Aliases: map[string]string{"Contexto": "BoundedContext"}, Original: false}
	if !cml.ParseWithDialect("es", []byte("Contexto ServiceIntegration"), dialect).Valid {
		t.Fatal("custom aliases")
	}

	if cml.Parse("es", []byte("Contexto ServiceIntegration")).Valid {
		t.Fatal("global alias mutation")
	}

	if cml.ParseWithDialect("original", source, cml.Dialect{Original: true, Aliases: nil}).Valid {
		t.Fatal("original accepted extension")
	}

	if !cml.ParseWithDialect(
		"original",
		[]byte("BoundedContext ServiceIntegration"),
		cml.Dialect{Original: true, Aliases: nil},
	).Valid {
		t.Fatal("original rejected upstream")
	}

	if !cml.Parse(
		"soft",
		[]byte("BoundedContext Invariant {Aggregate rule {Entity testedBy {String implementedBy}}}"),
	).Valid {
		t.Fatal("new keywords reserved old names")
	}
}

// TestDialectFailures проверяет conflicts и сохранение имён, совпавших с soft keyword.
//
//nolint:gocognit,gocyclo // Отрицательные словари и round-trip проверяются в одном integration scenario.
func TestDialectFailures(t *testing.T) {
	t.Parallel()

	for name, dialect := range map[string]cml.Dialect{
		"unknown":  {Aliases: map[string]string{"Контекст": "Unknown"}, Original: false},
		"conflict": {Aliases: map[string]string{"Entity": "Aggregate"}, Original: false},
		"original": {Aliases: map[string]string{"Контекст": "BoundedContext"}, Original: true},
		"invalid":  {Aliases: map[string]string{"Контекст с пробелом": "BoundedContext"}, Original: false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if cml.ParseWithDialect(name, []byte("BoundedContext ServiceIntegration"), dialect).Valid {
				t.Fatal("invalid dictionary accepted")
			}
		})
	}

	var model cml.ContextMappingModel

	if err := cml.Unmarshal(
		[]byte(`BoundedContext rule {Aggregate A {Invariant rule {rule "rule"}}}`),
		&model,
	); err != nil {
		t.Fatal(err)
	}

	dialect := cml.Dialect{Aliases: map[string]string{"Инвариант": "Invariant", "Правило": "rule"}, Original: false}

	encoded, err := cml.MarshalWithDialect(&model, dialect)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(encoded), `BoundedContext rule`) ||
		!strings.Contains(string(encoded), `Инвариант rule`) {
		t.Fatal("identifier was translated")
	}

	var decoded cml.ContextMappingModel

	if decodeErr := cml.UnmarshalWithDialect(encoded, &decoded, dialect); decodeErr != nil {
		t.Fatal(decodeErr)
	}

	if decoded.BoundedContexts[0].Aggregates[0].Invariants[0].Rule != "rule" {
		t.Fatal("string translated")
	}

	if _, originalErr := cml.MarshalWithDialect(&model, cml.Dialect{Original: true, Aliases: nil}); originalErr == nil {
		t.Fatal("original serialized extension")
	}

	aliases := cml.RussianAliases()

	aliases["Сущность"] = "Service"

	if cml.RussianAliases()["Сущность"] != "Entity" {
		t.Fatal("global dictionary mutation")
	}
}

// TestInvariantScopes проверяет каждый допустимый owner и Unicode cross references.
func TestInvariantScopes(t *testing.T) {
	t.Parallel()

	source := []byte(`BoundedContext Сервис {
 Invariant Контекст {rule "r"}
 Aggregate АгрегатЗапросов {Invariant ^Агрегат {rule "r"}
 Entity Сервис {aggregateRoot - Параметры параметры Invariant ^Сущность {rule "r"}}
 ValueObject Параметры {String значение Invariant Значение {rule "r"}}
 } }`)
	result := cml.Validate("scopes.cml", source)

	if !result.Valid {
		t.Fatalf("%+v", result.Diagnostics)
	}

	if cml.CheckSyntaxWithDialect("scopes.cml", source, cml.Dialect{Original: true, Aliases: nil}).Valid {
		t.Fatal("original accepted Unicode names")
	}
}
