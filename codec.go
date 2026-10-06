package cmlgo

import (
	"errors"
	"fmt"
	"reflect"
)

// Ref задаёт типизированную ссылку CML по имени. Параметр T обозначает допустимый тип цели.
type Ref[T any] struct {
	Name string `json:"name"`
}

// ErrInvalidModel обозначает невалидную модель или неподдерживаемый вариант union.
var ErrInvalidModel = errors.New("invalid CML model")

// Unmarshal разбирает исходный CML непосредственно в типизированные Go-объекты.
// Семантическая проверка и загрузка импортов выполняются отдельно через ValidateFile.
func Unmarshal(source []byte, model *ContextMappingModel) error {
	return UnmarshalWithDialect(source, model, Dialect{Aliases: nil, Original: false})
}

// UnmarshalWithDialect заполняет typed model с выбранным словарём keywords.
func UnmarshalWithDialect(source []byte, model *ContextMappingModel, dialect Dialect) error {
	if model == nil {
		return fmt.Errorf("unmarshal CML: nil destination: %w", ErrInvalidModel)
	}

	result := ParseWithDialect("input.cml", source, dialect)
	if !result.Valid {
		return fmt.Errorf("unmarshal CML: %s: %w", result.Diagnostics[0].Message, ErrInvalidModel)
	}

	target := reflect.ValueOf(model).Elem()
	target.SetZero()

	if err := projectNode(result.Model, target); err != nil {
		return fmt.Errorf("unmarshal CML: %w", err)
	}

	return nil
}

// Marshal сериализует типизированные Go-объекты в исходный CML 6.12.0.
func Marshal(model *ContextMappingModel) ([]byte, error) {
	if model == nil {
		return nil, fmt.Errorf("marshal CML: nil model: %w", ErrInvalidModel)
	}

	node, err := encodeNode(reflect.ValueOf(model).Elem())
	if err != nil {
		return nil, fmt.Errorf("marshal CML: %w", err)
	}

	return marshalNode(node)
}

//nolint:cyclop,gocognit,gocyclo // Ветви CML проверяются conformance и corpus-тестами.
func projectNode(node *Node, target reflect.Value) error {
	typ := target.Type()
	target.FieldByName("Position").Set(reflect.ValueOf(node.Position))
	target.FieldByName("Raw").Set(reflect.ValueOf(node))

	own := typedKind(typ)
	if own != node.Kind {
		for i := range typ.NumField() {
			field := typ.Field(i)
			if field.Tag.Get("cmlVariant") == node.Kind {
				pointer := reflect.New(field.Type.Elem())
				if err := projectNode(node, pointer.Elem()); err != nil {
					return err
				}

				target.Field(i).Set(pointer)

				return nil
			}
		}

		return fmt.Errorf("cannot decode %s into %s: %w", node.Kind, typ, ErrInvalidModel)
	}

	for i := range typ.NumField() {
		field := typ.Field(i)
		feature := field.Tag.Get("cml")

		if feature == "" || feature == "-" {
			continue
		}

		values := node.Fields[feature]
		if len(values) == 0 {
			continue
		}

		dest := target.Field(i)
		if dest.Kind() == reflect.Slice {
			slice := reflect.MakeSlice(dest.Type(), len(values), len(values))
			for j, value := range values {
				if err := projectValue(value, slice.Index(j)); err != nil {
					return err
				}
			}

			dest.Set(slice)
		} else {
			if err := projectValue(values[0], dest); err != nil {
				return err
			}
		}
	}

	return nil
}

