package meta

import (
	"testing"

	"catalogsvc/internal/catalog"
	"catalogsvc/internal/model"
	"catalogsvc/internal/notify"
)

func TestMetaUpdateAtomicReplace(t *testing.T) {
	cat := catalog.New()
	if err := cat.Register(&model.Table{
		ID:     "t1",
		Name:   "orders",
		Fields: []model.Field{{Name: "a", Type: "TEXT"}, {Name: "b", Type: "TEXT"}, {Name: "c", Type: "TEXT"}},
	}); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	versioner := New(cat, notify.New())
	schema := model.Schema{Fields: []model.Field{{Name: "x", Type: "BIGINT"}, {Name: "y", Type: "BIGINT"}}}
	if err := versioner.ApplySchema("t1", schema); err != nil {
		t.Fatalf("apply schema failed: %v", err)
	}
	table, err := cat.Get("t1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if len(table.Fields) != 2 {
		t.Fatalf("stale fields survived the update: %v", table.Fields)
	}
	if table.Fields[0].Name != "x" || table.Fields[1].Name != "y" {
		t.Fatalf("field list is a mix of old and new: %v", table.Fields)
	}
}
