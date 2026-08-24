package meta

import (
	"sort"

	"catalogsvc/internal/model"
)

// History returns every version of a table ordered by number.
func (m *Meta) History(tableID string) []*model.Version {
	m.mu.RLock()
	defer m.mu.RUnlock()
	versions := m.versions[tableID]
	next := make([]*model.Version, 0, len(versions))
	for _, version := range versions {
		next = append(next, version.Clone())
	}
	sort.Slice(next, func(i, j int) bool {
		return next[i].Number < next[j].Number
	})
	return next
}

// Count returns the number of recorded versions for a table.
func (m *Meta) Count(tableID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.versions[tableID])
}

// Find returns a version by number, or nil when absent.
func (m *Meta) Find(tableID string, number int) *model.Version {
	m.mu.RLock()
	defer m.mu.RUnlock()
	version := m.find(tableID, number)
	if version == nil {
		return nil
	}
	return version.Clone()
}

// Supersede marks every published version older than the current one.
func (m *Meta) Supersede(tableID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	number, ok := m.current[tableID]
	if !ok {
		return
	}
	for _, version := range m.versions[tableID] {
		if version.State == model.VersionPublished && version.Number != number {
			version.State = model.VersionSuperseded
		}
	}
}

// PublishedNumbers lists readable version numbers for a table.
func (m *Meta) PublishedNumbers(tableID string) []int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var numbers []int
	for _, version := range m.versions[tableID] {
		if version.IsReadable() {
			numbers = append(numbers, version.Number)
		}
	}
	sort.Ints(numbers)
	return numbers
}
