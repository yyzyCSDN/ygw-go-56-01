package walk

// advance computes the next frontier from the supplied adjacency snapshot.
func (c *Cursor) advance(adj map[string][]string) []string {
	rev := ReverseEdges(adj)
	var next []string
	seen := make(map[string]bool)
	for _, id := range c.frontier {
		targets := adj[id]
		if c.upstream {
			targets = rev[id]
		}
		next = append(next, collectTargets(targets, c.visited, seen)...)
	}
	for _, id := range next {
		markSeen(c.visited, id)
	}
	c.frontier = next
	return next
}

// Frontier exposes the current pending level for diagnostics.
func (c *Cursor) Frontier() []string {
	return append([]string(nil), c.frontier...)
}

// VisitedCount reports how many tables the walk has reached.
func (c *Cursor) VisitedCount() int {
	return len(c.visited)
}

// ReverseEdges builds the inverse adjacency map.
func ReverseEdges(adj map[string][]string) map[string][]string {
	rev := make(map[string][]string)
	for from, targets := range adj {
		for _, to := range targets {
			rev[to] = append(rev[to], from)
		}
	}
	return rev
}
