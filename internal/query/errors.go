package query

import "errors"

// ErrFieldNotFound is returned when a named field is absent from a version.
var ErrFieldNotFound = errors.New("query: field not found")

// IsNotFound reports whether an error means the table is missing.
func IsNotFound(err error) bool {
	return errors.Is(err, errTableNotFound)
}

// IsVersionUnavailable reports whether an error means no readable version.
func IsVersionUnavailable(err error) bool {
	return errors.Is(err, errVersionUnavailable)
}

// Sentinel errors kept internal so callers compare via helpers.
var (
	errTableNotFound     = errors.New("query: table not found")
	errVersionUnavailable = errors.New("query: version unavailable")
)
