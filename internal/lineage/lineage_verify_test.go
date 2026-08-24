package lineage

import "testing"

func TestCycleDetectUsesLatestGraph(t *testing.T) {
	graph := New()
	for _, id := range []string{"a", "b", "c"} {
		graph.AddNode(id)
	}
	if err := graph.AddEdge("a", "b", "r"); err != nil {
		t.Fatalf("a->b failed: %v", err)
	}
	if err := graph.AddEdge("b", "c", "r"); err != nil {
		t.Fatalf("b->c failed: %v", err)
	}
	err := graph.AddEdge("c", "a", "r")
	if err == nil {
		t.Fatalf("cycle formed by the latest edge was not detected")
	}
	if _, ok := graph.Edge("c", "a"); ok {
		t.Fatalf("cyclic edge remained in the graph")
	}
}
