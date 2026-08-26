package model

import "testing"

func TestTableCloneAndFields(t *testing.T) {
	table := &Table{
		ID:     "t1",
		Name:   "orders",
		Schema: "ods",
		Owner:  "data-team",
		Fields: []Field{{Name: "id", Type: "BIGINT", Primary: true}, {Name: "amount", Type: "DOUBLE"}},
	}
	clone := table.Clone()
	clone.Fields[0].Name = "mutated"
	if table.Fields[0].Name != "id" {
		t.Fatalf("clone shares field storage: %v", table.Fields[0].Name)
	}
	if field, ok := table.Field("amount"); !ok || field.Type != "DOUBLE" {
		t.Fatalf("field lookup failed: %+v", field)
	}
	if names := table.FieldNames(); len(names) != 2 || names[0] != "id" {
		t.Fatalf("unexpected field names: %v", names)
	}
	withSchema := table.WithSchema([]Field{{Name: "only", Type: "TEXT"}})
	if len(withSchema.Fields) != 1 || len(table.Fields) != 2 {
		t.Fatalf("WithSchema mutated the receiver")
	}
	withMeta := table.WithMeta("new-name", "dwd", "other")
	if table.Name == "new-name" || withMeta.Owner != "other" {
		t.Fatalf("WithMeta mutated the receiver")
	}
	if EqualFields(table.Fields, clone.Fields) {
		t.Fatalf("clone shares field storage")
	}
}

func TestFieldHelpers(t *testing.T) {
	fields := []Field{
		NewField("id", "INTEGER", false, true),
		NewField("note", "STRING", true, false),
	}
	if fields[0].Type != "BIGINT" || fields[1].Type != "TEXT" {
		t.Fatalf("type normalization failed: %+v", fields)
	}
	set := BuildFieldSet(fields)
	if _, ok := set["note"]; !ok {
		t.Fatalf("field set missing note")
	}
	next := BuildFieldSet([]Field{{Name: "extra", Type: "TEXT"}})
	if added := next.Added(set); len(added) != 1 || added[0] != "extra" {
		t.Fatalf("added diff wrong: %v", added)
	}
	if removed := next.Removed(set); len(removed) != 2 {
		t.Fatalf("removed diff wrong: %v", removed)
	}
	merged := MergePreserving(fields, []Field{{Name: "id", Type: "BIGINT"}})
	if len(merged) != 2 {
		t.Fatalf("merge preserving produced %d fields", len(merged))
	}
}

func TestEdgeAndNodeStates(t *testing.T) {
	edge := LineageEdge{From: "a", To: "b", State: EdgeActive, Reason: "join"}
	if edge.Key() != "a|b" {
		t.Fatalf("edge key wrong: %s", edge.Key())
	}
	clone := edge.Clone(EdgeUpdating)
	if clone.State != EdgeUpdating || edge.State != EdgeActive {
		t.Fatalf("edge clone mutated original")
	}
	if !CanTransition(NodeActive, NodeUpdating) || CanTransition(NodeRemoved, NodeActive) {
		t.Fatalf("node transition rules wrong")
	}
}

func TestVersionStateMachine(t *testing.T) {
	version := &Version{
		TableID: "t1",
		Number:  1,
		State:   VersionDraft,
		Schema:  Schema{Fields: []Field{{Name: "a", Type: "TEXT"}}},
	}
	clone := version.Clone()
	clone.Schema.Fields[0].Name = "b"
	if version.Schema.Fields[0].Name != "a" {
		t.Fatalf("version clone shares schema storage")
	}
	if version.IsReadable() {
		t.Fatalf("draft must not be readable")
	}
	if !StateTransitionAllowed(VersionDraft, VersionPublished) || StateTransitionAllowed(VersionSuperseded, VersionPublished) {
		t.Fatalf("version transitions wrong")
	}
	published := version.Clone()
	published.State = VersionPublished
	latest := LatestVersion([]*Version{version, published})
	if latest == nil || latest.Number != 1 || latest.State != VersionPublished {
		t.Fatalf("latest version wrong: %+v", latest)
	}
}

func TestEventKey(t *testing.T) {
	event := ChangeEvent{TableID: "t9", Version: 7, Type: EventSchemaChanged}
	if event.Key() != "schema.changed:t9:7" {
		t.Fatalf("event key wrong: %s", event.Key())
	}
}
