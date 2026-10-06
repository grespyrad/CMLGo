package cmlgo

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func collection(n *Node) bool {
	return has(n, fieldCollectionType) && n.Text(fieldCollectionType) != symbolNone
}

func inSet(s, values string) bool {
	for v := range strings.FieldsSeq(values) {
		if s == v {
			return true
		}
	}

	return false
}

//nolint:cyclop,funlen,gocognit,gocyclo,nestif // Ветви CML проверяются conformance и corpus-тестами.
func (v *validator) tactical(n *Node) {
	if flag(n, fieldGapClass) && flag(n, fieldNoGapClass) {
		v.reportError(n, "gap", "Unclear specification of gap")
	}

	if n.Name() != "" {
		initial, _ := utf8.DecodeRuneInString(n.Name())
		if (typeMatches(n.Kind, kindSimpleDomainObject) || n.Kind == kindService || n.Kind == kindRepository) &&
			!unicode.IsUpper(initial) {
			v.warn(n, "name-case", "Name should begin with an upper case letter")
		}

		if inSet(n.Kind, "Attribute Reference DtoAttribute DtoReference Parameter") &&
			!unicode.IsLower(initial) {
			v.warn(n, "name-case", "Name should begin with a lower case letter")
		}
	}

	if typeMatches(n.Kind, kindDomainObject) {
		if one(n, fieldRepository) != nil && !flag(n, fieldAggregateRoot) {
			v.reportError(n, "repository-root", "Only aggregate roots can have Repository")
		}

		if target := v.resolve(n, fieldBelongsTo); target != nil && !flag(target, fieldAggregateRoot) {
			v.reportError(n, "belongs-to", "belongsTo must refer to an aggregate root")
		}

		if has(n, fieldDiscriminatorValue) && inSet(n.Kind, "Entity ValueObject") && v.resolve(n, fieldExtends) == nil {
			v.reportError(n, "discriminator", "discriminatorValue requires extending another domain object")
		}
	}

	if n.Kind == kindValueObject && flag(n, fieldNotPersistent) {
		if flag(n, fieldAggregateRoot) {
			v.reportError(n, "persistent-root", "aggregateRoot requires a persistent ValueObject")
		}

		if flag(n, fieldScaffold) {
			v.reportError(n, "persistent-scaffold", "Scaffold is not useful for a non-persistent ValueObject")
		}
	}

	if typeMatches(n.Kind, kindEvent) && !flag(n, fieldPersistent) {
		if flag(n, fieldScaffold) {
			v.reportError(n, "persistent-scaffold", "Scaffold is not useful for a non-persistent event")
		}

		if one(n, fieldRepository) != nil {
			v.reportError(n, "persistent-repository", "Repository is not useful for a non-persistent event")
		}
	}

	if n.Kind == kindRepository && !strings.HasSuffix(n.Name(), kindRepository) {
		v.reportError(n, "repository-name", "Name of repository must end with 'Repository'")
	}

	if inSet(n.Kind, "Attribute Reference") {
		if flag(n, fieldRequired) && flag(n, fieldNotChangeable) {
			v.warn(n, "property-required", "The combination not changeable and required does not make sense")
		}

		if flag(n, fieldKey) && flag(n, fieldNotChangeable) {
			v.warn(n, "property-key", "Key property is always not changeable")
		}

		if flag(n, fieldKey) && flag(n, fieldRequired) {
			v.warn(n, "property-key", "Key property is always required")
		}

		if flag(n, fieldKey) && flag(n, fieldNullable) {
			nonnullable := false

			for _, child := range descendants(n.parent) {
				if child.parent == n.parent && inSet(child.Kind, "Attribute Reference DtoAttribute DtoReference") &&
					flag(child, fieldKey) &&
					!flag(child, fieldNullable) {
					nonnullable = true
				}
			}

			if !nonnullable {
				v.reportError(
					n,
					"nullable-key",
					"Natural key must not be nullable; composite keys need one non-nullable property",
				)
			}
		}
	}

	switch n.Kind {
	case kindAttribute:
		v.checktacticalAttribute(n)

	case kindReference:
		v.checktacticalReference(n)

	case kindEnum:
		v.checktacticalEnum(n)

	case kindParameter, kindServiceOperation, kindRepositoryOperation:
		field := fieldReturnType

		if n.Kind == kindParameter {
			field = fieldParameterType
		}

		typ := one(n, field)
		if typ != nil && !has(typ, fieldDomainObjectType) && has(typ, fieldType) {
			for _, target := range v.named[typ.Text(fieldType)] {
				if typeMatches(target.Kind, kindSimpleDomainObject) {
					v.warn(typ, "reference-notation", "Use @ for a domain object type")

					break
				}
			}
		}
	default:
		// Другие правила не требуют этой проверки.
	}
}

