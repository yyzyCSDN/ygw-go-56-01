package catalog

import "github.com/cespare/xxhash/v2"

// TableKey produces a stable 64-bit fingerprint for a table identity.
func TableKey(id, name string) uint64 {
	return xxhash.Sum64String(id + "\x00" + name)
}

// EdgeKey produces a stable fingerprint for a lineage edge pair.
func EdgeKey(from, to string) uint64 {
	return xxhash.Sum64String(from + "\x00" + to)
}

// FingerprintSchema produces a fingerprint of an ordered field list.
func FingerprintSchema(fields []string) uint64 {
	var digest xxhash.Digest
	for _, name := range fields {
		_, _ = digest.WriteString(name)
		_, _ = digest.WriteString("\x00")
	}
	return digest.Sum64()
}
