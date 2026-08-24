package lineage

import (
	"sort"

	"catalogsvc/internal/model"
	"github.com/cespare/xxhash/v2"
)

// HashEdgeKey hashes a directed edge into a 64-bit bucket value.
func HashEdgeKey(from, to string) uint64 {
	return xxhash.Sum64String(from + ">" + to)
}

// EdgeDigest produces a stable fingerprint for an unordered set of edges.
func EdgeDigest(edges []model.LineageEdge) uint64 {
	keys := make([]string, 0, len(edges))
	for _, edge := range edges {
		keys = append(keys, edge.Key())
	}
	sort.Strings(keys)
	var digest xxhash.Digest
	for _, key := range keys {
		_, _ = digest.WriteString(key)
		_, _ = digest.WriteString("\x00")
	}
	return digest.Sum64()
}

// Fingerprint returns the digest of the graph's current edge set.
func (g *Graph) Fingerprint() uint64 {
	return EdgeDigest(g.Edges())
}
