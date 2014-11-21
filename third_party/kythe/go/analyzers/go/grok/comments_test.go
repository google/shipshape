package grok

import (
	"fmt"
	"testing"

	kythepb "third_party/kythe/proto/kythe_proto"
)

func TestCommentEdges(t *testing.T) {
	unit := makeCompilation(testTarget, sourceInput("mahFile", `
// mahFile is a package
package foo

// Foo is a function
func Foo() bool { return false }

// Bar is a variable
var Bar string

// Taz is a type
type Taz struct {
	// Fleep is a field
	Fleep int
}

// Morg is a method
func (Taz) Morg() {}

const (
	// Carl is a constant
	Carl = "der Große"

	// Callie is a constant
	Callie = 15
)`))

	indexer := NewIndexer()
	var sink testSink
	if err := indexer.Analyze(makeRequest(unit), unit, &sink); err != nil {
		t.Fatal(err)
	}
	sink.postprocess()

	tests := []struct {
		name, kind, label string
	}{
		{"mahFile", "FILE", "package"},
		{"Foo", "FUNCTION", "function"},
		{"Bar", "VARIABLE", "variable"},
		{"Taz", "STRUCT", "type"},
		{"Fleep", "FIELD", "field"},
		{"Morg", "METHOD", "method"},
		{"Carl", "VARIABLE", "constant"},
		{"Callie", "VARIABLE", "constant"},
	}

	for _, test := range tests {
		source := sink.uniqueNode(node.withIdent(test.name).withKind(test.kind))
		if source == nil {
			t.Errorf("Missing %q node %q", test.kind, test.name)
			continue
		}
		link := sink.someEdge(edge.fromNode(source).withKind("DOCUMENTED_WITH"))
		if link == nil {
			t.Errorf("Missing %q ---DOCUMENTED_WITH--> ? edge", source.GetTicket())
			continue
		}
		comment := link.GetEndNode()
		if kind, want := comment.GetKind().String(), "COMMENT"; kind != want {
			t.Errorf("Wrong target kind: got %q, want %q", kind, want)
			continue
		}
		back := sink.someEdge(edge.fromNode(comment).withKind("DOCUMENTS"))
		if back == nil {
			t.Errorf("Missing %q ---DOCUMENTS--> %q", comment.GetTicket(), source.GetTicket())
			continue
		}
		if comment.GetLocation().GetUri() == "" {
			t.Errorf("The location is empty for comment %q", comment.GetTicket())
		}

		// Verify that the comment has the expected text.
		want := fmt.Sprintf("// %s is a %s", test.name, test.label)
		got := string(comment.GetContent().GetSourceText().GetEncodedText())
		if got != want {
			t.Errorf("Comment text: got %q, want %q", got, want)
		}
	}

	// Also verify that the package node gets connected to its documentation.
	fwd, rev := sink.edgePair(node.withKind("PACKAGE"), node.withKind("COMMENT"),
		"DOCUMENTED_WITH", "DOCUMENTS")
	if fwd == nil {
		t.Error("Missing edge from package node to comment")
	}
	if rev == nil {
		t.Error("Missing edge from comment to package node")
	}
}

func TestCommentCorners(t *testing.T) {
	unit := makeCompilation(testTarget, sourceInput("mahFile", `
package foo
func ForeverAlone() {} // Has no doc comment

// Comment line 1
// Comment line 2
var Multiple bool`))

	indexer := NewIndexer()
	var sink testSink
	if err := indexer.Analyze(makeRequest(unit), unit, &sink); err != nil {
		t.Fatal(err)
	}
	sink.postprocess()

	alone := sink.uniqueNode(node.withIdent("ForeverAlone").withKind("FUNCTION"))
	if alone == nil {
		t.Fatal("Missing function ForeverAlone")
	}
	if link := sink.someEdge(edge.fromNode(alone).withKind("DOCUMENTED_WITH")); link != nil {
		t.Errorf("Unexpected edge %q ---%s--> %q", link.GetStartTicket(), link.GetKind().String(),
			link.GetEndTicket())
	}

	const multiText = "// Comment line 1\n// Comment line 2"
	multi := sink.uniqueNode(node.withIdent("Multiple").withKind("VARIABLE"))
	if multi == nil {
		t.Fatal("Missing variable Multiple")
	}
	if link := sink.someEdge(edge.fromNode(multi).withKind("DOCUMENTED_WITH")); link == nil {
		t.Errorf("Missing edge %q ---DOCUMENTED_WITH--> ?", multi.GetTicket())
	} else if text := string(link.GetEndNode().GetContent().GetSourceText().GetEncodedText()); text != multiText {
		t.Errorf("Comment text: got %q, want %q", text, multiText)
	}
}

