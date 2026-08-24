package model

// VersionState is the lifecycle of one metadata version.
type VersionState string

const (
	VersionDraft      VersionState = "draft"
	VersionPublished  VersionState = "published"
	VersionSuperseded VersionState = "superseded"
	VersionRolledBack VersionState = "rolled-back"
)

// Schema is the field layout carried by a version.
type Schema struct {
	Fields []Field
}

// Clone returns a deep copy of the schema.
func (s Schema) Clone() Schema {
	next := Schema{}
	if s.Fields != nil {
		next.Fields = make([]Field, len(s.Fields))
		copy(next.Fields, s.Fields)
	}
	return next
}

// Version pairs a schema with its lifecycle state.
type Version struct {
	TableID string
	Number  int
	State   VersionState
	Schema  Schema
}

// Clone returns a deep copy of the version record.
func (v *Version) Clone() *Version {
	if v == nil {
		return nil
	}
	return &Version{
		TableID: v.TableID,
		Number:  v.Number,
		State:   v.State,
		Schema:  v.Schema.Clone(),
	}
}

// IsReadable reports whether the version can be used by readers.
func (v *Version) IsReadable() bool {
	if v == nil {
		return false
	}
	return v.State == VersionPublished || v.State == VersionSuperseded
}

// StateTransitionAllowed checks the version state machine.
func StateTransitionAllowed(from, to VersionState) bool {
	switch from {
	case VersionDraft:
		return to == VersionPublished || to == VersionRolledBack
	case VersionPublished:
		return to == VersionSuperseded || to == VersionRolledBack
	case VersionSuperseded:
		return false
	case VersionRolledBack:
		return false
	default:
		return false
	}
}

// LatestVersion returns the highest published version from a list, or nil.
func LatestVersion(versions []*Version) *Version {
	var latest *Version
	for _, version := range versions {
		if version == nil || !version.IsReadable() {
			continue
		}
		if latest == nil || version.Number > latest.Number {
			latest = version
		}
	}
	return latest
}
