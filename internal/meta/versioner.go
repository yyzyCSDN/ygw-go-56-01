package meta

import (
	"context"
	"errors"

	"catalogsvc/internal/model"
)

// ErrNoDraft is returned when publish finds no draft version to release.
var ErrNoDraft = errors.New("meta: no draft version")

// Draft creates the next version of a table in the draft state.
func (m *Meta) Draft(tableID string, schema model.Schema) (*model.Version, error) {
	if !m.catalog.HasTable(tableID) {
		return nil, catalogErrNotFound()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	versions := m.versions[tableID]
	number := 1
	if len(versions) > 0 {
		number = versions[len(versions)-1].Number + 1
	}
	version := &model.Version{
		TableID: tableID,
		Number:  number,
		State:   model.VersionDraft,
		Schema:  schema.Clone(),
	}
	m.versions[tableID] = append(versions, version)
	return version.Clone(), nil
}

// Publish transitions the newest draft into the published state and notifies
// subscribers. Notification failures are propagated to the caller so that a
// permanently unreachable subscriber is reported rather than silently dropped.
func (m *Meta) Publish(tableID string) (*model.Version, error) {
	m.mu.Lock()
	version := m.latestDraft(tableID)
	if version == nil {
		m.mu.Unlock()
		return nil, ErrNoDraft
	}
	version.State = model.VersionPublished
	m.current[tableID] = version.Number
	event := model.ChangeEvent{
		TableID: tableID,
		Version: version.Number,
		Type:    model.EventVersionPublish,
		Fields:  fieldNames(version.Schema.Fields),
	}
	m.mu.Unlock()
	return version.Clone(), m.notifier.Push(context.Background(), event)
}

// Rollback moves the current published version to rolled-back and activates
// the previous readable version, if one exists.
func (m *Meta) Rollback(tableID string) error {
	m.mu.Lock()
	version := m.resolveCurrent(tableID)
	if version == nil {
		m.mu.Unlock()
		return ErrVersionUnavailable
	}
	version.State = model.VersionRolledBack
	number := version.Number
	previous := m.previousReadable(tableID, version.Number)
	if previous == nil {
		delete(m.current, tableID)
	} else {
		m.current[tableID] = previous.Number
	}
	m.mu.Unlock()
	event := model.ChangeEvent{
		TableID: tableID,
		Version: number,
		Type:    model.EventVersionRolled,
	}
	return m.notifier.Push(context.Background(), event)
}

// SwitchTo activates a specific published version for a table.
func (m *Meta) SwitchTo(tableID string, number int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	version := m.find(tableID, number)
	if version == nil || !version.IsReadable() {
		return ErrVersionUnavailable
	}
	m.current[tableID] = number
	return nil
}

// latestDraft returns the most recent draft without locking.
func (m *Meta) latestDraft(tableID string) *model.Version {
	versions := m.versions[tableID]
	for i := len(versions) - 1; i >= 0; i-- {
		if versions[i].State == model.VersionDraft {
			return versions[i]
		}
	}
	return nil
}

// previousReadable finds the highest readable version below a number.
func (m *Meta) previousReadable(tableID string, below int) *model.Version {
	var previous *model.Version
	for _, version := range m.versions[tableID] {
		if version.Number >= below || !version.IsReadable() {
			continue
		}
		if previous == nil || version.Number > previous.Number {
			previous = version
		}
	}
	return previous
}

func catalogErrNotFound() error {
	return errors.New("meta: table not registered")
}
