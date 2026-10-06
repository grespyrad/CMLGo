package cmlgo

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
)

const (
	unicodeUnitLength = 5
	unicodePairLength = 11
)

func unicodeEscape(source string) (rune, int, error) {
	first, err := unicodeUnit(source)
	if err != nil {
		return 0, 0, err
	}

	switch {
	case first >= 0xD800 && first <= 0xDBFF:
		return surrogatePair(source, first)
	case first >= 0xDC00 && first <= 0xDFFF:
		return 0, 0, fmt.Errorf("unpaired Unicode surrogate: %w", ErrInvalidModel)
	default:
		return first, unicodeUnitLength, nil
	}
}

func surrogatePair(source string, first rune) (rune, int, error) {
	if len(source) < unicodePairLength || !strings.HasPrefix(source[unicodeUnitLength:], "\\u") {
		return 0, 0, fmt.Errorf("missing low Unicode surrogate: %w", ErrInvalidModel)
	}

	second, err := unicodeUnit(source[unicodeUnitLength+1:])
	if err != nil {
		return 0, 0, err
	}

	if second < 0xDC00 || second > 0xDFFF {
		return 0, 0, fmt.Errorf("invalid low Unicode surrogate: %w", ErrInvalidModel)
	}

	return utf16.DecodeRune(first, second), unicodePairLength, nil
}

func unicodeUnit(source string) (rune, error) {
	if len(source) < unicodeUnitLength || source[0] != 'u' {
		return 0, fmt.Errorf("incomplete Unicode escape: %w", ErrInvalidModel)
	}

	value, err := strconv.ParseUint(source[1:unicodeUnitLength], 16, 16)
	if err != nil {
		return 0, fmt.Errorf("decode Unicode escape: %w", err)
	}

	return rune(value), nil
}
