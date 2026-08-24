package model

// Table is one registered catalog entry describing a data table.
type Table struct {
	ID       string
	Name     string
	Schema   string
	Owner    string
	Fields   []Field
	Revision int
}

// Clone returns a deep copy of the table so callers can safely mutate it.
func (t *Table) Clone() *Table {
	if t == nil {
		return nil
	}
	next := &Table{
		ID:       t.ID,
		Name:     t.Name,
		Schema:   t.Schema,
		Owner:    t.Owner,
		Revision: t.Revision,
	}
	if t.Fields != nil {
		next.Fields = make([]Field, len(t.Fields))
		copy(next.Fields, t.Fields)
	}
	return next
}

// Field returns the field with the given name, if it exists.
func (t *Table) Field(name string) (Field, bool) {
	if t == nil {
		return Field{}, false
	}
	for _, field := range t.Fields {
		if field.Name == name {
			return field, true
		}
	}
	return Field{}, false
}

// FieldNames returns the ordered list of field names on the table.
func (t *Table) FieldNames() []string {
	if t == nil {
		return nil
	}
	names := make([]string, 0, len(t.Fields))
	for _, field := range t.Fields {
		names = append(names, field.Name)
	}
	return names
}

// WithSchema returns a copy of the table with its schema replaced by the
// supplied field list. The receiver is never mutated.
func (t *Table) WithSchema(fields []Field) *Table {
	next := t.Clone()
	next.Fields = make([]Field, len(fields))
	copy(next.Fields, fields)
	next.Revision++
	return next
}

// WithMeta returns a copy of the table with descriptive metadata replaced.
func (t *Table) WithMeta(name, schema, owner string) *Table {
	next := t.Clone()
	next.Name = name
	next.Schema = schema
	next.Owner = owner
	next.Revision++
	return next
}

// EqualFields reports whether two tables expose exactly the same field list.
func EqualFields(left, right []Field) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
