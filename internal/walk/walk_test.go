package walk

import (
	"testing"

	"catalogsvc/internal/lineage"
)

func chain() *lineage.Graph {
	graph := lineage.New()
	for _, id := range []string{"a", "b", "c", "d"} {
		graph.AddNode(id)
	}
	_ = graph.AddEdge("a", "b", "r")
	_ = graph.AddEdge("b", "c", "r")
	_ = graph.AddEdge("d", "b", "r")
	return graph
}

func TestUpstreamWalk(t *testing.T) {
	graph := chain()
	result := Upstream(graph, "c")
	if len(result) != 3 {
		t.Fatalf("upstream wrong: %v", result)
	}
	cursor := NewUpstreamCursor(graph, "c")
	if cursor.Done() {
		t.Fatalf("cursor must not be done at root")
	}
	first := cursor.NextLevel()
	if len(first) != 1 || first[0] != "b" {
		t.Fatalf("first level wrong: %v", first)
	}
	second := cursor.NextLevel()
	if len(second) != 2 {
		t.Fatalf("second level wrong: %v", second)
	}
	cursor.NextLevel()
	if !cursor.Done() {
		t.Fatalf("cursor should be done")
	}
	if cursor.Stale() {
		t.Fatalf("static graph must not be stale")
	}
	if cursor.VisitedCount() != 4 {
		t.Fatalf("visited count wrong: %d", cursor.VisitedCount())
	}
}

func TestDownstreamWalk(t *testing.T) {
	graph := chain()
	result := Downstream(graph, "a")
	if len(result) != 2 {
		t.Fatalf("downstream wrong: %v", result)
	}
	cursor := NewDownstreamCursor(graph, "a")
	levels := 0
	for !cursor.Done() {
		cursor.NextLevel()
		levels++
	}
	if levels != 3 {
		t.Fatalf("downstream levels wrong: %d", levels)
	}
}

func TestCursorHelpers(t *testing.T) {
	graph := chain()
	cursor := NewUpstreamCursor(graph, "c")
	if frontier := cursor.Frontier(); len(frontier) != 1 || frontier[0] != "c" {
		t.Fatalf("frontier wrong: %v", frontier)
	}
	if depth := DepthOf(cursor, "c"); depth != 0 {
		t.Fatalf("root depth wrong: %d", depth)
	}
	if depth := DepthOf(cursor, "b"); depth != -1 {
		t.Fatalf("unvisited depth wrong: %d", depth)
	}
	_ = cursor.NextLevel()
	if depth := DepthOf(cursor, "b"); depth != 1 {
		t.Fatalf("visited depth wrong: %d", depth)
	}
	rev := ReverseEdges(graph.Adjacency())
	if len(rev["b"]) != 2 {
		t.Fatalf("reverse edges wrong: %+v", rev)
	}
	if !markSeen(map[string]bool{}, "x") {
		t.Fatalf("markSeen should report new")
	}
	next := collectTargets([]string{"p", "p"}, map[string]bool{}, map[string]bool{})
	if len(next) != 1 {
		t.Fatalf("collectTargets failed to dedupe: %v", next)
	}
}
