package versionregistry

import (
	"sort"

	"github.com/tokenlive/tokenlive-admin/pkg/productversion"
)

// Group contains only aggregate version information, never node identities.
type Group struct {
	Version   string `json:"version"`
	BuildKind string `json:"build_kind"`
	Count     int    `json:"count"`
}

// GroupNodes groups canonical stable versions by build kind. Unknown and
// uncomparable versions remain visible, and output is ordered by version/kind.
func GroupNodes(nodes []Node) []Group {
	type groupKey struct {
		version   string
		buildKind string
	}
	counts := make(map[groupKey]int)
	for _, node := range nodes {
		version := node.Version
		if version == "" {
			version = "unknown"
		} else if canonical, ok := productversion.StableVersion(version); ok {
			version = canonical
		}
		counts[groupKey{version: version, buildKind: node.BuildKind}]++
	}
	groups := make([]Group, 0, len(counts))
	for key, count := range counts {
		groups = append(groups, Group{Version: key.version, BuildKind: key.buildKind, Count: count})
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Version != groups[j].Version {
			return groups[i].Version < groups[j].Version
		}
		return groups[i].BuildKind < groups[j].BuildKind
	})
	return groups
}
