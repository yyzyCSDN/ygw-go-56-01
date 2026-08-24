package lineage

// HasCycle reports whether the adjacency map contains a directed cycle.
func HasCycle(adj map[string][]string) bool {
	visiting := make(map[string]bool)
	visited := make(map[string]bool)
	ids := make([]string, 0, len(adj))
	for id := range adj {
		ids = append(ids, id)
	}
	sortStrings(ids)
	for _, id := range ids {
		if !visited[id] && visit(id, adj, visiting, visited) {
			return true
		}
	}
	return false
}

// WouldCreateCycle checks whether adding from->to to the adjacency would
// introduce a cycle, without mutating the input map.
func WouldCreateCycle(adj map[string][]string, from, to string) bool {
	next := make(map[string][]string, len(adj)+1)
	for id, targets := range adj {
		next[id] = append([]string(nil), targets...)
	}
	next[from] = append(next[from], to)
	return HasCycle(next)
}

func visit(id string, adj map[string][]string, visiting, visited map[string]bool) bool {
	if visiting[id] {
		return true
	}
	if visited[id] {
		return false
	}
	visiting[id] = true
	for _, target := range adj[id] {
		if visit(target, adj, visiting, visited) {
			return true
		}
	}
	visiting[id] = false
	visited[id] = true
	return false
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
