package ingest

import (
	"testing"

	"catalogsvc/internal/catalog"
	"catalogsvc/internal/meta"
	"catalogsvc/internal/model"
	"catalogsvc/internal/notify"
)

func TestImportMergeKeepsIncrementalFields(t *testing.T) {
	cat := catalog.New()
	if err := cat.Register(&model.Table{
		ID:     "t1",
		Name:   "orders",
		Fields: []model.Field{{Name: "manual", Type: "TEXT"}, {Name: "owner_note", Type: "TEXT"}},
	}); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	versioner := meta.New(cat, notify.New())
	importer := New(cat, versioner, notify.New())
	if err := importer.Begin(); err != nil {
		t.Fatalf("begin failed: %v", err)
	}
	segment := Segment{
		ID:     "sync-1",
		Commit: true,
		Tables: []model.Table{{
			ID:     "t1",
			Name:   "orders",
			Fields: []model.Field{{Name: "order_id", Type: "BIGINT"}, {Name: "amount", Type: "DOUBLE"}},
		}},
	}
	if err := importer.Apply(segment); err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	table, err := cat.Get("t1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if _, ok := table.Field("manual"); !ok {
		t.Fatalf("manually added field was overwritten by the sync: %v", table.Fields)
	}
	if _, ok := table.Field("owner_note"); !ok {
		t.Fatalf("incremental local field lost: %v", table.Fields)
	}
	if _, ok := table.Field("order_id"); !ok {
		t.Fatalf("imported field missing: %v", table.Fields)
	}
}
