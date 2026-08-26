package query

import (
	"testing"

	"catalogsvc/internal/catalog"
	"catalogsvc/internal/meta"
	"catalogsvc/internal/model"
	"catalogsvc/internal/notify"
)

func TestQueryVersionNilNoPanic(t *testing.T) {
	cat := catalog.New()
	if err := cat.Register(&model.Table{
		ID:     "t1",
		Name:   "orders",
		Fields: []model.Field{{Name: "id", Type: "BIGINT"}},
	}); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	versioner := meta.New(cat, notify.New())
	if _, err := versioner.Draft("t1", model.Schema{Fields: []model.Field{{Name: "id", Type: "BIGINT"}}}); err != nil {
		t.Fatalf("draft failed: %v", err)
	}
	if _, err := versioner.Publish("t1"); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	svc := New(versioner, cat, catalog.NewCache())
	if _, err := svc.GetFields("t1"); err != nil {
		t.Fatalf("normal version read failed: %v", err)
	}
	if err := versioner.SwitchTo("t1", 999); err == nil {
		// The switch was accepted without validation; the subsequent read must
		// return a clear error instead of dereferencing a missing version.
		if _, readErr := svc.GetFields("t1"); readErr == nil {
			t.Fatalf("read during an invalid version switch must return an error")
		}
	}
}
