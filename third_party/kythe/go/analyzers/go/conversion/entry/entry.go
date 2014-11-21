// Package entry implements a converter that generates a sequence of
// Entry proto messages (defined in storage_proto) from an IndexArtifact
// proto message (defined in internal_proto).
package entry

import (
	"fmt"
	"strconv"
	"strings"

	enumpb "third_party/kythe/proto/enums_proto"
	intpb "third_party/kythe/proto/internal_proto"
	kythepb "third_party/kythe/proto/kythe_proto"
	spb "third_party/kythe/proto/storage_proto"

	"code.google.com/p/goprotobuf/proto"
)

const (
	NodeEntryPrefix   = "/kythe/node/"
	NodeContentPrefix = "/kythe/node/content/"
)

// Converter implements conversion.Converter to emit a sequence of
// Entry proto messages from an IndexArtifact proto message.
type Converter struct {
	BaseVName *spb.VName
}

// Convert implements conversion.Converter interface. It takes an IndexArtifact
// proto message and converts it to a sequence of Entry proto messages.
// It returns an error if the input proto message is not an IndexArtifact or
// if the IndexArtifact is a malformed one.
// The conversion mechanism follows the one used in the Java indexer.
// TODO(mataevs) - remove this and delete this package when there will be
// a better conversion mechanism.
func (c *Converter) Convert(msg proto.Message) ([]proto.Message, error) {
	artifact, ok := msg.(*intpb.IndexArtifact)
	if !ok {
		return nil, fmt.Errorf("Error converting to intpb.IndexArtifact.")
	}

	if artifact.Node != nil {
		return c.nodeToEntries(artifact.Node), nil
	}
	if artifact.PartialEdge != nil {
		return c.partialEdgeToEntries(artifact.PartialEdge), nil
	}
	if artifact.NodeContent != nil {
		return c.nodeContentToEntries(artifact.NodeContent), nil
	}

	return nil, fmt.Errorf("No conversion possible for %v", artifact)
}

// nodeToEntries takes a kythepb.Node and returns a sequence of
// proto messages, one entry for each fact.
func (c *Converter) nodeToEntries(node *kythepb.Node) []proto.Message {
	var entries []proto.Message

	// Construct the source vname
	vname := mergeVNames(c.BaseVName, ticketToVName(node.GetTicket(), node.GetLanguage().String()))

	// Construct entries for kind, identifier, display name
	entries = append(entries,
		entryWithFact(vname, NodeEntryPrefix+"kind", node.GetKind().String()),
		entryWithFact(vname, NodeEntryPrefix+"identifier", node.GetIdentifier()),
		entryWithFact(vname, NodeEntryPrefix+"display_name", node.GetDisplayName()))

	// Add entries for location
	if node.Location != nil {
		entries = append(entries, entryWithFact(vname, NodeEntryPrefix+"location/uri", node.GetLocation().GetUri()))
		entries = append(entries, entriesForSpan(vname, NodeEntryPrefix+"location/", node.GetLocation().GetSpan())...)
	}

	// TODO(mataevs) - emit entries for modifiers

	// Construct node entries for dimensions
	if dims := node.GetDimension(); dims != nil {
		for i, dim := range dims {
			prefix := NodeEntryPrefix + "dimension#" + strconv.Itoa(i) + "/"
			for j, size := range dim.GetSize() {
				entries = append(entries, entryWithFact(vname, prefix+"size#"+strconv.Itoa(j), fmt.Sprintf("%d", size)))
			}
			if dim.Offset != nil {
				entries = append(entries, entryWithFact(vname, prefix+"offset", fmt.Sprintf("%d", dim.GetOffset())))
			}
		}
	}

	// Append entries for node content, if there is one
	if node.Content != nil {
		entries = append(entries, c.nodeContentToEntries(node.GetContent())...)
	}

	// Append entries for pagerank and snippet
	entries = append(entries,
		entryWithFact(vname, NodeEntryPrefix+"pagerank", fmt.Sprintf("%d", node.GetPagerank())),
		entryWithFact(vname, NodeEntryPrefix+"snippet", node.GetSnippet()))

	return entries
}

