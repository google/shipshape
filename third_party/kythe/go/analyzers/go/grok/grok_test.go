package grok

import (
	"fmt"

	"third_party/kythe/go/platform/local"

	"code.google.com/p/goprotobuf/proto"

	apb "third_party/kythe/proto/analysis_proto"
	intpb "third_party/kythe/proto/internal_proto"
	kythepb "third_party/kythe/proto/kythe_proto"
	spb "third_party/kythe/proto/storage_proto"
)

// input represents a required input for a test compilation.
type input struct {
	path, text string
	isSource   bool
}

// sourceInput returns a source input constructed from the given text.
func sourceInput(path string, text string) input {
	return input{
		path:     path,
		text:     text,
		isSource: true,
	}
}

// fakeImport returns a fake archive import for the given package name.
func fakeImport(path, name string) input {
	return input{
		path: path,
		text: fmt.Sprintf("go object linux amd64\nimport\n$$\npackage %s\n$$", name),
	}
}

// makeCompilation creates a self-contained local compilation with the given
// signature and a single source file, suitable for testing
func makeCompilation(signature string, inputs ...input) *local.Compilation {
	c := local.NewCompilation()
	c.Proto.VName = &spb.VName{Language: proto.String("go"), Corpus: proto.String("google3")}
	c.Proto.GoArguments = new(apb.GoArguments)
	c.SetSignature(signature)
	for _, input := range inputs {
		c.AddData(input.path, []byte(input.text))
		if input.isSource {
			c.SetSource(input.path)
		}
	}
	return c
}

// makeRequest creates an AnalysisRequest from a local compilation.
func makeRequest(unit *local.Compilation) *apb.AnalysisRequest {
	return &apb.AnalysisRequest{Compilation: unit.Proto}
}

// A testSink implements analysis.Sink to capture index artifacts for testing.
// This sink provides methods to query the stored artifacts in various ways; in
// that respect, it simulates some of the features of the Grok API.
//
// To select nodes:
//   sink.allNodes(filter)   -- all nodes matched by a nodeFilter.
//   sink.someNode(filter)   -- some node matched by a nodeFilter.
//   sink.uniqueNode(filter) -- the unique node matched by a nodeFilter.
//   sink.allEdges(filter)   -- all edges matched by an edgeFilter.
//   sink.someEdge(filter)   -- some edge matched by an edgeFilter.
//   sink.uniqueEdge(filter) -- the unique edge matched by an edgeFilter.
//
type testSink struct {
	nodes   []*kythepb.Node
	edges   []*intpb.PartialEdge
	content []*kythepb.NodeContent
}

// WriteBytes implements the required method of analysis.Sink.
func (t *testSink) WriteBytes(data []byte) error {
	art := new(intpb.IndexArtifact)
	if err := proto.Unmarshal(data, art); err != nil {
		return err
	}
	if art.Node != nil {
		t.nodes = append(t.nodes, art.Node)
	}
	if art.PartialEdge != nil {
		t.edges = append(t.edges, art.PartialEdge)
	}
	if art.NodeContent != nil {
		t.content = append(t.content, art.NodeContent)
	}
	return nil
}

// postprocess modifies the stored artifacts to associate nodes with their node
// content and populate the start and end nodes of edges, wherever possible.
// This simulates part of the postprocessing machinery to make testing easier.
func (t *testSink) postprocess() {
	nmap := make(map[string]*kythepb.Node)
	for _, node := range t.nodes {
		nmap[node.GetTicket()] = node
	}

	for _, edge := range t.edges {
		if start := nmap[edge.GetStartTicket()]; start != nil {
			edge.StartNode = start
		}
		if end := nmap[edge.GetEndTicket()]; end != nil {
			edge.EndNode = end
		}
	}

	for _, nc := range t.content {
		if node := nmap[nc.GetTicket()]; node != nil {
			node.Content = nc
		}
	}
}

// A nodeFilter selects Node messages having arbitrary properties.
// A filter can be composed with other filters to select nodes with complex
// properties, or you can provide your own custom filter as a function.
type nodeFilter func(*kythepb.Node) bool

var node nodeFilter

// anyNode selects any Node message.
func anyNode(*kythepb.Node) bool { return true }

