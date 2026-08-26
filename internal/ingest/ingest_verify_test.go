package ingest

import (
	"errors"
	"testing"

	"catalogsvc/internal/catalog"
	"catalogsvc/internal/meta"
	"catalogsvc/internal/model"
	"catalogsvc/internal/notify"
)

func TestImportRollbackCoversAllSegments(t *testing.T) {
	cat := catalog.New()
	versioner := meta.New(cat, notify.New())
	importer := New(cat, versioner, notify.New())
	if err := importer.Begin(); err != nil {
		t.Fatalf("begin failed: %v", err)
	}
	committed := Segment{
		ID:     "seg-a",
		Commit: true,
		Tables: []model.Table{{ID: "t1", Name: "orders", Fields: []model.Field{{Name: "id", Type: "BIGINT"}}}},
	}
	if err := importer.Apply(committed); err != nil {
		t.Fatalf("apply committed failed: %v", err)
	}
	inflight := Segment{
		ID:     "seg-b",
		Tables: []model.Table{{ID: "t2", Name: "customers"}},
	}
	if err := importer.Apply(inflight); err != nil {
		t.Fatalf("apply in-flight failed: %v", err)
	}
	importer.Fail(errors.New("upstream aborted mid-batch"))
	if err := importer.Rollback(); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}
	if cat.HasTable("t1") {
		t.Fatalf("committed segment not rolled back")
	}
	if cat.HasTable("t2") {
		t.Fatalf("in-flight segment was left behind by rollback")
	}
}