// nodeContentToEntries takes a kythepb.NodeContent and returns a
// sequence of proto messages, one entry for each fact.
func (c *Converter) nodeContentToEntries(nc *kythepb.NodeContent) []proto.Message {
	var entries []proto.Message

	vname := mergeVNames(c.BaseVName, ticketToVName(nc.GetTicket(), ""))

	if nc.SourceText != nil {
		prefix := NodeContentPrefix + "source_text/"
		text := nc.GetSourceText()

		entries = append(entries,
			entryWithFact(vname, prefix+"encoded_text", string(text.GetEncodedText())),
			entryWithFact(vname, prefix+"encoding", text.GetEncoding()))
		if text.Md5HexString != nil {
			entries = append(entries, entryWithFact(vname, prefix+"md5", text.GetMd5HexString()))
		}
	}

	if nc.Diagnostic != nil {
		prefix := NodeContentPrefix + "diagnostic/"
		diag := nc.GetDiagnostic()

		entries = append(entries,
			entryWithFact(vname, prefix+"type", diag.GetType().String()),
			entryWithFact(vname, prefix+"message", diag.GetMessage()),
			entryWithFact(vname, prefix+"code", diag.GetCode()),
			entryWithFact(vname, prefix+"reporter", diag.GetReporter()))
		if diag.Range != nil {
			entries = append(entries, entriesForSpan(vname, prefix, diag.GetRange())...)
		}
	}

	return entries
}

// partialEdgeToEntries takes an intpb.PartialEdge and returns a
// sequence of proto messages one entry for each fact.
func (c *Converter) partialEdgeToEntries(edge *intpb.PartialEdge) []proto.Message {
	var entries []proto.Message

	// Do not emit entries for reverse edges.
	if ReverseEdge[edge.GetKind()] {
		return entries
	}

	source := mergeVNames(c.BaseVName, ticketToVName(edge.GetStartTicket(), ""))
	target := mergeVNames(source, ticketToVName(edge.GetEndTicket(), ""))

	entry := entryForEdge(source, edge.GetKind().String(), target)
	if edge.Position != nil {
		entry.FactName = proto.String("/kythe/edge/position")
		entry.FactValue = []byte(fmt.Sprintf("%d", edge.GetPosition()))
	}
	entries = append(entries, entry)

	return entries
}

// ticketToVName emits a VName from a ticket string.
func ticketToVName(ticket string, language string) *spb.VName {
	if language == "" {
		language = "go"
	} else {
		language = strings.ToLower(language)
	}

	return &spb.VName{Signature: &ticket, Language: &language}
}

// mergeVNames takes two VNames and returns a new VName, which
// represents a merge of the two - if a property is defined in the
// second VName, then it is used, otherwise the value of the property
// from the first VName is used.
func mergeVNames(vold, vnew *spb.VName) *spb.VName {
	vname := &spb.VName{
		Signature: vold.Signature,
		Corpus:    vold.Corpus,
		Root:      vold.Root,
		Path:      vold.Path,
		Language:  vold.Language,
	}
	if s := vnew.Signature; s != nil {
		vname.Signature = s
	}
	if c := vnew.Corpus; c != nil {
		vname.Corpus = c
	}
	if r := vnew.Root; r != nil {
		vname.Root = r
	}
	if p := vnew.Path; p != nil {
		vname.Path = p
	}
	if l := vnew.Language; l != nil {
		vname.Language = l
	}
	return vname
}

func entryWithFact(source *spb.VName, name, value string) *spb.Entry {
	return &spb.Entry{
		Source:    source,
		FactName:  &name,
		FactValue: []byte(value),
	}
}

// entriesForSpan emits a sequence of entries for a kythepb.Span.
func entriesForSpan(source *spb.VName, prefix string, span *kythepb.Span) []proto.Message {
	var entries []proto.Message

	entries = append(entries, entryWithFact(source, prefix+"span/type", span.GetType().String()))
	if span.GetType() == kythepb.Span_RANGE {
		entries = append(entries, entriesForPosition(source, prefix+"span/start/", span.GetStart())...)
		entries = append(entries, entriesForPosition(source, prefix+"span/end/", span.GetEnd())...)
	} else if span.GetType() == kythepb.Span_POINT {
		entries = append(entries, entriesForPosition(source, prefix+"span/start/", span.GetStart())...)
	}

	return entries
}

func entriesForPosition(source *spb.VName, prefix string, position *kythepb.Position) []proto.Message {
	return []proto.Message{
		entryWithFact(source, prefix+"offset", fmt.Sprintf("%d", position.GetOffset())),
		entryWithFact(source, prefix+"lineno", fmt.Sprintf("%d", position.GetLineno())),
		entryWithFact(source, prefix+"charno", fmt.Sprintf("%d", position.GetCharno())),
	}
}

