package lineage

import (
	"errors"
	"sort"
	"sync"

	"catalogsvc/internal/model"
)

// ErrUnknownNode is returned when an edge references an unregistered table.
var ErrUnknownNode = errors.New("lineage: unknown node")

// ErrCycle is returned when adding an edge would close a dependency cycle.
var ErrCycle = errors.New("lineage: cycle detected")

// Graph stores lineage nodes and dependency edges with lookup indexes.
type Graph struct {
	mu    sync.RWMutex
	nodes map[string]model.NodeState
	edges map[string]model.LineageEdge
	out   map[string][]string
	in    map[string][]string
}

// New creates an empty lineage graph.
func New() *Graph {
	return &Graph{
		nodes: make(map[string]model.NodeState),
		edges: make(map[string]model.LineageEdge),
		out:   make(map[string][]string),
		in:    make(map[string][]string),
	}
}

// AddEdge records a dependency from one table to another. The edge is applied
// first and then the live graph is checked for cycles; a cycle rolls the edge
// back and reports ErrCycle.
func (g *Graph) AddEdge(from, to, reason string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.nodes[from] == model.NodeRemoved || g.nodes[to] == model.NodeRemoved {
		return ErrUnknownNode
	}
	if _, ok := g.nodes[from]; !ok {
		return ErrUnknownNode
	}
	if _, ok := g.nodes[to]; !ok {
		return ErrUnknownNode
	}
	edge := model.LineageEdge{
		From:   from,
		To:     to,
		State:  model.EdgeActive,
		Reason: reason,
	}
	key := edge.Key()
	if _, exists := g.edges[key]; exists {
		g.edges[key] = edge
		return nil
	}
	g.edges[key] = edge
	g.out[from] = append(g.out[from], to)
	g.in[to] = append(g.in[to], from)
	if HasCycle(g.adjacencyCopy()) {
		delete(g.edges, key)
		g.out[from] = removeValue(g.out[from], to)
		g.in[to] = removeValue(g.in[to], from)
		return ErrCycle
	}
	return nil
}

// DeleteEdge removes a dependency edge and clears both lookup indexes.
func (g *Graph) DeleteEdge(from, to string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	key := (model.LineageEdge{From: from, To: to}).Key()
	if _, ok := g.edges[key]; !ok {
		return
	}
	delete(g.edges, key)
	g.out[from] = removeValue(g.out[from], to)
	g.in[to] = removeValue(g.in[to], from)
}

// Edge returns a stored edge by its endpoints.
func (g *Graph) Edge(from, to string) (model.LineageEdge, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	edge, ok := g.edges[(model.LineageEdge{From: from, To: to}).Key()]
	return edge, ok
}

// Edges returns every active edge sorted by key.
func (g *Graph) Edges() []model.LineageEdge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	keys := make([]string, 0, len(g.edges))
	for key := range g.edges {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	edges := make([]model.LineageEdge, 0, len(keys))
	for _, key := range keys {
		edges = append(edges, g.edges[key])
	}
	return edges
}

// Outgoing returns tables that the given table depends on.
func (g *Graph) Outgoing(id string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return append([]string(nil), g.out[id]...)
}

// Incoming returns tables that depend on the given table.
func (g *Graph) Incoming(id string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return append([]string(nil), g.in[id]...)
}

// Adjacency returns a live copy of the out-edge index.
func (g *Graph) Adjacency() map[string][]string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.adjacencyCopy()
}

// adjacencyCopy snapshots the out index without locking.
func (g *Graph) adjacencyCopy() map[string][]string {
	adj := make(map[string][]string, len(g.out))
	for id, targets := range g.out {
		adj[id] = append([]string(nil), targets...)
	}
	return adj
}

func removeValue(values []string, target string) []string {
	for i, value := range values {
		if value == target {
			return append(values[:i], values[i+1:]...)
		}
	}
	return values
}
