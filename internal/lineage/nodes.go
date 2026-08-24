package lineage

import (
	"errors"
	"sort"

	"catalogsvc/internal/model"
)

// ErrBadTransition is returned when a node state change is not allowed.
var ErrBadTransition = errors.New("lineage: node transition not allowed")

// AddNode registers a table as an active lineage node.
func (g *Graph) AddNode(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.nodes[id]; !ok {
		g.nodes[id] = model.NodeActive
	}
}

// MarkUpdating moves a node into the updating state.
func (g *Graph) MarkUpdating(id string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !model.CanTransition(g.nodes[id], model.NodeUpdating) {
		return ErrBadTransition
	}
	g.nodes[id] = model.NodeUpdating
	for key, edge := range g.edges {
		if edge.From == id || edge.To == id {
			edge.State = model.EdgeUpdating
			g.edges[key] = edge
		}
	}
	return nil
}

// MarkActive moves a node back into the active state.
func (g *Graph) MarkActive(id string) error {
	return g.transition(id, model.NodeActive)
}

// MarkRemoved removes a node and all edges incident to it.
func (g *Graph) MarkRemoved(id string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !model.CanTransition(g.nodes[id], model.NodeRemoved) {
		return ErrBadTransition
	}
	g.nodes[id] = model.NodeRemoved
	for key, edge := range g.edges {
		if edge.From == id || edge.To == id {
			edge.State = model.EdgeRemoved
			g.edges[key] = edge
			delete(g.edges, key)
		}
	}
	delete(g.out, id)
	delete(g.in, id)
	return nil
}

// State returns the current lifecycle state of a node.
func (g *Graph) State(id string) (model.NodeState, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	state, ok := g.nodes[id]
	return state, ok
}

// ActiveIDs returns all nodes that have not been removed, sorted.
func (g *Graph) ActiveIDs() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	ids := make([]string, 0, len(g.nodes))
	for id, state := range g.nodes {
		if state != model.NodeRemoved {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func (g *Graph) transition(id string, to model.NodeState) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !model.CanTransition(g.nodes[id], to) {
		return ErrBadTransition
	}
	g.nodes[id] = to
	return nil
}