func (sink *testSink) findLink(t *testing.T, src *kythepb.Node, kind string) *kythepb.Node {
	if link := sink.someEdge(edge.fromNode(src).withKind(kind)); link != nil {
		return link.EndNode
	}
	t.Fatalf("Missing %q ---%s--> ? edge", src.GetTicket(), kind)
	return nil
}

func TestCommentStructure(t *testing.T) {
	unit := makeCompilation(testTarget, sourceInput(testPath, `
package foo

// A comment with multiple lines
// what fun that will be
func F() {}

/* Give that variable some block comments.
 Variables love block comments. */
var V int`))

	indexer := NewIndexer()
	var sink testSink
	if err := indexer.Analyze(makeRequest(unit), unit, &sink); err != nil {
		t.Fatal(err)
	}
	sink.postprocess()

	tests := []struct {
		id, raw, stripped string
	}{
		{"F", "// A comment with multiple lines\n// what fun that will be",
			"A comment with multiple lines\nwhat fun that will be"},
		{"V", "/* Give that variable some block comments.\n Variables love block comments. */",
			"Give that variable some block comments.\nVariables love block comments. "},
	}

	for _, test := range tests {
		node := sink.someNode(node.withIdent(test.id))
		if node == nil {
			t.Errorf("Missing node for identifier %q", test.id)
			continue
		}
		com := sink.findLink(t, node, "DOCUMENTED_WITH")
		doc := sink.findLink(t, com, "TREE_PARENT")
		para := sink.findLink(t, doc, "TREE_PARENT")
		txt := sink.findLink(t, para, "TREE_PARENT")

		raw := string(com.GetContent().GetSourceText().GetEncodedText())
		if raw != test.raw {
			t.Errorf("Raw comment %q: got %q, want %q", com.GetTicket(), raw, test.raw)
		}
		cooked := string(txt.GetContent().GetSourceText().GetEncodedText())
		if cooked != test.stripped {
			t.Errorf("Stripped comment %q: got %q, want %q", txt.GetTicket(), cooked, test.stripped)
		}
	}
}

func TestCommentParagraphs(t *testing.T) {
	unit := makeCompilation(testTarget, sourceInput(testPath, `
package foo

// Give that variable some block comments.
// Variables love block comments.
//
// Give that comment some paragraphs
// Comments love paragraphs
var V int`))

	indexer := NewIndexer()
	var sink testSink
	if err := indexer.Analyze(makeRequest(unit), unit, &sink); err != nil {
		t.Fatal(err)
	}
	sink.postprocess()

	vNode := sink.someNode(node.withIdent("V"))
	if vNode == nil {
		t.Errorf("Missing node for identifier V")
	}
	com := sink.findLink(t, vNode, "DOCUMENTED_WITH")
	doc := sink.findLink(t, com, "TREE_PARENT")
	if got, want := sink.countEdges(edge.fromNode(doc).withKind("TREE_PARENT")), 2; got != want {
		t.Errorf("Parent edges: got %d, want %d", got, want)
	}

	paragraphs := sink.allNodes(node.withIdent("p").withKind("MARKUP_TAG").and(isOpenTag))
	if got, want := len(paragraphs), 2; got != want {
		t.Errorf("<p> tags: got %d, want %d", got, want)
	}
	if got, want := sink.countNodes(node.withIdent("p").withKind("MARKUP_TAG").and(isCloseTag)), 2; got != want {
		t.Errorf("</p> tags: got %d, want %d", got, want)
	}

	// Each paragraph must have two children.
	for _, p := range paragraphs {
		if got, want := sink.countEdges(edge.fromNode(p).withKind("TREE_PARENT")), 2; got != want {
			t.Errorf("Parent edges from paragraph: got %d, want %d", got, want)
		}
		txtEdge := sink.uniqueEdge(edge.fromNode(p).withKind("TREE_PARENT").withPosition(0))
		if nil == sink.uniqueNode(node.withTicket(txtEdge.EndNode.GetTicket()).withKind("TEXT")) {
			t.Error("Text node is nil.")
		}
		endTagEdge := sink.uniqueEdge(edge.fromNode(p).withKind("TREE_PARENT").withPosition(1))
		if nil == sink.uniqueNode(node.withTicket(endTagEdge.EndNode.GetTicket()).withKind("MARKUP_TAG")) {
			t.Error("End tag is nil.")
		}
	}
}

