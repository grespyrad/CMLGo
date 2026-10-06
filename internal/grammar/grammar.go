// Package grammar contains the pinned CML 6.12.0 grammar.
package grammar

import _ "embed"

// JSON is generated from the two Apache-2.0 Xtext source files.
//
//go:embed grammar.json
var JSON []byte

// OriginalJSON содержит неизменённую скомпилированную грамматику 6.12.0.
//
//go:embed original.json
var OriginalJSON []byte
