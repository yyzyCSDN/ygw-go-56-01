package ingest

import "fmt"

var segmentCounter int

// NextSegmentID allocates a monotonic segment identifier.
func NextSegmentID() string {
	segmentCounter++
	return fmt.Sprintf("seg-%04d", segmentCounter)
}

// committedTables returns the set of table ids written by committed segments.
func committedTables(segments []Segment) map[string]bool {
	tables := make(map[string]bool)
	for _, segment := range segments {
		for _, table := range segment.Tables {
			tables[table.ID] = true
		}
	}
	return tables
}

// inflightTables returns the set of table ids written by the in-flight segment.
func inflightTables(segment *Segment) map[string]bool {
	tables := make(map[string]bool)
	if segment == nil {
		return tables
	}
	for _, table := range segment.Tables {
		tables[table.ID] = true
	}
	return tables
}

// allWrittenTables unions committed and in-flight table ids.
func allWrittenTables(segments []Segment, inflight *Segment) map[string]bool {
	tables := committedTables(segments)
	for id := range inflightTables(inflight) {
		tables[id] = true
	}
	return tables
}
