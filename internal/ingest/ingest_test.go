package ingest

import (
	"testing"

	"catalogsvc/internal/catalog"
	"catalogsvc/internal/meta"
	"catalogsvc/internal/model"
	"catalogsvc/internal/notify"
)

func newImporter(t *testing.T) (*Importer, *catalog.Catalog) {
	t.Helper()
	cat := catalog.New()
	versioner := meta.New(cat, notify.New())
	return New(cat, versioner, notify.New()), cat
}

func TestImportNewTablesAndCommit(t *testing.T) {
	importer, cat := newImporter(t)
	if err := importer.Begin(); err != nil {
		t.Fatalf("begin failed: %v", err)
	}
	first := Segment{ID: "seg-1", Commit: true, Tables: []model.Table{{ID: "t1", Name: "orders", Fields: []model.Field{{Name: "id", Type: "BIGINT"}}}}}
	if err := importer.Apply(first); err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	second := Segment{ID: "seg-2", Tables: []model.Table{{ID: "t2", Name: "customers"}}}
	if err := importer.Apply(second); err != nil {
		t.Fatalf("apply in-flight failed: %v", err)
	}
	if !importer.HasInflight() {
		t.Fatalf("in-flight segment not tracked")
	}
	if err := importer.Commit(); err != nil {
		t.Fatalf("commit failed: %v", err)
	}
	if importer.HasInflight() {
		t.Fatalf("in-flight segment still present after commit")
	}
	if written := importer.WrittenTables(); len(written) != 2 {
		t.Fatalf("written tables wrong: %v", written)
	}
	if ids := SegmentIDs([]Segment{first, second}); len(ids) != 2 {
		t.Fatalf("segment ids wrong: %v", ids)
	}
	if SegmentSize(first) != 1 || ImportedCount(map[string]bool{"t1": true, "t2": true}) != 2 {
		t.Fatalf("segment helpers wrong")
	}
	if !cat.HasTable("t1") || !cat.HasTable("t2") {
		t.Fatalf("imported tables missing")
	}
}

func TestRollbackRestoresModifiedTable(t *testing.T) {
	importer, cat := newImporter(t)
	if err := cat.Register(&model.Table{ID: "t1", Name: "orders", Fields: []model.Field{{Name: "before", Type: "TEXT"}}}); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if err := importer.Begin(); err != nil {
		t.Fatalf("begin failed: %v", err)
	}
	if err := importer.Apply(Segment{ID: "seg-1", Commit: true, Tables: []model.Table{{ID: "t1", Name: "orders", Fields: []model.Field{{Name: "after", Type: "BIGINT"}}}}}); err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if err := importer.Rollback(); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}
	table, err := cat.Get("t1")
	if err != nil {
		t.Fatalf("table missing after rollback: %v", err)
	}
	if len(table.Fields) != 1 || table.Fields[0].Name != "before" {
		t.Fatalf("rollback did not restore the pre-import schema: %+v", table.Fields)
	}
}

func TestFailureState(t *testing.T) {
	importer, _ := newImporter(t)
	if err := importer.Begin(); err != nil {
		t.Fatalf("begin failed: %v", err)
	}
	importer.Fail(nil)
	if !importer.Failed() || importer.LastError() == nil {
		t.Fatalf("failure state not recorded")
	}
	if err := importer.Apply(Segment{Tables: []model.Table{{ID: "t1", Name: "t1"}}}); err == nil {
		t.Fatalf("apply after failure must error")
	}
}

func TestRollbackScopeHelpers(t *testing.T) {
	importer, cat := newImporter(t)
	cat.Register(&model.Table{ID: "base", Name: "base"})
	baseline := cat.Snapshot()
	written := map[string]bool{"new": true}
	if PreservedCount(baseline, written) != 1 {
		t.Fatalf("preserved count wrong")
	}
	if ResidueCount(map[string]*model.Table{"new": nil}, baseline) != 1 {
		t.Fatalf("residue count wrong")
	}
	if err := importer.Begin(); err != nil {
		t.Fatalf("begin failed: %v", err)
	}
	importer.baseline = baseline
	if preserved, residue := importer.RollbackScope(); preserved != 1 || residue != 0 {
		t.Fatalf("rollback scope wrong: %d %d", preserved, residue)
	}
}
