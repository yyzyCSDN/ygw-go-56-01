package lineage

import "testing"

// TestDeleteEdgeClearsTraversalIndexes guards against the regression where
// DeleteEdge dropped the edge record but left the outgoing/incoming lookup
// indexes intact, so upstream/downstream walks kept reaching the deleted edge.
func TestDeleteEdgeClearsTraversalIndexes(t *testing.T) {
	graph := New()
	graph.AddNode("a")
	graph.AddNode("b")
	graph.AddNode("c")
	_ = graph.AddEdge("a", "b", "join")
	_ = graph.AddEdge("b", "c", "join")

	graph.DeleteEdge("b", "c")

	if out := graph.Outgoing("b"); len(out) != 0 {
		t.Fatalf("out index still references deleted edge: %v", out)
	}
	if in := graph.Incoming("c"); len(in) != 0 {
		t.Fatalf("in index still references deleted edge: %v", in)
	}
	adj := graph.Adjacency()
	if targets := adj["b"]; len(targets) != 0 {
		t.Fatalf("adjacency snapshot still reaches deleted edge: %v", targets)
	}

	// Deleting a non-existent edge must stay a safe no-op (idempotent delete).
	graph.DeleteEdge("b", "c")
	if out := graph.Outgoing("b"); len(out) != 0 {
		t.Fatalf("repeated delete mutated out index: %v", out)
	}
}
