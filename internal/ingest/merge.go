package ingest

import "catalogsvc/internal/model"

// applyTable upserts one table from an import segment. Existing tables are
// merged with the incoming field list so only the columns present upstream are
// overwritten; manually cataloged columns are preserved across upstream schema
// changes.
func (im *Importer) applyTable(table model.Table) error {
	if im.catalog.HasTable(table.ID) {
		return im.catalog.ApplyIncoming(table.ID, table.Fields, true)
	}
	return im.catalog.Register(&table)
}

// SegmentSize returns the number of tables in a segment.
func SegmentSize(segment Segment) int {
	return len(segment.Tables)
}

// SegmentIDs returns the ids of all committed segments.
func SegmentIDs(segments []Segment) []string {
	ids := make([]string, 0, len(segments))
	for _, segment := range segments {
		ids = append(ids, segment.ID)
	}
	return ids
}

// ImportedCount counts how many tables an import has written.
func ImportedCount(written map[string]bool) int {
	return len(written)
}
