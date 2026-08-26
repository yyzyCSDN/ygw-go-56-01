package catalog

import (
	"errors"
	"sort"
	"sync"

	"catalogsvc/internal/model"
)

// ErrDuplicateName is returned when a table name is already registered.
var ErrDuplicateName = errors.New("catalog: duplicate table name")

// ErrNotFound is returned when a table id is unknown to the catalog.
var ErrNotFound = errors.New("catalog: table not found")

// Catalog stores registered tables and the field index.
type Catalog struct {
	mu     sync.RWMutex
	tables map[string]*model.Table
	byName map[string]string
}

// New creates an empty catalog.
func New() *Catalog {
	return &Catalog{
		tables: make(map[string]*model.Table),
		byName: make(map[string]string),
	}
}

// Register adds a table after checking for name collisions.
func (c *Catalog) Register(table *model.Table) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if table == nil || table.ID == "" {
		return errors.New("catalog: table id is required")
	}
	if existing, ok := c.byName[table.Name]; ok && existing != table.ID {
		return ErrDuplicateName
	}
	c.tables[table.ID] = table.Clone()
	c.byName[table.Name] = table.ID
	return nil
}

// Get returns a copy of the stored table.
func (c *Catalog) Get(id string) (*model.Table, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	table, ok := c.tables[id]
	if !ok {
		return nil, ErrNotFound
	}
	return table.Clone(), nil
}

// List returns every registered table ordered by id.
func (c *Catalog) List() []*model.Table {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ids := make([]string, 0, len(c.tables))
	for id := range c.tables {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	tables := make([]*model.Table, 0, len(ids))
	for _, id := range ids {
		tables = append(tables, c.tables[id].Clone())
	}
	return tables
}

// Replace swaps the stored table for a complete new snapshot atomically.
// Readers observe either the old table or the new table, never a mix.
func (c *Catalog) Replace(table *model.Table) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.tables[table.ID]; !ok {
		return ErrNotFound
	}
	if existing, ok := c.byName[table.Name]; ok && existing != table.ID {
		return ErrDuplicateName
	}
	next := table.Clone()
	old := c.tables[table.ID]
	delete(c.byName, old.Name)
	c.tables[table.ID] = next
	c.byName[table.Name] = table.ID
	return nil
}

// Snapshot returns a deep copy of every table for rollback purposes.
func (c *Catalog) Snapshot() map[string]*model.Table {
	c.mu.RLock()
	defer c.mu.RUnlock()
	snapshot := make(map[string]*model.Table, len(c.tables))
	for id, table := range c.tables {
		snapshot[id] = table.Clone()
	}
	return snapshot
}

// RestoreTables applies a captured snapshot onto the catalog. Tables named in
// the snapshot are replaced by their snapshot values; tables absent from the
// snapshot are left untouched.
func (c *Catalog) RestoreTables(snapshot map[string]*model.Table) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, table := range snapshot {
		next := table.Clone()
		c.tables[id] = next
	}
	c.rebuildNames()
}

// DropNotIn deletes every table whose id is absent from the supplied baseline,
// so import-created tables are removed during rollback.
func (c *Catalog) DropNotIn(baseline map[string]*model.Table) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id := range c.tables {
		if _, ok := baseline[id]; !ok {
			delete(c.tables, id)
		}
	}
	c.rebuildNames()
}

// rebuildNames reconstructs the name index from the current table map.
func (c *Catalog) rebuildNames() {
	c.byName = make(map[string]string, len(c.tables))
	for id, table := range c.tables {
		c.byName[table.Name] = id
	}
}

// HasTable reports whether a table id is registered.
func (c *Catalog) HasTable(id string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.tables[id]
	return ok
}
