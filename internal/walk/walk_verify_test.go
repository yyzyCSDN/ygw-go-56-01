package walk

import (
	"testing"

	"catalogsvc/internal/lineage"
)

func TestLineageWalkUsesLatestGraph(t *testing.T) {
	graph := lineage.New()
	for _, id := range []string{"a", "b", "c"} {
		graph.AddNode(id)
	}
	if err := graph.AddEdge("b", "c", "r"); err != nil {
		t.Fatalf("b->c failed: %v", err)
	}
	cursor := NewUpstreamCursor(graph, "c")
	first := cursor.NextLevel()
	if len(first) != 1 || first[0] != "b" {
		t.Fatalf("first level wrong: %v", first)
	}
	// The graph is updated while the cursor is still walking.
	if err := graph.AddEdge("a", "b", "r"); err != nil {
		t.Fatalf("a->b failed: %v", err)
	}
	second := cursor.NextLevel()
	if len(second) != 1 || second[0] != "a" {
		t.Fatalf("walk dropped the layer added mid-walk: %v", second)
	}
}
