package cmlgo

// Position is a one-based Unicode line/column and a zero-based byte offset.
type Position struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Offset int    `json:"offset"`
}

// Diagnostic describes an error or warning at a source location.
type Diagnostic struct {
	Severity string   `json:"severity"`
	Code     string   `json:"code"`
	Message  string   `json:"message"`
	Position Position `json:"position"`
}

// ASTValue is a scalar, a contained AST node, or an unresolved typed reference.
type ASTValue struct {
	Text     string   `json:"text,omitempty"`
	Node     *Node    `json:"node,omitempty"`
	RefType  string   `json:"ref_type,omitempty"`
	Position Position `json:"position"`
}

// Node is an AST object; field names follow the upstream Xtext assignments.
type Node struct {
	Kind     string                `json:"kind"`
	Fields   map[string][]ASTValue `json:"fields"`
	Position Position              `json:"position"`
	parent   *Node
}

// Name returns the node's declared name, or an empty string.
func (n *Node) Name() string { return n.Text(fieldName) }

// Text returns the first scalar value of a field.
func (n *Node) Text(field string) string {
	if n == nil || len(n.Fields[field]) == 0 {
		return ""
	}

	return n.Fields[field][0].Text
}

// Children returns the directly contained nodes of a field.
func (n *Node) Children(field string) []*Node {
	var out []*Node

	if n != nil {
		for _, v := range n.Fields[field] {
			if v.Node != nil {
				out = append(out, v.Node)
			}
		}
	}

	return out
}

// Result contains the root model and ordered diagnostics.
type Result struct {
	Valid       bool         `json:"valid"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Documents   []*Node      `json:"-"`
	Model       *Node        `json:"-"`
}