// and makes a nodeFilter that selects a node if n1 and n2 both select it.
// n1 may be nil but n2 may not.
func (n1 nodeFilter) and(n2 nodeFilter) nodeFilter {
	if n1 == nil {
		return n2
	}
	return func(n *kythepb.Node) bool {
		return n1(n) && n2(n)
	}
}

// withKind is a nodeFilter that selects nodes with the given kind.
// Kinds are matched by spelling.
func (n nodeFilter) withKind(kind string) nodeFilter {
	return n.and(func(node *kythepb.Node) bool {
		return node.GetKind().String() == kind
	})
}

// withIdent is a nodeFilter that selects nodes with the given identifier.
func (n nodeFilter) withIdent(id string) nodeFilter {
	return n.and(func(node *kythepb.Node) bool {
		return node.GetIdentifier() == id
	})
}

// withName is a nodeFilter that selects nodes with the given display name.
func (n nodeFilter) withName(name string) nodeFilter {
	return n.and(func(node *kythepb.Node) bool {
		return node.GetDisplayName() == name
	})
}

// withOffset is a nodeFilter that selects nodes whose start position has the
// given offset.
func (n nodeFilter) withOffset(pos int) nodeFilter {
	return n.and(func(node *kythepb.Node) bool {
		return node.GetLocation().GetSpan().GetStart().GetOffset() == int32(pos)
	})
}

// withLine is a nodeFilter that selects nodes located on the given 1-based line.
func (n nodeFilter) withLine(line int) nodeFilter {
	return n.and(func(node *kythepb.Node) bool {
		return node.GetLocation().GetSpan().GetStart().GetLineno() == int32(line)
	})
}

// withURI is a nodeFilter that selects nodes whose location URI is given.
func (n nodeFilter) withURI(uri string) nodeFilter {
	return n.and(func(node *kythepb.Node) bool {
		return node.GetLocation().GetUri() == uri
	})
}

// withTicket is a nodeFilter that selects nodes with the given ticket.
func (n nodeFilter) withTicket(ticket string) nodeFilter {
	return n.and(func(node *kythepb.Node) bool {
		return node.GetTicket() == ticket
	})
}

// withContent is a nodeFilter that selects nodes with the given content.
func (n nodeFilter) withContent(c string) nodeFilter {
	return n.and(func(node *kythepb.Node) bool {
		return string(node.GetContent().GetSourceText().GetEncodedText()) == c
	})
}

// isExactly is a nodeFilter that selects exactly the specified node.
func (n nodeFilter) isExactly(want *kythepb.Node) nodeFilter {
	return n.and(func(node *kythepb.Node) bool {
		return node == want
	})
}

// An edgeFilter selects PartialEdge messages having arbitrary properties.
type edgeFilter func(*intpb.PartialEdge) bool

var edge edgeFilter

// and makes an edgeFilter that selects an edge if e and e2 both select it.
// e1 may be nil but e2 may not.
func (e1 edgeFilter) and(e2 edgeFilter) edgeFilter {
	if e1 == nil {
		return e2
	}
	return func(e *intpb.PartialEdge) bool {
		return e1(e) && e2(e)
	}
}

// withKind is an edgeFilter that selects edges with the given kind.
// Kinds are matched by spelling.
func (e edgeFilter) withKind(kind string) edgeFilter {
	return e.and(func(edge *intpb.PartialEdge) bool {
		return edge.GetKind().String() == kind
	})
}

// fromNode is an edgeFilter that selects edges starting at the given node.
func (e edgeFilter) fromNode(node *kythepb.Node) edgeFilter {
	return e.and(func(edge *intpb.PartialEdge) bool {
		return edge.StartNode == node
	})
}

// between is an edgeMatcher that selects edges whose start node matches start
// and whose end node matches end.
func (e edgeFilter) between(start, end nodeFilter) edgeFilter {
	return e.and(func(edge *intpb.PartialEdge) bool {
		return start(edge.StartNode) && end(edge.EndNode)
	})
}

