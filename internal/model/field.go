package model

// Field describes one column of a catalog table.
type Field struct {
	Name     string
	Type     string
	Nullable bool
	Primary  bool
}

// NewField builds a field with a normalized type name.
func NewField(name, typ string, nullable, primary bool) Field {
	return Field{
		Name:     name,
		Type:     NormalizeType(typ),
		Nullable: nullable,
		Primary:  primary,
	}
}

// NormalizeType canonicalizes common SQL type spellings.
func NormalizeType(typ string) string {
	switch typ {
	case "INTEGER", "INT", "BIGINT":
		return "BIGINT"
	case "VARCHAR", "TEXT", "STRING":
		return "TEXT"
	case "DOUBLE", "FLOAT", "REAL":
		return "DOUBLE"
	case "TIMESTAMP", "DATETIME":
		return "TIMESTAMP"
	default:
		return typ
	}
}

// FieldSet indexes fields by name for fast membership checks.
type FieldSet map[string]Field

// BuildFieldSet converts a field list into a name-indexed set.
func BuildFieldSet(fields []Field) FieldSet {
	set := make(FieldSet, len(fields))
	for _, field := range fields {
		set[field.Name] = field
	}
	return set
}

// Added returns names present in next but absent from base.
func (set FieldSet) Added(base FieldSet) []string {
	var names []string
	for name := range set {
		if _, ok := base[name]; !ok {
			names = append(names, name)
		}
	}
	return names
}

// Removed returns names present in base but absent from next.
func (set FieldSet) Removed(base FieldSet) []string {
	var names []string
	for name := range base {
		if _, ok := set[name]; !ok {
			names = append(names, name)
		}
	}
	return names
}

// MergePreserving returns the incoming fields followed by any fields that only
// exist in the local field list. Incoming definitions win on name collisions.
func MergePreserving(local, incoming []Field) []Field {
	localSet := BuildFieldSet(local)
	incomingSet := BuildFieldSet(incoming)
	merged := make([]Field, 0, len(local)+len(incoming))
	merged = append(merged, incoming...)
	for name, field := range localSet {
		if _, ok := incomingSet[name]; !ok {
			merged = append(merged, field)
		}
	}
	return merged
}
