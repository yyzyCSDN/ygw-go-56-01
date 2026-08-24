package notify

import (
	"sort"
	"time"

	"catalogsvc/internal/catalog"
	"catalogsvc/internal/model"
)

// SnapshotBuilder produces immutable table snapshots for subscribers.
type SnapshotBuilder struct {
	catalog *catalog.Catalog
}

// NewSnapshotBuilder creates a snapshot builder over a catalog.
func NewSnapshotBuilder(c *catalog.Catalog) *SnapshotBuilder {
	return &SnapshotBuilder{catalog: c}
}

// Build captures one table plus the version number at delivery time.
func (b *SnapshotBuilder) Build(tableID string, version int) (model.Snapshot, error) {
	table, err := b.catalog.Get(tableID)
	if err != nil {
		return model.Snapshot{}, err
	}
	return model.Snapshot{
		Table:   table,
		Version: version,
		At:      time.Now().UTC(),
	}, nil
}

// BuildAll captures every registered table in id order.
func (b *SnapshotBuilder) BuildAll() []model.Snapshot {
	tables := b.catalog.List()
	snapshots := make([]model.Snapshot, 0, len(tables))
	for _, table := range tables {
		snapshots = append(snapshots, model.Snapshot{
			Table:   table,
			Version: table.Revision,
			At:      time.Now().UTC(),
		})
	}
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Table.ID < snapshots[j].Table.ID
	})
	return snapshots
}
