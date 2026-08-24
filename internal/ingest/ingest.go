package ingest

import (
	"errors"

	"catalogsvc/internal/catalog"
	"catalogsvc/internal/meta"
	"catalogsvc/internal/model"
	"catalogsvc/internal/notify"
)

// ErrImportFailed is returned when an import is attempted after a failure.
var ErrImportFailed = errors.New("ingest: import failed, rollback required")

// Segment is one batch of tables applied during an import.
type Segment struct {
	ID     string
	Tables []model.Table
	Commit bool
}

// Importer applies upstream metadata in segments and rolls back atomically.
type Importer struct {
	catalog  *catalog.Catalog
	meta     *meta.Meta
	notifier *notify.Notifier
	baseline map[string]*model.Table
	segments []Segment
	inflight *Segment
	written  map[string]bool
	failed   bool
	lastErr  error
}

// New creates an importer bound to the catalog, version manager and notifier.
func New(c *catalog.Catalog, m *meta.Meta, n *notify.Notifier) *Importer {
	return &Importer{
		catalog:  c,
		meta:     m,
		notifier: n,
		written:  make(map[string]bool),
	}
}

// Begin captures the pre-import baseline for rollback.
func (im *Importer) Begin() error {
	if len(im.segments) > 0 || im.inflight != nil {
		return errors.New("ingest: import already in progress")
	}
	im.baseline = im.catalog.Snapshot()
	im.written = make(map[string]bool)
	im.failed = false
	im.lastErr = nil
	return nil
}

// Apply merges one segment of tables into the catalog.
func (im *Importer) Apply(segment Segment) error {
	if im.failed {
		return ErrImportFailed
	}
	if segment.ID == "" {
		segment.ID = NextSegmentID()
	}
	for _, table := range segment.Tables {
		if err := im.applyTable(table); err != nil {
			im.failed = true
			im.lastErr = err
			return err
		}
		im.written[table.ID] = true
	}
	if segment.Commit {
		im.segments = append(im.segments, segment)
	} else {
		im.inflight = &segment
	}
	return nil
}

// Commit finalizes the current in-flight segment as committed.
func (im *Importer) Commit() error {
	if im.failed {
		return ErrImportFailed
	}
	if im.inflight == nil {
		return errors.New("ingest: no in-flight segment")
	}
	segment := *im.inflight
	segment.Commit = true
	im.segments = append(im.segments, segment)
	im.inflight = nil
	return nil
}

// Fail marks the import as failed so rollback must be invoked.
func (im *Importer) Fail(err error) {
	if err == nil {
		err = errors.New("ingest: upstream failure")
	}
	im.failed = true
	im.lastErr = err
}

// Rollback restores every table written since Begin, including the in-flight
// segment, so the catalog returns exactly to the pre-import baseline.
func (im *Importer) Rollback() error {
	if im.baseline == nil {
		return errors.New("ingest: no baseline captured")
	}
	tables := allWrittenTables(im.segments, im.inflight)
	restoreFromBaseline(im.catalog, im.baseline, tables)
	im.catalog.DropNotIn(im.baseline)
	im.reset()
	return nil
}

// LastError returns the error that failed the import, if any.
func (im *Importer) LastError() error {
	return im.lastErr
}

// Failed reports whether the import is in a failed state.
func (im *Importer) Failed() bool {
	return im.failed
}

// HasInflight reports whether an uncommitted segment is pending.
func (im *Importer) HasInflight() bool {
	return im.inflight != nil
}

// WrittenTables lists every table touched since Begin.
func (im *Importer) WrittenTables() []string {
	ids := make([]string, 0, len(im.written))
	for id := range im.written {
		ids = append(ids, id)
	}
	return ids
}

// RollbackScope reports how many baseline tables survive and how many
// import-created tables would remain after a rollback.
func (im *Importer) RollbackScope() (preserved, residue int) {
	if im.baseline == nil {
		return 0, 0
	}
	tables := allWrittenTables(im.segments, im.inflight)
	return PreservedCount(im.baseline, tables), ResidueCount(im.catalog.Snapshot(), im.baseline)
}

func (im *Importer) reset() {
	im.baseline = nil
	im.segments = nil
	im.inflight = nil
	im.written = make(map[string]bool)
	im.failed = false
	im.lastErr = nil
}
