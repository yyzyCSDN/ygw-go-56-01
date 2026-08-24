package lineage

import "testing"

func TestAddEdgeAndLookup(t *testing.T) {
	graph := New()
	graph.AddNode("a")
	graph.AddNode("b")
	if err := graph.AddEdge("a", "b", "join"); err != nil {
		t.Fatalf("add edge failed: %v", err)
	}
	if edge, ok := graph.Edge("a", "b"); !ok || edge.Reason != "join" {
		t.Fatalf("edge lookup failed: %+v", edge)
	}
	if outgoing := graph.Outgoing("a"); len(outgoing) != 1 || outgoing[0] != "b" {
		t.Fatalf("outgoing wrong: %v", outgoing)
	}
	if incoming := graph.Incoming("b"); len(incoming) != 1 || incoming[0] != "a" {
		t.Fatalf("incoming wrong: %v", incoming)
	}
	if err := graph.AddEdge("a", "b", "again"); err != nil {
		t.Fatalf("duplicate edge should be idempotent: %v", err)
	}
	if edges := graph.Edges(); len(edges) != 1 {
		t.Fatalf("duplicate edge created two entries: %v", edges)
	}
}

func TestDeleteEdgeRemovesEdge(t *testing.T) {
	graph := New()
	graph.AddNode("a")
	graph.AddNode("b")
	_ = graph.AddEdge("a", "b", "join")
	graph.DeleteEdge("a", "b")
	if _, ok := graph.Edge("a", "b"); ok {
		t.Fatalf("edge still present after delete")
	}
	if edges := graph.Edges(); len(edges) != 0 {
		t.Fatalf("edges list still contains deleted edge")
	}
	graph.DeleteEdge("a", "b")
}

func TestNodeStates(t *testing.T) {
	graph := New()
	graph.AddNode("a")
	if err := graph.MarkUpdating("a"); err != nil {
		t.Fatalf("mark updating failed: %v", err)
	}
	if err := graph.MarkActive("a"); err != nil {
		t.Fatalf("mark active failed: %v", err)
	}
	if err := graph.MarkRemoved("a"); err != nil {
		t.Fatalf("mark removed failed: %v", err)
	}
	if state, _ := graph.State("a"); state != "removed" {
		t.Fatalf("state wrong: %s", state)
	}
	if ids := graph.ActiveIDs(); len(ids) != 0 {
		t.Fatalf("removed node still active: %v", ids)
	}
	if err := graph.MarkRemoved("a"); err == nil {
		t.Fatalf("removed node must reject further transitions")
	}
}

func TestNonCyclicAddChain(t *testing.T) {
	graph := New()
	for _, id := range []string{"a", "b", "c"} {
		graph.AddNode(id)
	}
	if err := graph.AddEdge("a", "b", "x"); err != nil {
		t.Fatalf("edge a->b failed: %v", err)
	}
	if err := graph.AddEdge("b", "c", "x"); err != nil {
		t.Fatalf("edge b->c failed: %v", err)
	}
	if HasCycle(graph.Adjacency()) {
		t.Fatalf("chain must not contain a cycle")
	}
}

func TestWouldCreateCycleHelper(t *testing.T) {
	adj := map[string][]string{"a": {"b"}, "b": {"c"}}
	if WouldCreateCycle(adj, "c", "a") == false {
		t.Fatalf("c->a must create a cycle")
	}
	if WouldCreateCycle(adj, "a", "c") == true {
		t.Fatalf("a->c must stay acyclic")
	}
}

func TestHashesAndFingerprint(t *testing.T) {
	graph := New()
	graph.AddNode("a")
	graph.AddNode("b")
	_ = graph.AddEdge("a", "b", "r")
	if HashEdgeKey("a", "b") == HashEdgeKey("b", "a") {
		t.Fatalf("directed hash collision")
	}
	if EdgeDigest(graph.Edges()) != graph.Fingerprint() {
		t.Fatalf("digest mismatch")
	}
}
