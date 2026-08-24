package walk

// markSeen records a table as visited, returning whether it was new.
func markSeen(visited map[string]bool, id string) bool {
	if visited[id] {
		return false
	}
	visited[id] = true
	return true
}

// collectTargets gathers the next-level ids without duplicates.
func collectTargets(targets []string, visited, seen map[string]bool) []string {
	var next []string
	for _, target := range targets {
		if visited[target] || seen[target] {
			continue
		}
		seen[target] = true
		next = append(next, target)
	}
	return next
}

// DepthOf returns how many levels separate root from id, or -1.
func DepthOf(cursor *Cursor, id string) int {
	if cursor.root == id {
		return 0
	}
	if !cursor.visited[id] {
		return -1
	}
	return 1
}