func entryForEdge(source *spb.VName, kind string, target *spb.VName) *spb.Entry {
	return &spb.Entry{
		Source:   source,
		EdgeKind: &kind,
		Target:   target,
		FactName: proto.String("/"),
	}
}

var ReverseEdge = map[enumpb.EdgeEnum_Kind]bool{
	enumpb.EdgeEnum_EXTENDED_BY:            true,
	enumpb.EdgeEnum_DECLARES:               true,
	enumpb.EdgeEnum_IMPLEMENTED_BY:         true,
	enumpb.EdgeEnum_OVERRIDDEN_BY:          true,
	enumpb.EdgeEnum_DIRECTLY_OVERRIDDEN_BY: true,
	enumpb.EdgeEnum_INHERITED_BY:           true,
	enumpb.EdgeEnum_DIRECTLY_INHERITED_BY:  true,
	enumpb.EdgeEnum_CAPTURED_BY:            true,
	enumpb.EdgeEnum_COMPOSING_TYPE:         true,
	enumpb.EdgeEnum_TYPE_PARAMETER_OF:      true,
	enumpb.EdgeEnum_SPECIALIZED_BY:         true,
	enumpb.EdgeEnum_IS_TYPE_OF:             true,
	enumpb.EdgeEnum_RETURNED_BY:            true,
	enumpb.EdgeEnum_CALLED_AT:              true,
	enumpb.EdgeEnum_INSTANTIATED_AT:        true,
	enumpb.EdgeEnum_REFERENCED_AT:          true,
	enumpb.EdgeEnum_PROPERTY_OF:            true,
	enumpb.EdgeEnum_GENERATES:              true,
	enumpb.EdgeEnum_GENERATES_NAME:         true,
	enumpb.EdgeEnum_HAS_DECLARATION:        true,
	enumpb.EdgeEnum_HAS_DEFINITION:         true,
	enumpb.EdgeEnum_KEY_METHOD:             true,
	enumpb.EdgeEnum_REQUIRED_BY:            true,
	enumpb.EdgeEnum_CONSUMED_BY:            true,
	enumpb.EdgeEnum_HAS_OUTPUT:             true,
	enumpb.EdgeEnum_ALLOWED_ACCESS_TO:      true,
	enumpb.EdgeEnum_CALLGRAPH_FROM:         true,
	enumpb.EdgeEnum_ENCLOSED_USAGE:         true,
	enumpb.EdgeEnum_ANNOTATED_WITH:         true,
	enumpb.EdgeEnum_PARENT:                 true,
	enumpb.EdgeEnum_DIAGNOSTIC_OF:          true,
	enumpb.EdgeEnum_OUTLINE_PARENT:         true,
	enumpb.EdgeEnum_CONTAINS_DECLARATION:   true,
	enumpb.EdgeEnum_CONTAINS_USAGE:         true,
	enumpb.EdgeEnum_THROWN_BY:              true,
	enumpb.EdgeEnum_CAUGHT_BY:              true,
	enumpb.EdgeEnum_THROWGRAPH_FROM:        true,
	enumpb.EdgeEnum_PACKAGE_CONTAINS:       true,
	enumpb.EdgeEnum_VARIABLE_USED_IN:       true,
	enumpb.EdgeEnum_NAMESPACE_CONTAINS:     true,
	enumpb.EdgeEnum_CONTAINS_COMMENT:       true,
	enumpb.EdgeEnum_DOCUMENTED_WITH:        true,
	enumpb.EdgeEnum_TREE_PARENT:            true,
	enumpb.EdgeEnum_XLANG_PROVIDES_NAME:    true,
	enumpb.EdgeEnum_XLANG_PROVIDES:         true,
	enumpb.EdgeEnum_PARAMETER_TYPE_OF:      true,
	enumpb.EdgeEnum_INITIALIZES:            true,
	enumpb.EdgeEnum_HAS_FIGMENT:            true,
	enumpb.EdgeEnum_IS_IDENTIFIER_OF:       true,
	enumpb.EdgeEnum_GUICE_BOUND_AT:         true,
	enumpb.EdgeEnum_GUICE_IMPL_PROVIDED_BY: true,
	enumpb.EdgeEnum_GUICE_INJECTED_AT:      true,
	enumpb.EdgeEnum_HAS_GUICE_CONSTRUCTOR:  true,
}
