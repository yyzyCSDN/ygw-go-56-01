package catalog

import (
	"fmt"
	"testing"
	"time"

	"catalogsvc/internal/model"
)

func TestCatalogCacheLockReleased(t *testing.T) {
	cache := NewCache()
	loader := func(id string) (*model.Table, error) {
		return &model.Table{ID: id, Name: id, Revision: 1}, nil
	}

	first := make(chan error, 1)
	go func() {
		_, err := cache.Get("missing-1", loader)
		first <- err
	}()
	select {
	case err := <-first:
		if err != nil {
			t.Fatalf("first query failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cache read lock was not released after a miss")
	}

	second := make(chan error, 1)
	go func() {
		table, err := cache.Get("missing-2", loader)
		if err != nil {
			second <- err
			return
		}
		if table == nil || table.ID != "missing-2" {
			second <- fmt.Errorf("unexpected result: %+v", table)
			return
		}
		second <- nil
	}()
	select {
	case err := <-second:
		if err != nil {
			t.Fatalf("subsequent query failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("subsequent catalog query blocked by a leaked lock")
	}
}
