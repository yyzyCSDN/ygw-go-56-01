package ingest

import (
	"catalogsvc/internal/catalog"
	"catalogsvc/internal/model"
)

// restoreFromBaseline resets the named tables to their pre-import snapshots
// while leaving other tables untouched.
func restoreFromBaseline(c *catalog.Catalog, baseline map[string]*model.Table, tables map[string]bool) {
	restore := make(map[string]*model.Table)
	for id, table := range baseline {
		if tables[id] {
			restore[id] = table
		}
	}
	c.RestoreTables(restore)
}

// PreservedCount returns how many baseline tables would survive a rollback.
func PreservedCount(baseline map[string]*model.Table, tables map[string]bool) int {
	count := 0
	for id := range baseline {
		if !tables[id] {
			count++
		}
	}
	return count
}

// ResidueCount counts current tables missing from the baseline.
func ResidueCount(current, baseline map[string]*model.Table) int {
	count := 0
	for id := range current {
		if _, ok := baseline[id]; !ok {
			count++
		}
	}
	return count
}
