package cmlgo

import (
	"fmt"
)

//nolint:cyclop,funlen,gocognit,gocyclo // Ветви CML проверяются conformance и corpus-тестами.
func (v *validator) uniqueness(n *Node) {
	global := n.Kind
	switch n.Kind {
	case kindUseCase, kindUserStory:
		global = kindUserRequirement
	case kindBoundedContext, kindDomain, kindSubdomain, kindSculptorModule, kindAggregate, kindRepository:
	default:
		global = ""
	}

	if global != "" {
		count := 0

		for _, candidate := range v.named[n.Name()] {
			if typeMatches(candidate.Kind, global) {
				count++
			}
		}

		if count > 1 {
			v.reportError(n, "duplicate-name", fmt.Sprintf("%s name '%s' is not unique", n.Kind, n.Name()))
		}
	}

	if n.Kind == kindFlow || n.Kind == kindCoordination {
		count := 0

		for _, candidate := range v.named[n.Name()] {
			if candidate.Kind == n.Kind &&
				ancestor(candidate, kindContextMappingModel) == ancestor(n, kindContextMappingModel) {
				count++
			}
		}

		if count > 1 {
			v.reportError(n, "duplicate-name", fmt.Sprintf("%s name '%s' is not unique", n.Kind, n.Name()))
		}
	}

	if n.Kind == kindAggregate || n.Kind == kindSculptorModule || n.Kind == kindSubdomain {
		seen := map[string]bool{}

		for _, child := range descendants(n) {
			if child.parent == n && typeMatches(child.Kind, kindSimpleDomainObject) {
				if seen[child.Name()] {
					v.reportError(child, "duplicate-domain-object", "Domain object name is not unique in its container")
				}

				seen[child.Name()] = true
			}
		}
	}

	if n.Kind == kindBoundedContext || n.Kind == kindSubdomain {
		seen := map[string]bool{}

		for _, child := range descendants(n) {
			if child.Kind == kindService {
				if seen[child.Name()] {
					v.reportError(child, "duplicate-service", "Service name is not unique in its context")
				}

				seen[child.Name()] = true
			}
		}
	}

	if typeMatches(n.Kind, kindSimpleDomainObject) {
		count := 0

		for _, candidate := range v.named[n.Name()] {
			if typeMatches(candidate.Kind, kindSimpleDomainObject) &&
				ancestor(candidate, kindContextMappingModel) == ancestor(n, kindContextMappingModel) {
				count++
			}
		}

		if count > 1 {
			v.warn(n, "ambiguous-domain-object", "Domain object name also occurs elsewhere in this model")
		}
	}
}

//nolint:cyclop,funlen,gocognit,gocyclo,nestif,revive // Ветви CML проверяются conformance и corpus-тестами.
func (v *validator) strategic(n *Node) {
	switch n.Kind {
	case kindContextMap:
		v.checkstrategicContextMap(n)

	case kindCustomerSupplierRelationship:
		v.checkstrategicCustomerSupplierRelationship(n)

	case kindBoundedContext:
		v.checkstrategicBoundedContext(n)

	case kindSubdomain:
		v.checkstrategicSubdomain(n)

	case kindAggregate:
		v.checkstrategicAggregate(n)

	case kindSculptorModule:
		v.checkstrategicSculptorModule(n)

	case kindCoordinationStep:
		v.checkstrategicCoordinationStep(n)

	default:
		// Другие правила не требуют этой проверки.
	}

	if ancestor(n, kindFlow) != nil {
		for _, field := range []string{fieldCommand, fieldOperation, fieldCommands, fieldOperations} {
			for _, ref := range n.Fields[field] {
				target := v.resolveValue(n, ref)
				if target != nil && ancestor(target, kindBoundedContext) != ancestor(n, kindBoundedContext) {
					v.reportError(
						n,
						"flow-context",
						"Command or operation must belong to the same bounded context as the flow",
					)
				}
			}
		}
	}

	if has(n, fieldStateTransition) {
		agg := ancestor(n, kindAggregate)
		if n.Kind == kindDomainEventProductionStep {
			agg = v.resolve(n, fieldAggregate)
		}

		if agg != nil {
			states := map[*Node]bool{}

			for _, en := range agg.Children(fieldDomainObjects) {
				if flag(en, fieldDefinesAggregateLifecycle) {
					for _, state := range en.Children(fieldValues) {
						states[state] = true
					}
				}
			}

			for _, part := range descendants(one(n, fieldStateTransition)) {
				for _, values := range part.Fields {
					for _, ref := range values {
						if ref.RefType == kindEnumValue {
							state := v.resolveValue(part, ref)
							if state != nil && !states[state] {
								v.reportError(
									part,
									"state-transition",
									"State does not belong to the aggregate lifecycle",
								)
							}
						}
					}
				}
			}
		}
	}

	if n.Kind == kindNormalFeature || n.Kind == kindStoryFeature {
		for _, field := range []string{fieldEntity, fieldVerb} {
			text := n.Text(field)
			if text != "" && !validIdentifier(text) {
				v.warn(n, "feature-identifier", fmt.Sprintf("Feature %s is not a valid identifier", field))
			}
		}
	}
}

