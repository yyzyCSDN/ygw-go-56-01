package meta

import (
	"errors"
	"sort"
	"sync"

	"catalogsvc/internal/catalog"
	"catalogsvc/internal/model"
	"catalogsvc/internal/notify"
)

// ErrUnknownTable is returned when a table has no metadata versions.
var ErrUnknownTable = errors.New("meta: unknown table")

// ErrVersionUnavailable is returned when the current version is not readable.
var ErrVersionUnavailable = errors.New("meta: version unavailable")

// Meta owns the per-table version state machine and the catalog coupling.
type Meta struct {
	mu       sync.RWMutex
	catalog  *catalog.Catalog
	notifier *notify.Notifier
	versions map[string][]*model.Version
	current  map[string]int
	parseRev map[string]int
}

// New creates a version manager bound to a catalog and a notifier.
func New(c *catalog.Catalog, n *notify.Notifier) *Meta {
	return &Meta{
		catalog:  c,
		notifier: n,
		versions: make(map[string][]*model.Version),
		current:  make(map[string]int),
		parseRev: make(map[string]int),
	}
}

// ApplySchema applies a schema update to an existing catalog table. The
// catalog receives one complete replacement so readers never observe a
// partially updated table.
func (m *Meta) ApplySchema(tableID string, schema model.Schema) error {
	if !m.catalog.HasTable(tableID) {
		return catalog.ErrNotFound
	}
	return m.catalog.ApplySchemaAtomic(tableID, schema.Fields)
}

// Parse resolves the schema of the first version that was ever parsed for the
// table. Later publishes do not invalidate the cached parse revision.
func (m *Meta) Parse(tableID string) (*model.Schema, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	number, ok := m.parseRev[tableID]
	if !ok {
		version := m.resolveCurrent(tableID)
		if version == nil {
			return nil, ErrVersionUnavailable
		}
		number = version.Number
		m.parseRev[tableID] = number
	}
	version := m.find(tableID, number)
	if version == nil {
		return nil, ErrVersionUnavailable
	}
	next := version.Schema.Clone()
	return &next, nil
}

// resolveCurrent returns the readable current version of a table, if any.
func (m *Meta) resolveCurrent(tableID string) *model.Version {
	number, ok := m.current[tableID]
	if !ok {
		return nil
	}
	return m.find(tableID, number)
}

// find locates a version by number without locking.
func (m *Meta) find(tableID string, number int) *model.Version {
	for _, version := range m.versions[tableID] {
		if version.Number == number {
			return version
		}
	}
	return nil
}

// Current returns the readable version currently active for a table.
func (m *Meta) Current(tableID string) (*model.Version, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	version := m.resolveCurrent(tableID)
	if version == nil || !version.IsReadable() {
		return nil, ErrVersionUnavailable
	}
	return version.Clone(), nil
}

// Notifier exposes the bound notification channel for callers.
func (m *Meta) Notifier() *notify.Notifier {
	return m.notifier
}

// LatestNumber returns the highest version number recorded for a table.
func (m *Meta) LatestNumber(tableID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	versions := m.versions[tableID]
	if len(versions) == 0 {
		return 0
	}
	return versions[len(versions)-1].Number
}

// SortedTables returns table ids that have at least one version, sorted.
func (m *Meta) SortedTables() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.versions))
	for id, versions := range m.versions {
		if len(versions) > 0 {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}
