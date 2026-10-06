package cmlgo

import "fmt"

// Lexeme содержит исходный токен без изменения текста строк и комментариев.
type Lexeme struct {
	Kind     string   `json:"kind"`
	Text     string   `json:"text"`
	Position Position `json:"position"`
}

// Tokenize возвращает токены CML вместе с комментариями; whitespace исключается.
// Метод не проверяет синтаксическую или семантическую структуру модели.
func Tokenize(file string, source []byte) ([]Lexeme, error) {
	return TokenizeWithDialect(file, source, Dialect{Aliases: nil, Original: false})
}

// TokenizeWithDialect сохраняет raw tokens и позиции выбранного диалекта.
func TokenizeWithDialect(file string, source []byte, dialect Dialect) ([]Lexeme, error) {
	tokens, issue := lexWithDialect(file, source, grammarForDialect(dialect).literals, true, dialect)
	if issue != nil {
		return nil, fmt.Errorf(
			"tokenize CML at %d:%d: %s: %w",
			issue.Position.Line,
			issue.Position.Column,
			issue.Message,
			ErrInvalidModel,
		)
	}

	out := make([]Lexeme, 0, len(tokens)-1)
	for _, token := range tokens {
		if token.kind != symbolEof {
			out = append(out, Lexeme{Kind: token.kind, Text: token.raw, Position: token.pos})
		}
	}

	return out, nil
}
