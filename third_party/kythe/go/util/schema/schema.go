// Package schema defines constants used in the Kythe schema.
package schema

import "strings"

// Kythe node fact labels
const (
	NodeKindFact = "/kythe/node/kind"

	AnchorStartFact = "/kythe/loc/start"
	AnchorEndFact   = "/kythe/loc/end"

	FileTextFact     = "/kythe/text"
	FileEncodingFact = "/kythe/text/encoding"
)

// Kythe node kinds
const (
	AnchorKind = "anchor"
	FileKind   = "file"
	NameKind   = "name"
)

// Kythe edge kinds
const (
	edgePrefix = "/kythe/edge/"

	ChildOfEdge = edgePrefix + "childof"
	DefinesEdge = edgePrefix + "defines"
	RefEdge     = edgePrefix + "ref"
)

// reverseEdgePrefix is the Kythe edgeKind prefix for reverse edges.  Edge kinds
// must be prefixed at most once with this string.
// TODO(schroederc): along with the edge kinds, node kinds, etc. add this to a
//                   schema library
const reverseEdgePrefix = "%"

// EdgeDir represents the inherent direction of an edge kind.
type EdgeDir bool

// Forward edges are generally depedency edges and ensure that each node has a
// small out-degree in the Kythe graph.  Reverse edges are the opposite.
const (
	Forward EdgeDir = true
	Reverse EdgeDir = false
)

// EdgeDirection returns the edge direction of the given edge kind
func EdgeDirection(kind string) EdgeDir {
	if strings.HasPrefix(kind, reverseEdgePrefix) {
		return Reverse
	}
	return Forward
}

// MirrorEdge returns the reverse edge kind for a given forward edge kind and
// returns the forward edge kind for a given reverse edge kind.
func MirrorEdge(kind string) string {
	if rev := strings.TrimPrefix(kind, reverseEdgePrefix); rev != kind {
		return rev
	}
	return reverseEdgePrefix + kind
}
