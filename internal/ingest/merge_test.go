package ingest

import (
	"testing"

	"catalogsvc/internal/model"
)

// TestApplyPreservesManualColumns reproduces the reported bug: a manually
// cataloged column must survive an upstream sync whose source does not mention
// it. The sync may only overwrite columns present in the source; every other
// locally owned column is left untouched (fields are not lost).
func TestApplyPreservesManualColumns(t *testing.T) {
	importer, cat := newImporter(t)
	if err := cat.Register(&model.Table{
		ID:   "t1",
		Name: "orders",
		Fields: []model.Field{
			{Name: "id", Type: "TEXT"},     // upstream will correct the type
			{Name: "amount", Type: "DOUBLE"},
			{Name: "audit_note", Type: "TEXT"}, // manually cataloged, unknown upstream
		},
	}); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	if err := importer.Begin(); err != nil {
		t.Fatalf("begin failed: %v", err)
	}
	// Upstream adds "status" and redefines "id"; it does not carry "audit_note".
	if err := importer.Apply(Segment{ID: "seg-1", Commit: true, Tables: []model.Table{{
		ID:   "t1",
		Name: "orders",
		Fields: []model.Field{
			{Name: "id", Type: "BIGINT"},
			{Name: "amount", Type: "DOUBLE"},
			{Name: "status", Type: "TEXT"},
		},
	}}}); err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	got, err := cat.Get("t1")
	if err != nil {
		t.Fatalf("table missing: %v", err)
	}

	// No field is lost: the union of upstream + local-only columns survives.
	wantPresent := []string{"id", "amount", "status", "audit_note"}
	for _, name := range wantPresent {
		if _, ok := got.Field(name); !ok {
			t.Fatalf("field %q lost after sync (fields: %v)", name, got.FieldNames())
		}
	}

	// Incoming definitions win on name collisions.
	if f, ok := got.Field("id"); !ok || f.Type != "BIGINT" {
		t.Fatalf("upstream type for id not applied: %+v", f)
	}
	// Manually cataloged column keeps its original definition.
	if f, ok := got.Field("audit_note"); !ok || f.Type != "TEXT" {
		t.Fatalf("manual column altered: %+v", f)
	}
}
