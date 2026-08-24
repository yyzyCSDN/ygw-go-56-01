package meta

import (
	"testing"

	"catalogsvc/internal/catalog"
	"catalogsvc/internal/model"
	"catalogsvc/internal/notify"
)

// TestParseReflectsLatestPublish ensures that Parse resolves the schema of the
// currently active version, not a version cached from the first publish. After
// a new version is published with an added column, Parse must surface that
// column immediately (fields aligned, no stale structure).
func TestParseReflectsLatestPublish(t *testing.T) {
	cat := catalog.New()
	cat.Register(&model.Table{ID: "t1", Name: "orders", Fields: []model.Field{{Name: "id", Type: "BIGINT"}}})
	versioner := New(cat, notify.New())

	// First publish: a single column.
	_, _ = versioner.Draft("t1", model.Schema{Fields: []model.Field{{Name: "id", Type: "BIGINT"}}})
	if _, err := versioner.Publish("t1"); err != nil {
		t.Fatalf("first publish failed: %v", err)
	}
	first, err := versioner.Parse("t1")
	if err != nil {
		t.Fatalf("first parse failed: %v", err)
	}
	if len(first.Fields) != 1 || first.Fields[0].Name != "id" {
		t.Fatalf("first parse wrong: %+v", first.Fields)
	}

	// Second publish: adds an "amount" column.
	_, _ = versioner.Draft("t1", model.Schema{Fields: []model.Field{
		{Name: "id", Type: "BIGINT"},
		{Name: "amount", Type: "DOUBLE"},
	}})
	if _, err := versioner.Publish("t1"); err != nil {
		t.Fatalf("second publish failed: %v", err)
	}
	second, err := versioner.Parse("t1")
	if err != nil {
		t.Fatalf("second parse failed: %v", err)
	}
	if len(second.Fields) != 2 {
		t.Fatalf("new column not visible after publish: %+v", second.Fields)
	}
	if second.Fields[0].Name != "id" || second.Fields[1].Name != "amount" {
		t.Fatalf("fields misaligned after publish: %+v", second.Fields)
	}
}