func TestCommentPre(t *testing.T) {
	unit := makeCompilation(testTarget, sourceInput(testPath, `
package foo

// Example use:
//   V = 10
//   fmt.Println("%v", V)
var V int`))

	indexer := NewIndexer()
	var sink testSink
	if err := indexer.Analyze(makeRequest(unit), unit, &sink); err != nil {
		t.Fatal(err)
	}
	sink.postprocess()

	vNode := sink.someNode(node.withIdent("V"))
	if vNode == nil {
		t.Errorf("Missing node for identifier V")
	}
	com := sink.findLink(t, vNode, "DOCUMENTED_WITH")
	doc := sink.findLink(t, com, "TREE_PARENT")
	if got, want := sink.countEdges(edge.fromNode(doc).withKind("TREE_PARENT")), 2; got != want {
		t.Errorf("Parent edges: got %d, want %d", got, want)
	}

	paragraphs := sink.allNodes(node.withIdent("p").withKind("MARKUP_TAG").and(isOpenTag))
	if got, want := len(paragraphs), 1; got != want {
		t.Errorf("<p> tags: got %d, want %d", got, want)
	}
	if got, want := sink.countNodes(node.withIdent("p").withKind("MARKUP_TAG").and(isCloseTag)), 1; got != want {
		t.Errorf("</p> tags: got %d, want %d", got, want)
	}
	// Each paragraph must have two children.
	for _, p := range paragraphs {
		if got, want := sink.countEdges(edge.fromNode(p).withKind("TREE_PARENT")), 2; got != want {
			t.Errorf("Parent edges from paragraph: got %d, want %d", got, want)
		}
		txtEdge := sink.uniqueEdge(edge.fromNode(p).withKind("TREE_PARENT").withPosition(0))
		if txtEdge == nil {
			t.Error("Text edge is nil.")
		}
		if nil == sink.uniqueNode(node.withTicket(txtEdge.EndNode.GetTicket()).withKind("TEXT")) {
			t.Error("Text is nil.")
		}
		endTagEdge := sink.uniqueEdge(edge.fromNode(p).withKind("TREE_PARENT").withPosition(1))
		if nil == sink.uniqueNode(node.withTicket(endTagEdge.EndNode.GetTicket()).withKind("MARKUP_TAG")) {
			t.Error("End tag is nil.")
		}
	}

	preTag := sink.uniqueNode(node.withIdent("pre").withKind("MARKUP_TAG").and(isOpenTag))
	if nil == sink.uniqueEdge(edge.withKind("TREE_PARENT").withPosition(0).between(
		node.withTicket(preTag.GetTicket()),
		node.withKind("TEXT").withContent("V = 10\nfmt.Println(&#34;%v&#34;, V)"))) {
		t.Errorf("Pre text edge does not exist.")
	}
	if nil == sink.uniqueEdge(edge.withKind("TREE_PARENT").withPosition(1).between(
		node.withTicket(preTag.GetTicket()), node.withKind("MARKUP_TAG").and(isCloseTag))) {
		t.Errorf("End pre does not exist.")
	}
}

func TestCommentHeading(t *testing.T) {
	unit := makeCompilation(testTarget, sourceInput(testPath, `
package foo

// Testing the heading
//
// Header
//
// Give that function some comments. Functions love comments.
func Fun() string { return "" }`))

	indexer := NewIndexer()
	var sink testSink
	if err := indexer.Analyze(makeRequest(unit), unit, &sink); err != nil {
		t.Fatal(err)
	}
	sink.postprocess()

	funNode := sink.someNode(node.withIdent("Fun"))
	if funNode == nil {
		t.Errorf("Missing node for identifier Fun")
	}
	com := sink.findLink(t, funNode, "DOCUMENTED_WITH")
	doc := sink.findLink(t, com, "TREE_PARENT")
	if got, want := sink.countEdges(edge.fromNode(doc).withKind("TREE_PARENT")), 3; got != want {
		t.Errorf("Parent edges: got %d, want %d", got, want)
	}

	if got, want := sink.countNodes(node.withIdent("p").withKind("MARKUP_TAG").and(isOpenTag)), 2; got != want {
		t.Errorf("<p> tags: got %d, want %d", got, want)
	}
	if got, want := sink.countNodes(node.withIdent("p").withKind("MARKUP_TAG").and(isCloseTag)), 2; got != want {
		t.Errorf("</p> tags: got %d, want %d", got, want)
	}
	// There must be one heading
	heads := sink.allNodes(node.withIdent("h3").withKind("MARKUP_TAG").and(isOpenTag))
	if got, want := len(heads), 1; got != want {
		t.Errorf("<h3> tags: got %d, want %d", got, want)
	}
	if got, want := sink.countNodes(node.withIdent("h3").withKind("MARKUP_TAG").and(isCloseTag)), 1; got != want {
		t.Errorf("</h3> tags: got %d, want %d", got, want)
	}
	if nil == sink.uniqueEdge(edge.withKind("TREE_PARENT").withPosition(0).between(
		node.withTicket(heads[0].GetTicket()),
		node.withKind("TEXT").withContent("Header"))) {
		t.Errorf("Head text edge does not exist.")
	}
}

func isOpenTag(n *kythepb.Node) bool {
	return n.GetModifiers().GetOpenDelimiter()
}

func isCloseTag(n *kythepb.Node) bool {
	return n.GetModifiers().GetCloseDelimiter()
}