func validIdentifier(value string) bool {
	return value != "" && identifierLength(value, false) == len(value)
}

//nolint:cyclop,gocognit,gocyclo // Ветви CML проверяются conformance и corpus-тестами.
func (v *validator) reachable(downstream, upstream *Node) bool {
	if downstream == upstream {
		return true
	}

	for _, n := range v.nodes {
		if n.Kind != kindContextMap {
			continue
		}

		for _, rel := range n.Children(fieldRelationships) {
			if v.resolve(rel, fieldDownstream) == downstream && v.resolve(rel, fieldUpstream) == upstream {
				return true
			}

			if (v.resolve(rel, fieldParticipant1) == downstream && v.resolve(rel, fieldParticipant2) == upstream) ||
				(v.resolve(rel, fieldParticipant2) == downstream && v.resolve(rel, fieldParticipant1) == upstream) {
				return true
			}
		}
	}

	return false
}

//nolint:cyclop,gocognit,gocyclo // Ветви CML проверяются conformance и corpus-тестами.
func (v *validator) checkstrategicContextMap(n *Node) {
	teams := 0

	for _, ref := range n.Fields[fieldBoundedContexts] {
		bc := v.resolveValue(n, ref)
		if bc != nil && bc.Text(fieldType) == symbolTEAM {
			teams++

			if n.Text(fieldType) == symbolSYSTEM_LANDSCAPE {
				v.reportError(n, "map-team", "SYSTEM_LANDSCAPE map cannot contain TEAM contexts")
			}
		}
	}

	if n.Text(fieldType) == symbolORGANIZATIONAL && teams == 0 {
		v.warn(n, "map-team", "ORGANIZATIONAL map contains no TEAM")
	}

	for _, rel := range n.Children(fieldRelationships) {
		fields := []string{fieldUpstream, fieldDownstream}

		if rel.Kind == kindPartnership || rel.Kind == kindSharedKernel {
			fields = []string{fieldParticipant1, fieldParticipant2}
		}

		for _, field := range fields {
			if has(rel, field) && !includes(n, fieldBoundedContexts, rel.Text(field)) {
				v.reportError(
					rel,
					"map-membership",
					fmt.Sprintf("Context '%s' is not contained in the map", rel.Text(field)),
				)
			}
		}

		if rel.Text(fields[0]) != "" && rel.Text(fields[0]) == rel.Text(fields[1]) {
			v.reportError(rel, "self-relationship", "A context cannot have a relationship with itself")
		}

		up := v.resolve(rel, fieldUpstream)
		for _, ref := range rel.Fields[fieldUpstreamExposedAggregates] {
			agg := v.resolveValue(rel, ref)
			if up != nil && agg != nil && !contained(agg, up) {
				v.reportError(rel, "exposed-aggregate", "Exposed aggregate must belong to the upstream context")
			}
		}
	}
}

