package model

import "time"

// EventType classifies catalog change notifications.
type EventType string

const (
	EventTableCreated   EventType = "table.created"
	EventSchemaChanged  EventType = "schema.changed"
	EventVersionPublish EventType = "version.published"
	EventVersionRolled  EventType = "version.rolled-back"
	EventLineageChanged EventType = "lineage.changed"
)

// ChangeEvent is delivered to subscribers after a catalog mutation.
type ChangeEvent struct {
	TableID  string
	Version  int
	Type     EventType
	Fields   []string
	Occurred time.Time
}

// Key returns a stable identity used for retry bookkeeping.
func (e ChangeEvent) Key() string {
	return string(e.Type) + ":" + e.TableID + ":" + itoa(e.Version)
}

// Snapshot is an immutable view of one table delivered to subscribers.
type Snapshot struct {
	Table   *Table
	Version int
	At      time.Time
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for value > 0 {
		pos--
		buf[pos] = byte('0' + value%10)
		value /= 10
	}
	return string(buf[pos:])
}
