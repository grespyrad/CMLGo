package cmlgo_test

import (
	"fmt"

	cml "github.com/grespyrad/CMLGo"
)

// ExampleValidate показывает проверку декларативного инварианта.
func ExampleValidate() {
	source := []byte(`BoundedContext ServiceIntegration {
  Aggregate Requests {
   Invariant SingleOwner {
    rule "Состояние запроса изменяет владелец агрегата"
    testedBy "example.contracts.TestSingleOwner"
    implementedBy "example.requests.Handle"
   }
  }
 }`)
	result := cml.Validate("service.cml", source)
	fmt.Println(result.Valid, len(result.Diagnostics))

	// Output: true 0
}

// ExampleUnmarshal показывает чтение и повторную сериализацию типизированной модели.
func ExampleUnmarshal() {
	source := []byte(`BoundedContext ServiceIntegration {
  Aggregate Requests { Entity Request { aggregateRoot String requestId } }
 }`)

	var model cml.ContextMappingModel

	if err := cml.Unmarshal(source, &model); err != nil {
		fmt.Println(err)

		return
	}

	fmt.Println(model.BoundedContexts[0].Aggregates[0].DomainObjects[0].Entity.Name)

	encoded, err := cml.Marshal(&model)
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Println(cml.Validate("encoded.cml", encoded).Valid)

	// Output:
	// Request
	// true
}

// ExampleMarshalWithDialect показывает вывод DDD aliases без перевода имени модели.
func ExampleMarshalWithDialect() {
	var model cml.ContextMappingModel

	if err := cml.Unmarshal([]byte("BoundedContext ServiceIntegration"), &model); err != nil {
		fmt.Println(err)

		return
	}

	encoded, err := cml.MarshalWithDialect(&model, cml.Dialect{
		Aliases: cml.RussianAliases(), Original: false,
	})
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Print(string(encoded))

	// Output: ОграниченныйКонтекст ServiceIntegration
}
