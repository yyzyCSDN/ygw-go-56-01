package catalog

import (
	"errors"
	"testing"

	"catalogsvc/internal/model"
)

func TestRegisterGetListReplace(t *testing.T) {
	cat := New()
	table := &model.Table{ID: "t1", Name: "orders", Schema: "ods", Owner: "team", Fields: []model.Field{{Name: "id", Type: "BIGINT", Primary: true}}}
	if err := cat.Register(table); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if err := cat.Register(&model.Table{ID: "t2", Name: "orders"}); !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("duplicate name not rejected: %v", err)
	}
	got, err := cat.Get("t1")
	if err != nil || got.Name != "orders" {
		t.Fatalf("get failed: %v %+v", err, got)
	}
	got.Fields[0].Name = "mutated"
	again, _ := cat.Get("t1")
	if again.Fields[0].Name != "id" {
		t.Fatalf("get leaks internal storage")
	}
	if len(cat.List()) != 1 || !cat.HasTable("t1") {
		t.Fatalf("list/has failed")
	}
	replaced := &model.Table{ID: "t1", Name: "orders_v2", Fields: []model.Field{{Name: "id", Type: "BIGINT"}}}
	if err := cat.Replace(replaced); err != nil {
		t.Fatalf("replace failed: %v", err)
	}
	got, _ = cat.Get("t1")
	if got.Name != "orders_v2" {
		t.Fatalf("replace did not apply: %+v", got)
	}
}

func TestApplySchemaAtomic(t *testing.T) {
	cat := New()
	cat.Register(&model.Table{ID: "t1", Name: "t1", Fields: []model.Field{{Name: "old", Type: "TEXT"}}})
	if err := cat.ApplySchemaAtomic("t1", []model.Field{{Name: "new", Type: "BIGINT"}}); err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	got, _ := cat.Get("t1")
	if len(got.Fields) != 1 || got.Fields[0].Name != "new" {
		t.Fatalf("atomic apply wrong: %+v", got.Fields)
	}
}

func TestSnapshotRestoreAndDrop(t *testing.T) {
	cat := New()
	cat.Register(&model.Table{ID: "a", Name: "a", Fields: []model.Field{{Name: "x", Type: "TEXT"}}})
	cat.Register(&model.Table{ID: "b", Name: "b"})
	snapshot := cat.Snapshot()
	cat.Register(&model.Table{ID: "c", Name: "c"})
	cat.RestoreTables(map[string]*model.Table{"a": snapshot["a"]})
	if !cat.HasTable("b") || !cat.HasTable("c") {
		t.Fatalf("RestoreTables must keep untouched tables")
	}
	cat.DropNotIn(snapshot)
	if cat.HasTable("c") || !cat.HasTable("a") || !cat.HasTable("b") {
		t.Fatalf("DropNotIn wrong: %v", cat.List())
	}
}

func TestApplyIncomingOverwritesWhenRequested(t *testing.T) {
	cat := New()
	cat.Register(&model.Table{ID: "t1", Name: "t1", Fields: []model.Field{{Name: "manual", Type: "TEXT"}}})
	if err := cat.ApplyIncoming("t1", []model.Field{{Name: "upstream", Type: "BIGINT"}}, false); err != nil {
		t.Fatalf("apply incoming failed: %v", err)
	}
	got, _ := cat.Get("t1")
	if len(got.Fields) != 1 || got.Fields[0].Name != "upstream" {
		t.Fatalf("overwrite policy not applied: %+v", got.Fields)
	}
}

func TestFieldHelpersAndHashes(t *testing.T) {
	fields := []model.Field{{Name: "a", Type: "TEXT"}, {Name: "b", Type: "BIGINT", Primary: true}}
	index := FieldIndex(fields)
	if _, ok := index["b"]; !ok {
		t.Fatalf("field index missing b")
	}
	added, removed := SchemaDiff([]model.Field{{Name: "a"}}, fields)
	if len(added) != 1 || added[0] != "b" || len(removed) != 0 {
		t.Fatalf("schema diff wrong: %v %v", added, removed)
	}
	if TableKey("t1", "n1") == TableKey("t2", "n1") {
		t.Fatalf("table keys collide")
	}
	if EdgeKey("a", "b") == EdgeKey("b", "a") {
		t.Fatalf("edge keys must be directed")
	}
	if FingerprintSchema([]string{"x"}) == FingerprintSchema([]string{"y"}) {
		t.Fatalf("schema fingerprints collide")
	}
}

func TestCacheHitPutEvict(t *testing.T) {
	cache := NewCache()
	table := &model.Table{ID: "t1", Name: "n1", Fields: []model.Field{{Name: "id", Type: "BIGINT"}}}
	cache.Put(table)
	loader := func(id string) (*model.Table, error) { return nil, errors.New("should not load") }
	got, err := cache.Get("t1", loader)
	if err != nil || got.ID != "t1" {
		t.Fatalf("cache hit failed: %v %+v", err, got)
	}
	got.Fields[0].Name = "mutated"
	again, _ := cache.Get("t1", loader)
	if again.Fields[0].Name != "id" {
		t.Fatalf("cache leaks entry storage")
	}
	if cache.Size() != 1 {
		t.Fatalf("cache size wrong")
	}
	cache.Evict("t1")
	if cache.Size() != 0 {
		t.Fatalf("evict failed")
	}
}