//nolint:cyclop,gocognit,gocyclo // Ветви CML проверяются conformance и corpus-тестами.
func (v *validator) attribute(n *Node) {
	typ := n.Text(fieldType)
	many := collection(n)
	isString := typ == symbolStringSymbol && !many
	numeric := inSet(typ, "int long float double Integer Long Float Double BigInteger BigDecimal") && !many
	temporal := inSet(typ, "Date DateTime Timestamp") && !many
	boolean := inSet(typ, "boolean Boolean") && !many
	checks := []struct {
		field   string
		allowed bool
		message string
	}{
		{fieldLength, isString, "length is only relevant for strings"},
		{fieldCreditCardNumber, isString, "creditCardNumber is only relevant for strings"},
		{fieldEmail, isString, "email is only relevant for strings"},
		{fieldNotEmpty, isString || many, "notEmpty is only relevant for strings or collections"},
		{fieldPast, temporal, "past is only relevant for temporal types"},
		{fieldFuture, temporal, "future is only relevant for temporal types"},
		{fieldMin, numeric, "min is only relevant for numeric types"},
		{fieldMax, numeric, "max is only relevant for numeric types"},
		{fieldRange, numeric, "range is only relevant for numeric types"},
		{fieldDigits, numeric, "digits is only relevant for numeric types"},
		{fieldAssertTrue, boolean, "assertTrue is only relevant for boolean types"},
		{fieldAssertFalse, boolean, "assertFalse is only relevant for boolean types"},
	}

	for _, c := range checks {
		if has(n, c.field) && !c.allowed {
			v.reportError(n, "attribute-constraint", c.message)
		}
	}

	if has(n, fieldLength) {
		numericLength := n.Text(fieldLength) != ""
		for _, r := range n.Text(fieldLength) {
			if r < '0' || r > '9' {
				numericLength = false
			}
		}

		if !numericLength {
			v.reportError(n, "attribute-length", "length value must be numeric")
		}
	}

	if flag(n, fieldNullable) && inSet(typ, "int long float double boolean") && !many {
		v.reportError(n, "nullable-primitive", "nullable is not relevant for primitive types")
	}

	for _, candidate := range v.named[typ] {
		if typeMatches(candidate.Kind, kindSimpleDomainObject) {
			v.warn(n, "reference-notation", "Use - for a domain object reference")

			break
		}
	}
}

