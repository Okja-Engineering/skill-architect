package skillgate

// STUB — types only, so the per-style tests compile and fail on values.

type NodeKind string

const (
	KindAbsent     NodeKind = "absent"
	KindScalar     NodeKind = "scalar"
	KindMapping    NodeKind = "mapping"
	KindSequence   NodeKind = "sequence"
	KindUnreadable NodeKind = "unreadable"
)

type mapEntry struct {
	key  string
	node *Node
}

// Node is one YAML node.
type Node struct {
	kind    NodeKind
	line    int
	value   string
	entries []mapEntry
	items   []*Node
	reason  string
	src     string
}

func (n *Node) Kind() NodeKind {
	if n == nil {
		return KindAbsent
	}
	return n.kind
}

func (n *Node) Line() int {
	if n == nil {
		return 0
	}
	return n.line
}

func (n *Node) Reason() string {
	if n == nil {
		return ""
	}
	return n.reason
}

func (n *Node) Scalar() (string, bool) {
	if n == nil || n.kind != KindScalar {
		return "", false
	}
	return n.value, true
}

func (n *Node) Get(key string) *Node {
	if n == nil || n.kind != KindMapping {
		return nil
	}
	for _, e := range n.entries {
		if e.key == key {
			return e.node
		}
	}
	return nil
}

func (n *Node) Keys() []string {
	if n == nil || n.kind != KindMapping {
		return nil
	}
	out := make([]string, 0, len(n.entries))
	for _, e := range n.entries {
		out = append(out, e.key)
	}
	return out
}

func (n *Node) Items() []*Node {
	if n == nil || n.kind != KindSequence {
		return nil
	}
	return n.items
}

// Refusal names one construct the reader would not read, and where.
type Refusal struct {
	Path   []string
	Line   int
	Reason string
}

func parseYAMLSubset(raw string, firstLine int) (*Node, []Refusal) {
	return nil, nil
}
