package meta

import (
	"testing"

	"catalogsvc/internal/catalog"
	"catalogsvc/internal/model"
	"catalogsvc/internal/notify"
)

func TestParseUsesPublishedVersion(t *testing.T) {
	cat := catalog.New()
	if err := cat.Register(&model.Table{
		ID:     "t1",
		Name:   "orders",
		Fields: []model.Field{{Name: "id", Type: "BIGINT"}},
	}); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	versioner := New(cat, notify.New())
	if _, err := versioner.Draft("t1", model.Schema{Fields: []model.Field{{Name: "id", Type: "BIGINT"}}}); err != nil {
		t.Fatalf("first draft failed: %v", err)
	}
	if _, err := versioner.Publish("t1"); err != nil {
		t.Fatalf("first publish failed: %v", err)
	}
	first, err := versioner.Parse("t1")
	if err != nil || len(first.Fields) != 1 {
		t.Fatalf("first parse failed: %v %+v", err, first)
	}
	if _, err := versioner.Draft("t1", model.Schema{Fields: []model.Field{{Name: "id", Type: "BIGINT"}, {Name: "amount", Type: "DOUBLE"}}}); err != nil {
		t.Fatalf("second draft failed: %v", err)
	}
	if _, err := versioner.Publish("t1"); err != nil {
		t.Fatalf("second publish failed: %v", err)
	}
	second, err := versioner.Parse("t1")
	if err != nil {
		t.Fatalf("second parse failed: %v", err)
	}
	if len(second.Fields) != 2 {
		t.Fatalf("parse still uses the old version structure: %v", second.Fields)
	}
	if _, ok := ParseField(*second, "amount"); !ok {
		t.Fatalf("new column missing from parsed schema: %v", second.Fields)
	}
}
