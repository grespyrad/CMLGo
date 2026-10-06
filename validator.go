package cmlgo

import (
	"cmp"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Validate parses and checks a standalone document. Imports require ValidateFile;
// this function never reads the filesystem.
func Validate(file string, source []byte) Result {
	return ValidateWithDialect(file, source, Dialect{Aliases: nil, Original: false})
}

// ValidateWithDialect проверяет standalone модель выбранного диалекта.
func ValidateWithDialect(file string, source []byte, dialect Dialect) Result {
	r := ParseWithDialect(file, source, dialect)
	if !r.Valid {
		return r
	}

	for _, i := range r.Model.Children(fieldImports) {
		r.Diagnostics = append(
			r.Diagnostics,
			Diagnostic{
				Severity: symbolError,
				Code:     "import-loader",
				Message:  "Use ValidateFile to resolve imports",
				Position: i.Position,
			},
		)
	}

	return validateModels(r, []*Node{r.Model})
}

// ValidateFile loads a CML file and its transitive relative/file URI imports.
// Each call owns its graph; repeated and cyclic imports are visited once.
func ValidateFile(path string) Result {
	return ValidateFileWithDialect(path, Dialect{Aliases: nil, Original: false})
}

// ValidateFileWithDialect проверяет импортируемые документы с единым словарём.
//
//nolint:cyclop,funlen,gocognit,gocyclo // Ветви CML проверяются conformance и corpus-тестами.
func ValidateFileWithDialect(path string, dialect Dialect) Result {
	models := map[string]*Node{}
	visited := map[string]bool{}

	var (
		diagnostics []Diagnostic
		root        *Node
		load        func(string, Position)
	)

	load = func(path string, position Position) {
		absolute, err := filepath.Abs(path)
		if err != nil {
			diagnostics = append(
				diagnostics,
				Diagnostic{
					Severity: symbolError,
					Code:     "io",
					Message:  fmt.Sprintf("Resolve path: %v", err),
					Position: position,
				},
			)

			return
		}

		absolute = filepath.Clean(absolute)
		if visited[absolute] {
			return
		}

		visited[absolute] = true
		if len(visited) > maxImports {
			diagnostics = append(
				diagnostics,
				Diagnostic{
					Severity: symbolError,
					Code:     symbolLimit,
					Message:  "Import graph exceeds 256 files",
					Position: position,
				},
			)

			return
		}

		data, err := os.ReadFile(absolute)
		if err != nil {
			diagnostics = append(
				diagnostics,
				Diagnostic{
					Severity: symbolError,
					Code:     "io",
					Message:  fmt.Sprintf("Read CML: %v", err),
					Position: position,
				},
			)

			return
		}

		r := ParseWithDialect(absolute, data, dialect)

		diagnostics = append(diagnostics, r.Diagnostics...)

		if !r.Valid {
			return
		}

		models[absolute] = r.Model

		if root == nil {
			root = r.Model
		}

		for _, i := range r.Model.Children(fieldImports) {
			uri := i.Text(fieldImportURI)
			parsed, err := url.Parse(uri)

			if err != nil || parsed.Scheme != "" && parsed.Scheme != "file" {
				diagnostics = append(
					diagnostics,
					Diagnostic{
						Severity: symbolError,
						Code:     diagnosticImportURI,
						Message:  "Import must be a relative path or local file URI",
						Position: i.Position,
					},
				)

				continue
			}

			if parsed.Host != "" && parsed.Host != "localhost" {
				diagnostics = append(
					diagnostics,
					Diagnostic{
						Severity: symbolError,
						Code:     diagnosticImportURI,
						Message:  "Remote file URI is unsupported",
						Position: i.Position,
					},
				)

				continue
			}

			imported := parsed.Path
			if !filepath.IsAbs(imported) {
				imported = filepath.Join(filepath.Dir(absolute), imported)
			}

			if !strings.HasSuffix(imported, ".cml") {
				diagnostics = append(
					diagnostics,
					Diagnostic{
						Severity: symbolError,
						Code:     diagnosticImportURI,
						Message:  "Imported file must have .cml extension",
						Position: i.Position,
					},
				)

				continue
			}

			load(imported, i.Position)
		}
	}
	load(path, Position{File: path, Line: 1, Column: 1, Offset: 0})

	var (
		all   = make([]*Node, 0, len(models))
		paths = make([]string, 0, len(models))
	)

	for p := range models {
		paths = append(paths, p)
	}

	slices.Sort(paths)

	for _, p := range paths {
		all = append(all, models[p])
	}

	return validateModels(Result{Model: root, Diagnostics: diagnostics, Valid: false, Documents: nil}, all)
}

func descendants(n *Node) []*Node {
	if n == nil {
		return nil
	}

	out := []*Node{n}

	fields := make([]string, 0, len(n.Fields))

	for k := range n.Fields {
		fields = append(fields, k)
	}

	slices.Sort(fields)

	for _, field := range fields {
		for _, v := range n.Fields[field] {
			if v.Node != nil {
				out = append(out, descendants(v.Node)...)
			}
		}
	}

	return out
}

func ancestor(n *Node, kind string) *Node {
	for n != nil {
		if n.Kind == kind {
			return n
		}

		n = n.parent
	}

	return nil
}

func contained(n, container *Node) bool {
	if container == nil {
		return false
	}

	for n != nil {
		if n == container {
			return true
		}

		n = n.parent
	}

	return false
}
func has(n *Node, field string) bool  { return n != nil && len(n.Fields[field]) > 0 }
func flag(n *Node, field string) bool { return n.Text(field) == symbolTrue }
func one(n *Node, field string) *Node {
	nodes := n.Children(field)
	if len(nodes) > 0 {
		return nodes[0]
	}

	return nil
}

func includes(n *Node, field, value string) bool {
	if n == nil {
		return false
	}

	for _, v := range n.Fields[field] {
		if v.Text == value {
			return true
		}
	}

	return false
}

//nolint:cyclop,gocognit,gocyclo // Ветви CML проверяются conformance и corpus-тестами.
func typeMatches(kind, target string) bool {
	if kind == target {
		return true
	}

	switch target {
	case kindDomainPart:
		return kind == kindDomain || kind == kindSubdomain
	case kindUserRequirement:
		return kind == kindUseCase || kind == kindUserStory
	case kindSimpleDomainObject:
		return typeMatches(kind, kindDomainObject) || kind == kindBasicType || kind == kindEnum ||
			kind == kindDataTransferObject ||
			kind == kindTrait
	case kindDomainObject:
		return kind == kindEntity || kind == kindValueObject || kind == kindDomainEvent || kind == kindCommandEvent
	case kindEvent:
		return kind == kindDomainEvent || kind == kindCommandEvent
	case kindServiceRepositoryOption:
		return kind == kindService || kind == kindRepository
	case kindServiceRepositoryOperationOption:
		return kind == kindServiceOperation || kind == kindRepositoryOperation
	case kindAbstractStakeholder:
		return kind == kindStakeholder || kind == kindStakeholderGroup
	}

	return false
}

type validator struct {
	nodes       []*Node
	named       map[string][]*Node
	diagnostics []Diagnostic
}

func (v *validator) issue(n *Node, severity, code, message string) {
	v.diagnostics = append(
		v.diagnostics,
		Diagnostic{Severity: severity, Code: code, Message: message, Position: n.Position},
	)
}

func (v *validator) reportError(n *Node, code, message string) {
	v.issue(n, symbolError, code, message)
}
func (v *validator) warn(n *Node, code, message string) { v.issue(n, symbolWarning, code, message) }

func (v *validator) resolve(n *Node, field string) *Node {
	if n == nil || len(n.Fields[field]) == 0 {
		return nil
	}

	return v.resolveValue(n, n.Fields[field][0])
}

//nolint:cyclop,gocognit,gocyclo // Ветви CML проверяются conformance и corpus-тестами.
func (v *validator) resolveValue(n *Node, value ASTValue) *Node {
	var candidates []*Node

	for _, candidate := range v.named[value.Text] {
		if typeMatches(candidate.Kind, value.RefType) {
			if (n.Kind == kindReference || n.Kind == kindAssociation) && typeMatches(candidate.Kind, kindDomainObject) {
				if ancestor(n, kindBoundedContext) != nil && ancestor(candidate, kindDomain) != nil ||
					ancestor(n, kindDomain) != nil && ancestor(candidate, kindBoundedContext) != nil {
					continue
				}
			}

			candidates = append(candidates, candidate)
		}
	}

	// Context-sensitive references in coordinations, delegates and opposites.
	var scope *Node

	if n.Kind == kindCoordinationStep {
		if value.RefType == kindService {
			scope = one(v.resolve(n, fieldBoundedContext), fieldApplication)
		}

		if value.RefType == kindServiceOperation {
			scope = v.resolve(n, fieldService)
		}
	}

	if n.Kind == kindOppositeHolder {
		scope = v.resolve(n.parent, fieldDomainObjectType)
	}

	if strings.HasSuffix(n.Kind, "OperationDelegate") && value.RefType != kindService &&
		value.RefType != kindServiceRepositoryOption {
		scope = v.resolve(n, fieldDelegate)
	}

	if scope != nil {
		for _, candidate := range candidates {
			if contained(candidate, scope) {
				return candidate
			}
		}

		return nil
	}

	for parent := n.parent; parent != nil; parent = parent.parent {
		for _, candidate := range candidates {
			if contained(candidate, parent) {
				return candidate
			}
		}
	}

	if len(candidates) > 0 {
		return candidates[0]
	}

	return nil
}

//nolint:cyclop,funlen,gocognit,gocyclo // Ветви CML проверяются conformance и corpus-тестами.
func validateModels(result Result, models []*Node) Result {
	result.Documents = models

	v := validator{named: map[string][]*Node{}, diagnostics: result.Diagnostics, nodes: nil}

	for _, model := range models {
		v.nodes = append(v.nodes, descendants(model)...)
	}

	slices.SortStableFunc(v.nodes, func(a, b *Node) int {
		if a.Position.File != b.Position.File {
			return strings.Compare(a.Position.File, b.Position.File)
		}

		return cmp.Compare(a.Position.Offset, b.Position.Offset)
	})

	for _, n := range v.nodes {
		if n.Name() != "" {
			v.named[n.Name()] = append(v.named[n.Name()], n)
		}
	}

	for _, n := range v.nodes {
		fields := make([]string, 0, len(n.Fields))

		for field := range n.Fields {
			fields = append(fields, field)
		}

		slices.Sort(fields)

		for _, field := range fields {
			for _, value := range n.Fields[field] {
				if value.RefType != "" && v.resolveValue(n, value) == nil {
					v.diagnostics = append(
						v.diagnostics,
						Diagnostic{
							Severity: symbolError,
							Code:     "unresolved-reference",
							Message:  fmt.Sprintf("Couldn't resolve reference to %s '%s'", value.RefType, value.Text),
							Position: value.Position,
						},
					)
				}
			}
		}

		v.uniqueness(n)
		v.strategic(n)
		v.tactical(n)
		v.invariants(n)
	}

	slices.SortStableFunc(v.diagnostics, func(a, b Diagnostic) int {
		if a.Position.File != b.Position.File {
			return strings.Compare(a.Position.File, b.Position.File)
		}

		if a.Position.Offset != b.Position.Offset {
			return cmp.Compare(a.Position.Offset, b.Position.Offset)
		}

		if a.Code != b.Code {
			return strings.Compare(a.Code, b.Code)
		}

		return strings.Compare(a.Message, b.Message)
	})

	result.Diagnostics = v.diagnostics
	if result.Diagnostics == nil {
		result.Diagnostics = []Diagnostic{}
	}

	result.Valid = result.Model != nil
	for _, d := range result.Diagnostics {
		if d.Severity == symbolError {
			result.Valid = false
		}
	}

	return result
}