// allNodes returns all nodes satisfying nf, or nil if no such nodes exist.
func (t *testSink) allNodes(nf nodeFilter) []*kythepb.Node {
	var found []*kythepb.Node
	for _, node := range t.nodes {
		if nf(node) {
			found = append(found, node)
		}
	}
	return found
}

// countNodes returns the number of nodes satisfying nf, or 0 if no such nodes exist.
func (t *testSink) countNodes(nf nodeFilter) int {
	return len(t.allNodes(nf))
}

// countEdges returns the number of edges satisfying ef, or 0 if no such edges exist.
func (t *testSink) countEdges(ef edgeFilter) int {
	return len(t.allEdges(ef))
}

// allEdges returns all edges satisfying ef, or nil if no such edges exist.
func (t *testSink) allEdges(ef edgeFilter) []*intpb.PartialEdge {
	var found []*intpb.PartialEdge
	for _, edge := range t.edges {
		if ef(edge) {
			found = append(found, edge)
		}
	}
	return found
}

// uniqueNode returns the single node satisfying nf.  Returns nil if such a
// node does not exist or if there is more than one.
func (t *testSink) uniqueNode(nf nodeFilter) *kythepb.Node {
	if nodes := t.allNodes(nf); len(nodes) == 1 {
		return nodes[0]
	}
	return nil
}

// uniqueEdge returns the single edge satisfying ef.  Returns nil if such an
// edge does not exist or if there is more than one.
func (t *testSink) uniqueEdge(ef edgeFilter) *intpb.PartialEdge {
	if edges := t.allEdges(ef); len(edges) == 1 {
		return edges[0]
	}
	return nil
}

// someNode returns some single node satisfying nf, or nil if none exists.
func (t *testSink) someNode(nf nodeFilter) *kythepb.Node {
	for _, node := range t.nodes {
		if nf(node) {
			return node
		}
	}
	return nil
}

// someEdge returns some single edge satisfying ef, or nil if none exists.
func (t *testSink) someEdge(ef edgeFilter) *intpb.PartialEdge {
	for _, edge := range t.edges {
		if ef(edge) {
			return edge
		}
	}
	return nil
}

// An edgeTree represents a tree structure of nodes, stored as an adjacency
// list.  The key is the source node, the values are the edges departing from
// that node.  Thus this is essentially a poor-man's EdgeSet.
type edgeTree map[*kythepb.Node][]*intpb.PartialEdge

// add adds the given edge as an outbound of from if it does not already exist.
func (e edgeTree) add(from *kythepb.Node, edge *intpb.PartialEdge) {
	for _, old := range e[from] {
		if old == edge {
			return
		}
	}
	e[from] = append(e[from], edge)
}

// expand returns the transitive closure of nodes reachable by starting from
// the nodes selected by start and recursively applying next until no further
// expansion can be done.  Returns a map from each node to its outbound edges
// under the relation.
//
// Requires that postprocess has been run.
//
// Example:
//   tree := sink.expand(node.withIdent("foo"), edge.withKind("TREE_PARENT"))
//
func (t *testSink) expand(start nodeFilter, next edgeFilter) edgeTree {
	result := make(edgeTree)
	seen := make(map[*kythepb.Node]bool)
	queue := t.allNodes(start)
	for len(queue) > 0 {
		var head *kythepb.Node
		queue, head = queue[:len(queue)-1], queue[len(queue)-1]

		for _, edge := range t.allEdges(edge.fromNode(head).and(next)) {
			if tail := edge.GetEndNode(); !seen[tail] {
				result.add(head, edge)
				queue = append(queue, tail)
				seen[tail] = true
			}
		}
	}
	return result
}

// edgePair returns a pair of edges between start and end with the specified
// forward and reverse kinds.  Either edge may be nil if not found.
func (t *testSink) edgePair(start, end nodeFilter, fKind, rKind string) (fwd, rev *intpb.PartialEdge) {
	fwd = t.someEdge(edge.withKind(fKind).between(start, end))
	rev = t.someEdge(edge.withKind(rKind).between(end, start))
	return
}

// withPosition is an edgeFilter that selects edges with a given position.
func (e edgeFilter) withPosition(pos int) edgeFilter {
	return e.and(func(edge *intpb.PartialEdge) bool {
		return edge.GetPosition() == int32(pos)
	})
}
