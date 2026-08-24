package lineage

import "testing"

func TestDeleteLineageEdgeClearsIndex(t *testing.T) {
	graph := New()
	for _, id := range []string{"a", "b", "c"} {
		graph.AddNode(id)
	}
	if err := graph.AddEdge("a", "b", "join"); err != nil {
		t.Fatalf("add edge failed: %v", err)
	}
	if err := graph.AddEdge("b", "c", "rollup"); err != nil {
		t.Fatalf("add edge failed: %v", err)
	}
	graph.DeleteEdge("a", "b")
	if got := graph.Outgoing("a"); len(got) != 0 {
		t.Fatalf("deleted edge still reachable through the index: %v", got)
	}
	if got := graph.Incoming("b"); len(got) != 0 {
		t.Fatalf("incoming index wrong after delete: %v", got)
	}
	if _, ok := graph.Edge("a", "b"); ok {
		t.Fatalf("edge map still contains deleted edge")
	}
}