//nolint:cyclop,funlen,gocognit,gocyclo // Ветви CML проверяются conformance и corpus-тестами.
func (v *validator) reference(n *Node) {
	many := collection(n)
	target := v.resolve(n, fieldDomainObjectType)
	oppositeHolder := one(n, fieldOppositeHolder)
	opposite := v.resolve(oppositeHolder, fieldOpposite)
	bidirectional := opposite != nil
	manyMany := many && bidirectional && collection(opposite)
	unidirectionalMany := many && !bidirectional

	if flag(n, fieldCache) && !many {
		v.reportError(n, "reference-cache", "Cache is only applicable for collections")
	}

	if flag(n, fieldInverse) && (!many && (!bidirectional || collection(opposite))) {
		v.reportError(n, "reference-inverse", "Inverse is only applicable for many or one-to-one references")
	}

	if has(n, fieldDatabaseJoinTable) {
		if !manyMany && (!unidirectionalMany || flag(n, fieldInverse)) {
			v.reportError(
				n,
				"reference-join-table",
				"databaseJoinTable requires many-to-many or unidirectional many without inverse",
			)
		}

		if manyMany && has(opposite, fieldDatabaseJoinTable) {
			v.warn(n, "reference-join-table", "Define databaseJoinTable on only one side")
		}
	}

	if has(n, fieldDatabaseJoinColumn) && (!unidirectionalMany || flag(n, fieldInverse)) {
		v.reportError(n, "reference-join-column", "databaseJoinColumn requires unidirectional many without inverse")
	}

	if many && flag(n, fieldNullable) {
		v.reportError(n, "reference-nullable", "nullable is not applicable for references with cardinality many")
	}

	if has(n, fieldDatabaseColumn) && many && bidirectional && !collection(opposite) {
		v.reportError(n, "reference-column", "databaseColumn should be defined at the opposite side")
	}

	if bidirectional && v.resolve(one(opposite, fieldOppositeHolder), fieldOpposite) != n {
		v.reportError(n, "reference-opposite", "Opposite must specify this reference as its opposite")
	}

	if flag(n, fieldNotChangeable) && many {
		v.warn(n, "reference-changeable", "Collection content remains changeable")
	}

	if has(n, fieldOrderBy) && !inSet(n.Text(fieldCollectionType), "Bag List") {
		v.reportError(n, "reference-order", "orderBy is only applicable for Bag or List")
	}

	if flag(n, fieldOrderColumn) && n.Text(fieldCollectionType) != symbolList {
		v.reportError(n, "reference-order", "orderColumn is only applicable for List")
	}

	if has(n, fieldOrderBy) && flag(n, fieldOrderColumn) {
		v.reportError(n, "reference-order", "Use either orderBy or orderColumn")
	}

	if flag(n, fieldKey) && many {
		v.reportError(n, "reference-key", "Natural key cannot be a many reference")
	}

	if target != nil && inSet(target.Kind, "BasicType Enum") {
		if has(n, fieldCascade) {
			v.reportError(n, "reference-cascade", "Cascade is not applicable for BasicType or enum")
		}

		if flag(n, fieldCache) {
			v.reportError(n, "reference-cache", "Cache is not applicable for BasicType or enum")
		}
	}

	if flag(n, fieldNotEmpty) && !many {
		v.reportError(n, "reference-not-empty", "notEmpty is only relevant for collections")
	}

	if has(n, fieldSize) && !many {
		v.reportError(n, "reference-size", "size is only relevant for collections")
	}

	if target != nil {
		count := 0

		for _, candidate := range v.named[target.Name()] {
			if typeMatches(candidate.Kind, kindSimpleDomainObject) &&
				ancestor(candidate, kindContextMappingModel) == ancestor(n, kindContextMappingModel) {
				count++
			}
		}

		if count > 1 {
			v.warn(n, "ambiguous-reference", "Reference name resolves to multiple domain objects in this model")
		}

		current, other := ancestor(n, kindBoundedContext), ancestor(target, kindBoundedContext)
		if current != nil && other != nil && current != other && !v.reachable(current, other) {
			v.warn(n, "unreachable-reference", "Reference type is not exposed by a reachable upstream context")
		}
	}
}

//nolint:cyclop,gocognit,gocyclo // Ветви CML проверяются conformance и corpus-тестами.
func (v *validator) enum(n *Node) {
	values, attrs := n.Children(fieldValues), n.Children(fieldAttributes)
	if len(values) == 0 {
		v.reportError(n, "enum-values", "At least one enum value must be defined")

		return
	}

	count := len(values[0].Children(fieldParameters))
	keys := 0

	for _, value := range values {
		size := len(value.Children(fieldParameters))
		if size != count {
			v.reportError(value, "enum-parameters", "Enum values must have the same number of parameters")
		}

		if len(attrs) > 0 && size != len(attrs) {
			v.reportError(value, "enum-attributes", "Enum attribute not defined")
		}

		if len(attrs) == 0 && size > 1 {
			v.reportError(value, "enum-implicit", "Only one implicit value attribute is allowed")
		}
	}

	hint := n.Text(fieldHint)

	for _, attr := range attrs {
		if flag(attr, fieldKey) {
			keys++

			if strings.Contains(hint, fieldOrdinal) {
				v.reportError(attr, "enum-ordinal", "ordinal is not allowed with a key attribute")
			}

			if strings.Contains(hint, symbolDatabaseLength) && attr.Text(fieldType) != symbolStringSymbol {
				v.reportError(attr, "enum-length", "databaseLength requires a String key")
			}
		}
	}

	if keys > 1 {
		v.reportError(n, "enum-key", "Only one enum attribute can be defined as key")
	}

	if strings.Contains(hint, fieldOrdinal) && strings.Contains(hint, symbolDatabaseLength) {
		v.reportError(n, "enum-ordinal", "ordinal and databaseLength cannot be combined")
	}

	if strings.Contains(hint, fieldOrdinal) && len(attrs) == 0 {
		for _, value := range values {
			if len(value.Children(fieldParameters)) == 1 {
				v.reportError(value, "enum-ordinal", "ordinal is not allowed with an implicit enum value")
			}
		}
	}
}
func (v *validator) checktacticalAttribute(n *Node) { v.attribute(n) }
func (v *validator) checktacticalReference(n *Node) { v.reference(n) }
func (v *validator) checktacticalEnum(n *Node)      { v.enum(n) }
