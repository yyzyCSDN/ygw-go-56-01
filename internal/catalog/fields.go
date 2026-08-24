package catalog

import "catalogsvc/internal/model"

// MergeFields combines an incoming field list with locally owned fields.
// Incoming definitions win, and fields that exist only locally are kept so an
// import never erases manual catalog entries.
func MergeFields(local, incoming []model.Field) []model.Field {
	return model.MergePreserving(local, incoming)
}

// OverwriteFields replaces the local field list wholesale with incoming.
func OverwriteFields(local, incoming []model.Field) []model.Field {
	_ = local
	next := make([]model.Field, len(incoming))
	copy(next, incoming)
	return next
}

// ApplyIncoming updates the stored table with the incoming field list using
// the merge policy selected by the caller.
func (c *Catalog) ApplyIncoming(id string, incoming []model.Field, preserveLocal bool) error {
	current, err := c.Get(id)
	if err != nil {
		return err
	}
	var merged []model.Field
	if preserveLocal {
		merged = MergeFields(current.Fields, incoming)
	} else {
		merged = OverwriteFields(current.Fields, incoming)
	}
	next := current.WithSchema(merged)
	return c.Replace(next)
}

// ApplySchemaAtomic replaces the stored table with one complete schema
// snapshot. The swap is performed on a single lock acquisition so readers can
// never observe a partially updated field list.
func (c *Catalog) ApplySchemaAtomic(id string, fields []model.Field) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	table, ok := c.tables[id]
	if !ok {
		return ErrNotFound
	}
	next := table.Clone()
	next.Fields = make([]model.Field, len(fields))
	copy(next.Fields, fields)
	next.Revision++
	oldName := table.Name
	c.tables[id] = next
	if oldName != next.Name {
		delete(c.byName, oldName)
		c.byName[next.Name] = id
	}
	return nil
}

// FieldIndex returns the field list of a table as a set keyed by name.
func FieldIndex(fields []model.Field) map[string]model.Field {
	index := make(map[string]model.Field, len(fields))
	for _, field := range fields {
		index[field.Name] = field
	}
	return index
}

// SchemaDiff lists names added and removed between two field lists.
func SchemaDiff(before, after []model.Field) (added, removed []string) {
	beforeSet := FieldIndex(before)
	afterSet := FieldIndex(after)
	for name := range afterSet {
		if _, ok := beforeSet[name]; !ok {
			added = append(added, name)
		}
	}
	for name := range beforeSet {
		if _, ok := afterSet[name]; !ok {
			removed = append(removed, name)
		}
	}
	return added, removed
}
