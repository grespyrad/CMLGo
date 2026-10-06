package cmlgo

import (
	"fmt"
	"maps"
	"unicode"
)

// Dialect задаёт дополнительные alias -> canonical keyword и строгую грамматику 6.12.0.
// Карта читается в течение вызова; вызывающий не должен менять её одновременно.
type Dialect struct {
	Aliases  map[string]string
	Original bool
}

// RussianAliases возвращает независимую копию подтверждённых книжных DDD терминов.
// Источник и страницы: docs/books/SOURCES.md. Технические поля остаются английскими.
func RussianAliases() map[string]string {
	return map[string]string{
		"ОграниченныйКонтекст": "BoundedContext", "КартаКонтекстов": "ContextMap",
		"ПредметнаяОбласть": "Domain", "Поддомен": "Subdomain", "Агрегат": "Aggregate",
		"Сущность": "Entity", "ОбъектЗначение": "ValueObject", "Команда": "Command",
		"Событие": "Event",
	}
}

//nolint:cyclop,gocognit,gocyclo // Проверки aliases и сохранения names покрыты dialect round-trip tests.
func dialectAliases(dialect Dialect) (map[string]string, error) {
	if dialect.Original {
		if len(dialect.Aliases) > 0 {
			return nil, fmt.Errorf("original dialect cannot have aliases: %w", ErrInvalidModel)
		}

		return map[string]string{}, nil
	}

	aliases := RussianAliases()
	literals := loadGrammar().literals

	for alias, canonical := range dialect.Aliases {
		if !validIdentifier(alias) || alias == canonical {
			return nil, fmt.Errorf("invalid keyword alias %q: %w", alias, ErrInvalidModel)
		}

		found := extensionKeyword(canonical)

		for _, literal := range literals {
			if literal == canonical {
				found = true
			}

			if literal == alias && literal != canonical {
				return nil, fmt.Errorf("alias conflicts with %q: %w", alias, ErrInvalidModel)
			}
		}

		if !found {
			return nil, fmt.Errorf("unknown canonical keyword %q: %w", canonical, ErrInvalidModel)
		}

		if existing, ok := aliases[alias]; ok && existing != canonical {
			return nil, fmt.Errorf("alias conflict %q: %w", alias, ErrInvalidModel)
		}

		aliases[alias] = canonical
	}

	return aliases, nil
}

func extensionKeyword(text string) bool {
	switch text {
	case "Invariant", "rule", "rationale", "testedBy", "implementedBy":
		return true
	default:
		return false
	}
}

func matchesLiteral(token token, literal string) bool {
	return token.text == literal &&
		(token.kind == opLit || token.kind == symbolId && extensionKeyword(literal) && token.raw == literal)
}

func identifierStart(r rune, original bool) bool {
	return r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || !original && unicode.IsLetter(r)
}

func identifierPart(r rune, original bool) bool {
	return identifierStart(r, original) || r >= '0' && r <= '9' ||
		!original && (unicode.IsDigit(r) || unicode.IsMark(r))
}

// MarshalWithDialect выводит выбранные ключи, не переводя имена, строки и комментарии.
// Если несколько aliases соответствуют ключу, выбирается лексикографически первый.
//
//nolint:cyclop,gocognit,gocyclo // Проверки aliases и сохранения names покрыты dialect round-trip tests.
func MarshalWithDialect(model *ContextMappingModel, dialect Dialect) ([]byte, error) {
	if _, err := dialectAliases(dialect); err != nil {
		return nil, err
	}

	source, err := Marshal(model)
	if err != nil {
		return nil, err
	}

	if dialect.Original {
		if !ParseWithDialect("serialized.cml", source, dialect).Valid {
			return nil, fmt.Errorf("model uses CMLGo extensions: %w", ErrInvalidModel)
		}

		return source, nil
	}

	preferred := map[string]string{}

	for alias, canonical := range maps.Clone(dialect.Aliases) {
		if prior := preferred[canonical]; prior == "" || alias < prior {
			preferred[canonical] = alias
		}
	}

	tokens, err := Tokenize("serialized.cml", source)
	if err != nil {
		return nil, err
	}

	keywords := keywordPositions(source, tokens)

	var output []byte

	offset := 0

	for _, token := range tokens {
		replacement := preferred[token.Text]
		if replacement == "" || !keywords[token.Position.Offset] {
			continue
		}

		output = append(output, source[offset:token.Position.Offset]...)
		output = append(output, replacement...)
		offset = token.Position.Offset + len(token.Text)
	}

	output = append(output, source[offset:]...)
	if !ParseWithDialect("serialized.cml", output, dialect).Valid {
		return nil, fmt.Errorf("localized CML invalid: %w", ErrInvalidModel)
	}

	return output, nil
}

//nolint:gocognit // Проверки aliases и сохранения names покрыты dialect round-trip tests.
func keywordPositions(source []byte, tokens []Lexeme) map[int]bool {
	positions := map[int]bool{}
	indexes := map[int]int{}

	for i, token := range tokens {
		indexes[token.Position.Offset] = i
		if token.Kind == opLit {
			positions[token.Position.Offset] = true
		}
	}

	result := Parse("serialized.cml", source)
	for _, node := range descendants(result.Model) {
		if node.Kind != kindInvariant {
			continue
		}

		positions[node.Position.Offset] = true

		for _, field := range []string{fieldRule, fieldRationale, fieldTestedBy, fieldImplementedBy} {
			values := node.Fields[field]
			if len(values) == 0 {
				continue
			}

			index := indexes[values[0].Position.Offset] - 1
			if index >= 0 && tokens[index].Text == "=" {
				index--
			}

			if index >= 0 {
				positions[tokens[index].Position.Offset] = true
			}
		}
	}

	return positions
}
