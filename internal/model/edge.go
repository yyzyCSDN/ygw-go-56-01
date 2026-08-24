package model

// EdgeState tracks the lifecycle of a lineage edge.
type EdgeState string

const (
	EdgeActive  EdgeState = "active"
	EdgeUpdating EdgeState = "updating"
	EdgeRemoved EdgeState = "removed"
)

// LineageEdge records a dependency from one table to another.
type LineageEdge struct {
	From   string
	To     string
	State  EdgeState
	Reason string
}

// Key returns the canonical map key for the edge.
func (e LineageEdge) Key() string {
	return e.From + "|" + e.To
}

// Clone returns a copy of the edge with an explicit state.
func (e LineageEdge) Clone(state EdgeState) LineageEdge {
	return LineageEdge{
		From:   e.From,
		To:     e.To,
		State:  state,
		Reason: e.Reason,
	}
}

// NodeState tracks the lifecycle of a lineage node.
type NodeState string

const (
	NodeActive  NodeState = "active"
	NodeUpdating NodeState = "updating"
	NodeRemoved NodeState = "removed"
)

// Node describes a table participating in lineage.
type Node struct {
	TableID string
	State   NodeState
}

// CanTransition reports whether a node may move from one state to another.
func CanTransition(from, to NodeState) bool {
	switch from {
	case NodeActive:
		return to == NodeUpdating || to == NodeRemoved
	case NodeUpdating:
		return to == NodeActive || to == NodeRemoved
	case NodeRemoved:
		return false
	default:
		return false
	}
}
