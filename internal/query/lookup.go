package query

import (
	"sort"

	"catalogsvc/internal/model"
)

// FindField locates a field by name in a field list.
func FindField(fields []model.Field, name string) (model.Field, bool) {
	for _, field := range fields {
		if field.Name == name {
			return field, true
		}
	}
	return model.Field{}, false
}

// FilterFields keeps only the named fields, preserving list order.
func FilterFields(fields []model.Field, names []string) []model.Field {
	wanted := make(map[string]bool, len(names))
	for _, name := range names {
		wanted[name] = true
	}
	filtered := make([]model.Field, 0, len(names))
	for _, field := range fields {
		if wanted[field.Name] {
			filtered = append(filtered, field)
		}
	}
	return filtered
}

// OrderFields sorts a field list by name.
func OrderFields(fields []model.Field) []model.Field {
	next := make([]model.Field, len(fields))
	copy(next, fields)
	sort.Slice(next, func(i, j int) bool {
		return next[i].Name < next[j].Name
	})
	return next
}

// KeyByID returns fields flagged as primary keys, or every field when none
// are flagged and strict is false.
func KeyByID(fields []model.Field, strict bool) []model.Field {
	var keys []model.Field
	for _, field := range fields {
		if field.Primary {
			keys = append(keys, field)
		}
	}
	if len(keys) == 0 && !strict {
		return fields
	}
	return keys
}