func (v *validator) checkstrategicCustomerSupplierRelationship(n *Node) {
	if includes(n, fieldUpstreamRoles, symbolOHS) {
		v.reportError(n, "customer-supplier-role", "Customer-Supplier cannot use OHS")
	}

	if includes(n, fieldDownstreamRoles, symbolCF) {
		v.reportError(n, "customer-supplier-role", "Customer-Supplier cannot use CF")
	}

	if includes(n, fieldDownstreamRoles, symbolACL) {
		v.warn(n, "customer-supplier-role", "ACL is unusual in a Customer-Supplier relationship")
	}
}

//nolint:cyclop,gocognit,gocyclo // Ветви CML проверяются conformance и corpus-тестами.
func (v *validator) checkstrategicBoundedContext(n *Node) {
	if has(n, fieldRealizedBoundedContexts) && n.Text(fieldType) != symbolTEAM {
		v.reportError(n, "team-realizes", "Only TEAM contexts can realize other bounded contexts")
	}

	domains := map[*Node]bool{}

	for _, ref := range n.Fields[fieldImplementedDomainParts] {
		part := v.resolveValue(n, ref)
		if part != nil && part.Kind == kindDomain {
			domains[part] = true
		}
	}

	if len(domains) > 1 {
		v.warn(n, "multiple-domains", "Bounded context implements multiple domains")
	}

	for _, ref := range n.Fields[fieldImplementedDomainParts] {
		part := v.resolveValue(n, ref)
		if part != nil && part.Kind == kindSubdomain && domains[part.parent] {
			v.reportError(n, "implemented-subdomain", "Subdomain is already implemented through its parent domain")
		}
	}
}

func (v *validator) checkstrategicSubdomain(n *Node) {
	if n.parent != nil && n.Name() == n.parent.Name() {
		v.reportError(n, "subdomain-name", "Subdomain name must differ from its domain name")
	}
}

//nolint:gocognit // Ветви CML проверяются conformance и corpus-тестами.
func (v *validator) checkstrategicAggregate(n *Node) {
	roots, states := 0, 0

	for _, child := range n.Children(fieldDomainObjects) {
		if flag(child, fieldAggregateRoot) {
			roots++
			if roots > 1 {
				v.reportError(child, "aggregate-root", "An aggregate can have only one aggregate root")
			}
		}

		if flag(child, fieldDefinesAggregateLifecycle) {
			states++
			if states > 1 {
				v.reportError(child, "aggregate-lifecycle", "An aggregate can have only one lifecycle enum")
			}
		}
	}

	if owner := v.resolve(n, fieldOwner); owner != nil && owner.Text(fieldType) != symbolTEAM {
		v.reportError(n, "team-owner", "Aggregate owner must be a TEAM context")
	}
}

func (v *validator) checkstrategicSculptorModule(n *Node) {
	if len(n.Children(fieldAggregates)) > 0 && len(n.Children(fieldDomainObjects)) > 0 {
		v.warn(
			n,
			"module-objects",
			"Module mixes aggregates and direct domain objects; generators may ignore objects",
		)
	}
}

//nolint:cyclop,gocognit,gocyclo // Ветви CML проверяются conformance и corpus-тестами.
func (v *validator) checkstrategicCoordinationStep(n *Node) {
	current := ancestor(n, kindBoundedContext)
	target := v.resolve(n, fieldBoundedContext)

	if current != nil && target != nil && !v.reachable(current, target) {
		v.reportError(n, "coordination-context", "Coordination context is not reachable as upstream")
	}

	service := v.resolve(n, fieldService)
	if target != nil && service != nil && service.parent != one(target, fieldApplication) {
		v.reportError(n, "coordination-service", "Coordination service must belong to the target Application")
	}

	operation := v.resolve(n, fieldOperation)
	if service != nil && operation != nil && operation.parent != service {
		v.reportError(n, "coordination-operation", "Coordination operation must belong to the selected service")
	}

	if service != nil && operation != nil {
		count := 0

		for _, op := range service.Children(fieldOperations) {
			if op.Name() == operation.Name() {
				count++
			}
		}

		if count > 1 {
			v.warn(n, "coordination-ambiguous", "Coordination operation is overloaded")
		}
	}
}
