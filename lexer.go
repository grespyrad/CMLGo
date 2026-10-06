package cmlgo

import (
	"strings"
	"unicode/utf8"
)

type token struct {
	kind, text, raw string
	pos             Position
}

//nolint:cyclop,funlen,gocognit,gocyclo,maintidx,nestif,revive // Ветви lexer проверены corpus и Unicode tests.
func lexWithDialect(
	file string,
	source []byte,
	literals []string,
	keepComments bool,
	dialect Dialect,
) ([]token, *Diagnostic) {
	pos := Position{File: file, Line: 1, Column: 1, Offset: 0}
	fail := func(code, msg string) ([]token, *Diagnostic) {
		return nil, &Diagnostic{Severity: symbolError, Code: code, Message: msg, Position: pos}
	}

	if !utf8.Valid(source) {
		return fail("utf8", "Source is not valid UTF-8")
	}

	if len(source) > 8<<20 {
		return fail(symbolLimit, "Source exceeds the 8 MiB limit")
	}

	aliases, aliasErr := dialectAliases(dialect)
	if aliasErr != nil {
		return fail("dialect", aliasErr.Error())
	}

	text := string(source)

	var out []token

	advance := func(s string) {
		for _, r := range s {
			if r == '\n' {
				pos.Line++

				pos.Column = 1
			} else {
				pos.Column++
			}
		}

		pos.Offset += len(s)
	}

	for pos.Offset < len(text) {
		rest := text[pos.Offset:]
		if strings.ContainsRune(" \t\r\n\f", rune(rest[0])) {
			advance(rest[:1])

			continue
		}

		if strings.HasPrefix(rest, "//") {
			n := strings.IndexByte(rest, '\n')
			if n < 0 {
				n = len(rest)
			}

			if keepComments {
				out = append(out, token{kind: fieldComment, text: rest[:n], raw: rest[:n], pos: pos})
			}

			advance(rest[:n])

			continue
		}

		if strings.HasPrefix(rest, "/*") {
			n := strings.Index(rest[2:], "*/")
			if n < 0 {
				return fail(symbolSyntax, "Unterminated block comment")
			}

			if keepComments {
				out = append(out, token{kind: fieldComment, text: rest[:n+4], raw: rest[:n+4], pos: pos})
			}

			advance(rest[:n+4])

			continue
		}

		start := pos

		if rest[0] == '\'' || rest[0] == '"' {
			quote := rest[0]

			var b []byte

			i := 1

			for i < len(rest) && rest[i] != quote {
				if rest[i] == '\\' {
					i++
					if i == len(rest) {
						return fail("syntax", "Unterminated escape")
					}

					switch rest[i] {
					case 'n':
						b = append(b, '\n')
					case 'r':
						b = append(b, '\r')
					case 't':
						b = append(b, '\t')
					case 'b':
						b = append(b, '\b')
					case 'f':
						b = append(b, '\f')
					case 'u':
						decoded, size, err := unicodeEscape(rest[i:])
						if err != nil {
							return fail(symbolSyntax, err.Error())
						}

						b = utf8.AppendRune(b, decoded)

						i += size - 1
					case '\\', '\'', '"':
						b = append(b, rest[i])
					default:
						return fail(symbolSyntax, "Invalid string escape")
					}

					i++
				} else {
					_, n := utf8.DecodeRuneInString(rest[i:])

					b = append(b, rest[i:i+n]...)

					i += n
				}
			}

			if i == len(rest) {
				return fail(symbolSyntax, "Unterminated string")
			}

			i++

			out = append(out, token{symbolString, string(b), rest[:i], start})
			advance(rest[:i])

			continue
		}

		if rest[0] == '^' {
			length := identifierLength(rest[1:], dialect.Original)
			if length > 0 {
				raw := rest[:length+1]

				out = append(out, token{symbolId, raw[1:], raw, start})
				advance(raw)

				continue
			}
		}

		literal := ""

		for _, s := range literals {
			if strings.HasPrefix(rest, s) && literalBoundary(rest, s, dialect.Original) {
				literal = s

				break
			}
		}

		if literal != "" {
			out = append(out, token{opLit, literal, literal, start})
			advance(literal)

			continue
		}

		if length := identifierLength(rest, dialect.Original); length > 0 {
			raw := rest[:length]
			kind, text := symbolId, raw

			if canonical, ok := aliases[raw]; ok {
				kind, text = opLit, canonical
			}

			out = append(out, token{kind, text, raw, start})
			advance(raw)

			continue
		}

		if rest[0] >= '0' && rest[0] <= '9' {
			i := 1
			for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
				i++
			}

			out = append(out, token{symbolInt, rest[:i], rest[:i], start})
			advance(rest[:i])

			continue
		}

		_, n := utf8.DecodeRuneInString(rest)

		out = append(out, token{symbolOther, rest[:n], rest[:n], start})
		advance(rest[:n])
	}

	out = append(out, token{kind: symbolEof, text: "", raw: "", pos: pos})

	return out, nil
}

//nolint:revive // Флаг выбирает исходные ASCII или расширенные Unicode identifiers.
func identifierLength(text string, original bool) int {
	length := 0

	for _, r := range text {
		if length == 0 && !identifierStart(r, original) || length > 0 && !identifierPart(r, original) {
			break
		}

		length += utf8.RuneLen(r)
	}

	return length
}

//nolint:revive // Граница keyword соответствует выбранному набору identifier runes.
func literalBoundary(rest, literal string, original bool) bool {
	last, _ := utf8.DecodeLastRuneInString(literal)
	if !identifierPart(last, original) || len(rest) == len(literal) {
		return true
	}

	next, _ := utf8.DecodeRuneInString(rest[len(literal):])

	return !identifierPart(next, original)
}
