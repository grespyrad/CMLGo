package cmlgo

import (
	"fmt"
	"slices"
	"strings"
)

// PlantUML generates a context map and per-context class diagrams. It requires
// a valid Result from Validate or ValidateFile and invokes no external renderer.
// Diagram layout and filenames are not byte-compatible with the Java generator.
//
//nolint:cyclop,funlen,gocognit,gocyclo,maintidx // Ветви CML проверяются conformance и corpus-тестами.
func PlantUML(result Result) (map[string]string, error) {
	if !result.Valid || result.Model == nil {
		return nil, fmt.Errorf("generate PlantUML: %w", ErrInvalidModel)
	}

	models := result.Documents
	if len(models) == 0 {
		models = []*Node{result.Model}
	}

	v := validator{named: map[string][]*Node{}, nodes: nil, diagnostics: nil}

	for _, model := range models {
		v.nodes = append(v.nodes, descendants(model)...)
	}

	for _, n := range v.nodes {
		if n.Name() != "" {
			v.named[n.Name()] = append(v.named[n.Name()], n)
		}
	}

	aliases := map[*Node]string{}

	for i, n := range v.nodes {
		aliases[n] = fmt.Sprintf("n%d", i)
	}

	files := map[string]string{}

	for _, model := range models {
		for _, m := range model.Children(fieldMap) {
			var b textBuffer

			b.text("@startuml\n")

			for _, value := range m.Fields[fieldBoundedContexts] {
				bc := v.resolveValue(m, value)
				if bc != nil {
					b.formatf("rectangle %s as %s\n", pumlQuote(bc.Name()), aliases[bc])
				}
			}

			for _, r := range m.Children(fieldRelationships) {
				left, right, arrow, label := v.resolve(
					r,
					fieldUpstream,
				), v.resolve(
					r,
					fieldDownstream,
				), "-->", "Upstream-Downstream"
				if r.Kind == kindSharedKernel || r.Kind == kindPartnership {
					left = v.resolve(r, fieldParticipant1)
					right = v.resolve(r, fieldParticipant2)
					arrow = "<-->"
					label = r.Kind
				}

				if r.Kind == kindCustomerSupplierRelationship {
					label = "Customer-Supplier"
				}

				if left != nil && right != nil {
					b.formatf("%s %s %s : %s\n", aliases[left], arrow, aliases[right], pumlText(label))
				}
			}

			b.text("@enduml\n")

			name := m.Name()
			if name == "" {
				name = kindContextMap
			}

			files[name+".puml"] = b.String()
		}
	}

	for _, bc := range v.nodes {
		if bc.Kind != kindBoundedContext {
			continue
		}

		var b textBuffer

		b.text("@startuml\n")
		b.formatf("package %s {\n", pumlQuote(bc.Name()))

		for _, n := range descendants(bc) {
			if !typeMatches(n.Kind, kindSimpleDomainObject) && !inSet(n.Kind, "Service Resource Consumer Repository") {
				continue
			}

			kind := "class"

			if n.Kind == kindEnum {
				kind = symbolEnum
			}

			b.formatf("  %s %s as %s <<%s>> {\n", kind, pumlQuote(n.Name()), aliases[n], n.Kind)

			if flag(n, fieldAggregateRoot) {
				b.text("    {static} aggregateRoot\n")
			}

			for _, a := range n.Children(fieldAttributes) {
				b.formatf("    %s : %s\n", pumlText(a.Name()), pumlText(a.Text(fieldType)))
			}

			for _, r := range n.Children(fieldReferences) {
				b.formatf("    %s : %s\n", pumlText(r.Name()), pumlText(r.Text(fieldDomainObjectType)))
			}

			for _, op := range n.Children(fieldOperations) {
				params := make([]string, 0, len(op.Children(fieldParameters)))

				for _, p := range op.Children(fieldParameters) {
					params = append(params, p.Name()+" : "+complexType(one(p, fieldParameterType)))
				}

				b.formatf(
					"    %s(%s) : %s\n",
					pumlText(op.Name()),
					pumlText(strings.Join(params, ", ")),
					pumlText(complexType(one(op, fieldReturnType))),
				)
			}

			for _, value := range n.Children(fieldValues) {
				b.formatf("    %s\n", pumlText(value.Name()))
			}

			b.text("  }\n")
		}

		b.text("}\n")

		if vision := bc.Text(fieldDomainVisionStatement); vision != "" {
			b.formatf("note as vision_%s\n%s\nend note\n", aliases[bc], pumlText(vision))
		}

		for _, n := range descendants(bc) {
			if n.Kind != kindReference && n.Kind != kindAssociation {
				continue
			}

			target := v.resolve(n, fieldDomainObjectType)
			if target == nil || n.parent == nil {
				continue
			}

			if ancestor(target, kindBoundedContext) != bc {
				b.formatf("class %s as %s <<external>>\n", pumlQuote(target.Name()), aliases[target])
			}

			cardinality := "1"

			if collection(n) {
				cardinality = "*"
			}

			b.formatf(
				"%s --> %s : %s [%s]\n",
				aliases[n.parent],
				aliases[target],
				pumlText(n.Name()),
				cardinality,
			)
		}

		invariantNotes(&b, bc, aliases)
		b.text("@enduml\n")

		files[bc.Name()+".puml"] = b.String()
	}

	return files, nil
}

func pumlText(s string) string {
	return strings.NewReplacer("\r", "", "\n", "\\n", "@", "<U+0040>").Replace(s)
}

func pumlQuote(s string) string { return "\"" + strings.ReplaceAll(pumlText(s), "\"", "\\\"") + "\"" }

func complexType(n *Node) string {
	if n == nil {
		return symbolVoid
	}

	typ := n.Text(fieldType)
	if has(n, fieldDomainObjectType) {
		typ = n.Text(fieldDomainObjectType)
	}

	if has(n, fieldCollectionType) {
		typ = n.Text(fieldCollectionType) + "<" + typ + ">"
	}

	if has(n, fieldMapCollectionType) {
		typ = "Map<" + n.Text(fieldMapKeyType) + ", " + typ + ">"
	}

	return typ
}

// DiagramNames returns sorted keys for deterministic file emission.
func DiagramNames(files map[string]string) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}

	slices.Sort(names)

	return names
}

type textBuffer struct{ data []byte }

func (b *textBuffer) text(text string) { b.data = append(b.data, text...) }
func (b *textBuffer) formatf(format string, args ...any) {
	b.data = fmt.Appendf(b.data, format, args...)
}
func (b *textBuffer) String() string { return string(b.data) }
