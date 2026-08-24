package walk

import "catalogsvc/internal/lineage"

// Cursor walks a lineage graph level by level from a root table.
type Cursor struct {
	g        *lineage.Graph
	root     string
	upstream bool
	frontier []string
	visited  map[string]bool
	adj      map[string][]string
	fp       uint64
}

// NewUpstreamCursor starts an upstream (dependency) walk at root.
func NewUpstreamCursor(g *lineage.Graph, root string) *Cursor {
	return newCursor(g, root, true)
}

// NewDownstreamCursor starts a downstream (dependent) walk at root.
func NewDownstreamCursor(g *lineage.Graph, root string) *Cursor {
	return newCursor(g, root, false)
}

func newCursor(g *lineage.Graph, root string, upstream bool) *Cursor {
	return &Cursor{
		g:        g,
		root:     root,
		upstream: upstream,
		frontier: []string{root},
		visited:  map[string]bool{root: true},
		adj:      g.Adjacency(),
		fp:       g.Fingerprint(),
	}
}

// NextLevel returns the next layer of tables reachable from the frontier. The
// adjacency is refreshed against the live graph before each step so a graph
// updated mid-walk is still reflected in later levels.
func (c *Cursor) NextLevel() []string {
	c.adj = c.g.Adjacency()
	return c.advance(c.adj)
}

// Done reports whether the walk has exhausted every reachable table.
func (c *Cursor) Done() bool {
	return len(c.frontier) == 0
}

// Stale reports whether the graph changed since the cursor was created.
func (c *Cursor) Stale() bool {
	return c.fp != c.g.Fingerprint()
}

// Upstream returns every table reachable upstream from root, in breadth order.
func Upstream(g *lineage.Graph, root string) []string {
	cursor := NewUpstreamCursor(g, root)
	return drain(cursor)
}

// Downstream returns every table reachable downstream from root.
func Downstream(g *lineage.Graph, root string) []string {
	cursor := NewDownstreamCursor(g, root)
	return drain(cursor)
}

func drain(cursor *Cursor) []string {
	var result []string
	for !cursor.Done() {
		result = append(result, cursor.NextLevel()...)
	}
	return result
}
