package query

import (
	"testing"

	"catalogsvc/internal/catalog"
	"catalogsvc/internal/meta"
	"catalogsvc/internal/model"
	"catalogsvc/internal/notify"
)

func newService(t *testing.T) (*Service, *meta.Meta, *catalog.Catalog) {
	t.Helper()
	cat := catalog.New()
	cat.Register(&model.Table{
		ID:     "t1",
		Name:   "orders",
		Fields: []model.Field{{Name: "id", Type: "BIGINT", Primary: true}, {Name: "amount", Type: "DOUBLE"}, {Name: "note", Type: "TEXT"}},
	})
	versioner := meta.New(cat, notify.New())
	table, err := cat.Get("t1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	_, err = versioner.Draft("t1", model.Schema{Fields: table.Fields})
	if err != nil {
		t.Fatalf("draft failed: %v", err)
	}
	if _, err := versioner.Publish("t1"); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	return New(versioner, cat, catalog.NewCache()), versioner, cat
}

func TestGetFieldAndFilters(t *testing.T) {
	svc, _, _ := newService(t)
	field, err := svc.GetField("t1", "amount")
	if err != nil || field.Type != "DOUBLE" {
		t.Fatalf("get field failed: %v %+v", err, field)
	}
	if _, err := svc.GetField("t1", "missing"); err == nil {
		t.Fatalf("missing field must error")
	}
	filtered, err := svc.GetFieldsFiltered("t1", []string{"note", "id"})
	if err != nil || len(filtered) != 2 {
		t.Fatalf("filter failed: %v %+v", err, filtered)
	}
	keys, err := svc.PrimaryFields("t1")
	if err != nil || len(keys) != 1 || keys[0].Name != "id" {
		t.Fatalf("primary keys wrong: %v %+v", err, keys)
	}
	sorted, err := svc.SortedFields("t1")
	if err != nil || sorted[0].Name != "amount" {
		t.Fatalf("sorted fields wrong: %v", sorted)
	}
}

func TestVersionAndList(t *testing.T) {
	svc, _, _ := newService(t)
	version, err := svc.Version("t1")
	if err != nil || version != 1 {
		t.Fatalf("version wrong: %v %d", err, version)
	}
	if tables := svc.ListTables(); len(tables) != 1 || tables[0].ID != "t1" {
		t.Fatalf("list wrong: %+v", tables)
	}
	svc.RefreshCache("t1")
	if !IsNotFound(errTableNotFound) || IsNotFound(nil) {
		t.Fatalf("IsNotFound helper wrong")
	}
	if !IsVersionUnavailable(errVersionUnavailable) || IsVersionUnavailable(nil) {
		t.Fatalf("IsVersionUnavailable helper wrong")
	}
}

func TestGetTableThroughCache(t *testing.T) {
	svc, _, cat := newService(t)
	table, err := cat.Get("t1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	svc.cache.Put(table)
	got, err := svc.GetTable("t1")
	if err != nil || got.ID != "t1" {
		t.Fatalf("get table failed: %v %+v", err, got)
	}
}

func TestLookupHelpers(t *testing.T) {
	fields := []model.Field{{Name: "b", Type: "TEXT"}, {Name: "a", Type: "BIGINT", Primary: true}}
	if _, ok := FindField(fields, "a"); !ok {
		t.Fatalf("find failed")
	}
	filtered := FilterFields(fields, []string{"a"})
	if len(filtered) != 1 || filtered[0].Name != "a" {
		t.Fatalf("filter failed: %+v", filtered)
	}
	ordered := OrderFields(fields)
	if ordered[0].Name != "a" {
		t.Fatalf("order failed: %+v", ordered)
	}
	keys := KeyByID(fields, true)
	if len(keys) != 1 {
		t.Fatalf("key by id failed: %+v", keys)
	}
	loose := KeyByID(fields, false)
	if len(loose) != 1 {
		t.Fatalf("key by id loose must still honor primary flag: %+v", loose)
	}
}