//nolint:exhaustive // Схема допускает string/bool; default отклоняет остальные reflect.Kind.
func projectValue(value ASTValue, target reflect.Value) error {
	if target.Kind() == reflect.Pointer {
		pointer := reflect.New(target.Type().Elem())

		switch {
		case value.RefType != "":
			pointer.Elem().FieldByName("Name").SetString(value.Text)
		case value.Node != nil:
			if err := projectNode(value.Node, pointer.Elem()); err != nil {
				return err
			}
		default:
			return fmt.Errorf("missing node for %s: %w", target.Type(), ErrInvalidModel)
		}

		target.Set(pointer)

		return nil
	}

	switch target.Kind() {
	case reflect.String:
		target.SetString(value.Text)
	case reflect.Bool:
		target.SetBool(value.Text == symbolTrue)
	default:
		return fmt.Errorf("unsupported scalar %s: %w", target.Type(), ErrInvalidModel)
	}

	return nil
}

//nolint:cyclop,funlen,gocognit,gocyclo // Ветви CML проверяются conformance и corpus-тестами.
func encodeNode(source reflect.Value) (*Node, error) {
	typ := source.Type()
	kind := typedKind(typ)

	var selected *Node

	ownFields := false
	node := &Node{Kind: kind, Fields: map[string][]ASTValue{}, Position: Position{}, parent: nil}

	for i := range typ.NumField() {
		field := typ.Field(i)
		value := source.Field(i)

		if variant := field.Tag.Get("cmlVariant"); variant != "" && !value.IsNil() {
			if selected != nil {
				return nil, fmt.Errorf("multiple variants of %s: %w", kind, ErrInvalidModel)
			}

			n, err := encodeNode(value.Elem())
			if err != nil {
				return nil, err
			}

			selected = n

			continue
		}

		feature := field.Tag.Get("cml")
		if feature == "" || feature == "-" ||
			value.IsZero() && (value.Kind() != reflect.String || !originalField(source, feature)) {
			continue
		}

		ownFields = true

		if value.Kind() == reflect.Slice {
			for j := range value.Len() {
				encoded, err := encodeValue(value.Index(j))
				if err != nil {
					return nil, err
				}

				node.Fields[feature] = append(node.Fields[feature], encoded)
			}
		} else {
			encoded, err := encodeValue(value)
			if err != nil {
				return nil, err
			}

			node.Fields[feature] = []ASTValue{encoded}
		}
	}

	if selected != nil {
		if ownFields {
			return nil, fmt.Errorf("union %s mixes variant and own fields: %w", kind, ErrInvalidModel)
		}

		return selected, nil
	}

	return node, nil
}

//nolint:exhaustive // Схема допускает string/bool; default отклоняет остальные reflect.Kind.
func encodeValue(source reflect.Value) (ASTValue, error) {
	if source.Kind() == reflect.Pointer {
		if source.IsNil() {
			return ASTValue{}, fmt.Errorf("nil model element: %w", ErrInvalidModel)
		}

		if stringsRef(source.Type().Elem()) {
			return ASTValue{
				Text:     source.Elem().FieldByName("Name").String(),
				RefType:  "reference",
				Node:     nil,
				Position: Position{},
			}, nil
		}

		node, err := encodeNode(source.Elem())

		return ASTValue{Node: node, Text: "", RefType: "", Position: Position{}}, err
	}

	switch source.Kind() {
	case reflect.String:
		return ASTValue{Text: source.String(), Node: nil, RefType: "", Position: Position{}}, nil
	case reflect.Bool:
		return ASTValue{Text: symbolTrue, Node: nil, RefType: "", Position: Position{}}, nil
	default:
		return ASTValue{}, fmt.Errorf("unsupported value %s: %w", source.Type(), ErrInvalidModel)
	}
}
func stringsRef(t reflect.Type) bool { return t.NumField() == 1 && t.Field(0).Name == "Name" }

func originalField(source reflect.Value, feature string) bool {
	raw := source.FieldByName("Raw")
	if raw.IsNil() {
		return false
	}

	node, ok := reflect.TypeAssert[*Node](raw)

	return ok && len(node.Fields[feature]) > 0
}

func typedKind(t reflect.Type) string {
	model, ok := reflect.TypeAssert[interface{ cmlKind() string }](reflect.New(t))
	if !ok {
		return ""
	}

	return model.cmlKind()
}
