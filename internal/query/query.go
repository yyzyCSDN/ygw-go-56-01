package query

import (
	"errors"

	"catalogsvc/internal/catalog"
	"catalogsvc/internal/meta"
	"catalogsvc/internal/model"
)

// Service answers version-aware catalog queries.
type Service struct {
	meta    *meta.Meta
	catalog *catalog.Catalog
	cache   *catalog.Cache
}

// New builds a query service over the given components.
func New(m *meta.Meta, c *catalog.Catalog, cache *catalog.Cache) *Service {
	return &Service{meta: m, catalog: c, cache: cache}
}

// GetTable returns the current registered table through the read cache.
func (s *Service) GetTable(tableID string) (*model.Table, error) {
	table, err := s.cache.Get(tableID, s.catalog.Get)
	if err != nil {
		if errors.Is(err, catalog.ErrNotFound) {
			return nil, errTableNotFound
		}
		return nil, err
	}
	return table, nil
}

// GetFields returns the fields of the currently published version.
func (s *Service) GetFields(tableID string) ([]model.Field, error) {
	resolved, err := s.meta.ResolveVersion(tableID)
	if err != nil {
		if errors.Is(err, meta.ErrVersionUnavailable) {
			return nil, errVersionUnavailable
		}
		return nil, err
	}
	return meta.ParseFields(resolved.Schema), nil
}

// GetField returns one named field from the published version.
func (s *Service) GetField(tableID, name string) (model.Field, error) {
	fields, err := s.GetFields(tableID)
	if err != nil {
		return model.Field{}, err
	}
	field, ok := FindField(fields, name)
	if !ok {
		return model.Field{}, ErrFieldNotFound
	}
	return field, nil
}

// GetFieldsFiltered returns only the requested field names from a version.
func (s *Service) GetFieldsFiltered(tableID string, names []string) ([]model.Field, error) {
	fields, err := s.GetFields(tableID)
	if err != nil {
		return nil, err
	}
	return FilterFields(fields, names), nil
}

// PrimaryFields returns only the primary key fields of a version.
func (s *Service) PrimaryFields(tableID string) ([]model.Field, error) {
	fields, err := s.GetFields(tableID)
	if err != nil {
		return nil, err
	}
	return KeyByID(fields, true), nil
}

// SortedFields returns the version fields ordered by name.
func (s *Service) SortedFields(tableID string) ([]model.Field, error) {
	fields, err := s.GetFields(tableID)
	if err != nil {
		return nil, err
	}
	return OrderFields(fields), nil
}

// ListTables returns every registered table in id order.
func (s *Service) ListTables() []*model.Table {
	return s.catalog.List()
}

// Version returns the current published version number of a table.
func (s *Service) Version(tableID string) (int, error) {
	version, err := s.meta.Current(tableID)
	if err != nil {
		return 0, err
	}
	return version.Number, nil
}

// RefreshCache evicts a table so the next read sees catalog state.
func (s *Service) RefreshCache(tableID string) {
	s.cache.Evict(tableID)
}
