package meta

import (
	"testing"

	"catalogsvc/internal/catalog"
	"catalogsvc/internal/model"
	"catalogsvc/internal/notify"
)

func newFixture(t *testing.T) (*Meta, *catalog.Catalog) {
	t.Helper()
	cat := catalog.New()
	cat.Register(&model.Table{ID: "t1", Name: "orders", Schema: "ods", Fields: []model.Field{{Name: "id", Type: "BIGINT"}}})
	versioner := New(cat, notify.New())
	return versioner, cat
}

func TestDraftPublishHistory(t *testing.T) {
	versioner, _ := newFixture(t)
	first, err := versioner.Draft("t1", model.Schema{Fields: []model.Field{{Name: "id", Type: "BIGINT"}}})
	if err != nil || first.Number != 1 || first.State != model.VersionDraft {
		t.Fatalf("draft failed: %v %+v", err, first)
	}
	published, err := versioner.Publish("t1")
	if err != nil || published.State != model.VersionPublished {
		t.Fatalf("publish failed: %v %+v", err, published)
	}
	if versioner.LatestNumber("t1") != 1 || versioner.Count("t1") != 1 {
		t.Fatalf("version counters wrong")
	}
	history := versioner.History("t1")
	if len(history) != 1 || history[0].Number != 1 {
		t.Fatalf("history wrong: %+v", history)
	}
	if found := versioner.Find("t1", 1); found == nil || found.Schema.Fields[0].Name != "id" {
		t.Fatalf("find failed: %+v", found)
	}
	if numbers := versioner.PublishedNumbers("t1"); len(numbers) != 1 || numbers[0] != 1 {
		t.Fatalf("published numbers wrong: %v", numbers)
	}
}

func TestPublishSequenceAndSupersede(t *testing.T) {
	versioner, _ := newFixture(t)
	_, _ = versioner.Draft("t1", model.Schema{Fields: []model.Field{{Name: "id", Type: "BIGINT"}}})
	_, _ = versioner.Publish("t1")
	second, err := versioner.Draft("t1", model.Schema{Fields: []model.Field{{Name: "id", Type: "BIGINT"}, {Name: "amount", Type: "DOUBLE"}}})
	if err != nil || second.Number != 2 {
		t.Fatalf("second draft wrong: %v %+v", err, second)
	}
	_, _ = versioner.Publish("t1")
	versioner.Supersede("t1")
	if state := versioner.Find("t1", 1).State; state != model.VersionSuperseded {
		t.Fatalf("supersede failed: %s", state)
	}
	current, err := versioner.Current("t1")
	if err != nil || current.Number != 2 {
		t.Fatalf("current version wrong: %v %+v", err, current)
	}
}

func TestRollbackActivatesPrevious(t *testing.T) {
	versioner, _ := newFixture(t)
	_, _ = versioner.Draft("t1", model.Schema{Fields: []model.Field{{Name: "id", Type: "BIGINT"}}})
	_, _ = versioner.Publish("t1")
	_, _ = versioner.Draft("t1", model.Schema{Fields: []model.Field{{Name: "id", Type: "BIGINT"}, {Name: "v2", Type: "TEXT"}}})
	_, _ = versioner.Publish("t1")
	if err := versioner.Rollback("t1"); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}
	current, err := versioner.Current("t1")
	if err != nil || current.Number != 1 {
		t.Fatalf("rollback did not activate previous: %v %+v", err, current)
	}
	if state := versioner.Find("t1", 2).State; state != model.VersionRolledBack {
		t.Fatalf("rollback did not mark version")
	}
}

func TestSwitchToAndSortedTables(t *testing.T) {
	versioner, _ := newFixture(t)
	_, _ = versioner.Draft("t1", model.Schema{Fields: []model.Field{{Name: "id", Type: "BIGINT"}}})
	_, _ = versioner.Publish("t1")
	if err := versioner.SwitchTo("t1", 1); err != nil {
		t.Fatalf("switch failed: %v", err)
	}
	if ids := versioner.SortedTables(); len(ids) != 1 || ids[0] != "t1" {
		t.Fatalf("sorted tables wrong: %v", ids)
	}
	field, ok := ParseField((model.Schema{Fields: []model.Field{{Name: "id", Type: "BIGINT"}}}), "id")
	if !ok || field.Name != "id" {
		t.Fatalf("parse field failed")
	}
	if FieldCount(model.Schema{Fields: []model.Field{{Name: "a"}}}) != 1 {
		t.Fatalf("field count failed")
	}
}

func TestMetaNotifierExposed(t *testing.T) {
	versioner, _ := newFixture(t)
	if versioner.Notifier() == nil {
		t.Fatalf("notifier not exposed")
	}
}
