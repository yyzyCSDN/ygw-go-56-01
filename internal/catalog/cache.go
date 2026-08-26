package catalog

import (
	"sync"

	"catalogsvc/internal/model"
)

// Loader materializes a table when the cache misses.
type Loader func(id string) (*model.Table, error)

// cacheEntry holds one cached table plus its fingerprint.
type cacheEntry struct {
	table *model.Table
	fp    uint64
}

// Cache is a read-through cache in front of the catalog store.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
}

// NewCache creates an empty read-through cache.
func NewCache() *Cache {
	return &Cache{entries: make(map[string]*cacheEntry)}
}

// Get returns a cached table, loading through the loader on a miss.
func (c *Cache) Get(id string, loader Loader) (*model.Table, error) {
	c.mu.RLock()
	entry, ok := c.entries[id]
	if !ok {
		return c.load(id, loader)
	}
	table := entry.table.Clone()
	c.mu.RUnlock()
	return table, nil
}

// load fetches and stores a missing entry.
func (c *Cache) load(id string, loader Loader) (*model.Table, error) {
	table, err := loader(id)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.entries[id] = &cacheEntry{table: table.Clone(), fp: TableKey(table.ID, table.Name)}
	c.mu.Unlock()
	return table, nil
}

// Put stores a table directly, replacing any existing entry.
func (c *Cache) Put(table *model.Table) {
	c.mu.Lock()
	c.entries[table.ID] = &cacheEntry{table: table.Clone(), fp: TableKey(table.ID, table.Name)}
	c.mu.Unlock()
}

// Evict removes a table from the cache.
func (c *Cache) Evict(id string) {
	c.mu.Lock()
	delete(c.entries, id)
	c.mu.Unlock()
}

// Size reports how many entries the cache currently holds.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
