package meta

import "catalogsvc/internal/model"

// ParseFields returns a defensive copy of a schema's field list.
func ParseFields(schema model.Schema) []model.Field {
	fields := make([]model.Field, len(schema.Fields))
	copy(fields, schema.Fields)
	return fields
}

// ParseField returns a named field from a schema.
func ParseField(schema model.Schema, name string) (model.Field, bool) {
	for _, field := range schema.Fields {
		if field.Name == name {
			return field, true
		}
	}
	return model.Field{}, false
}

// SchemaVersion pairs a schema with the version number it belongs to.
type SchemaVersion struct {
	Number int
	Schema model.Schema
}

// ResolveVersion extracts the version-numbered schema for a table.
func (m *Meta) ResolveVersion(tableID string) (SchemaVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	version := m.resolveCurrent(tableID)
	if version == nil {
		return SchemaVersion{}, ErrVersionUnavailable
	}
	return SchemaVersion{
		Number: version.Number,
		Schema: version.Schema.Clone(),
	}, nil
}

// FieldCount returns the number of fields in a schema.
func FieldCount(schema model.Schema) int {
	return len(schema.Fields)
}

func fieldNames(fields []model.Field) []string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field.Name)
	}
	return names
}
