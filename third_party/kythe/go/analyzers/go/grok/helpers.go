package grok

// This file defines helper methods on the indexWalker type for generating Grok
// index artifacts.

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"go/ast"
	"log"

	"code.google.com/p/go.tools/go/types"
	"code.google.com/p/goprotobuf/proto"

	enumpb "third_party/kythe/proto/enums_proto"
	kythepb "third_party/kythe/proto/kythe_proto"
)

// Constants for the "unique" flag to addAlias.
const (
	isUnique  = true
	notUnique = false
)

func nodeMods(node *kythepb.Node) *kythepb.Modifiers {
	if node.Modifiers == nil {
		node.Modifiers = new(kythepb.Modifiers)
	}
	return node.Modifiers
}

// addDecl writes the specified node to the output sink along with a searchable
// name and declaration-in-file edges for that node.  Unique indicates whether
// the display name for the node is expected to have a unique referent.
func (w *indexWalker) addDecl(node *kythepb.Node, unique bool) {
	// Anything that is being marked as a declaration should also have its
	// definition modifier set.
	nodeMods(node).Definition = proto.Bool(true)

	w.writeNode(node)
	file := node.GetLocation().GetUri()
	// Self-definition loop.
	w.addEdge(node.GetTicket(), node.GetTicket(), enumpb.EdgeEnum_HAS_DEFINITION, noPos)
	// Denormalization for file decorations.
	w.addEdges(file, node.GetTicket(),
		enumpb.EdgeEnum_CONTAINS_DECLARATION, enumpb.EdgeEnum_DECLARATION_IN_FILE, noPos)
	// Searchable name.
	w.addAlias(node, unique)
}

// addUsage creates, writes and returns a new usage node for the
// specified target along with its reference edges.
// If isCall, the reference is treated as a function call.
// Usage-in-file nodes are not written by this method; see addUsageEdges.
func (w *indexWalker) addUsage(target string, id *ast.Ident, isCall bool) *kythepb.Node {
	usage := w.makeUsage(id)
	w.writeNode(usage)

	log.Printf("addUsage %s %s %t", target, id.Name, isCall)

	// TODO(fromberger): Do we want to treat anything as an instantiation?  It
	// might make sense for the leading name in a composite literal expression.
	if isCall {
		w.addEdges(usage.GetTicket(), target, enumpb.EdgeEnum_CALL, enumpb.EdgeEnum_CALLED_AT, noPos)
	} else {
		w.addEdges(usage.GetTicket(), target, enumpb.EdgeEnum_REFERENCE, enumpb.EdgeEnum_REFERENCED_AT, noPos)
	}

	// Add usage context and callgraph edges, where appropriate.
	if ct := w.enclosingFuncTicket(); ct != "" {
		w.addEdges(usage.GetTicket(), ct, enumpb.EdgeEnum_USAGE_CONTEXT, enumpb.EdgeEnum_ENCLOSED_USAGE, noPos)
		if isCall {
			w.addEdges(ct, target, enumpb.EdgeEnum_CALLGRAPH_TO, enumpb.EdgeEnum_CALLGRAPH_FROM, noPos)
		}
	}

	return usage
}

// addUsageEdges adds the usage-in-file edges for the given usage node.
func (w *indexWalker) addUsageEdges(usage *kythepb.Node) {
	file := usage.GetLocation().GetUri()
	w.addEdges(file, usage.GetTicket(),
		enumpb.EdgeEnum_CONTAINS_USAGE, enumpb.EdgeEnum_USAGE_IN_FILE, noPos)
}

// addTree writes "tree" edges (DECLARES/DECLARED_BY) between parent and child.
// If pos >= 0 the position of the edge is also set.
func (w *indexWalker) addTree(parent, child string, pos int) {
	w.addEdges(parent, child, enumpb.EdgeEnum_DECLARES, enumpb.EdgeEnum_DECLARED_BY, pos)
}

// makeNode constructs a node with the given ticket and kind.
// The optional location determines the nodes location and snippet.
func (w *indexWalker) makeNode(ticket string, kind enumpb.NodeEnum_Kind, loc ast.Node) *kythepb.Node {
	return &kythepb.Node{
		Kind:     kind.Enum(),
		Ticket:   &ticket,
		Language: enumpb.Language_GO.Enum(),
		Location: w.makeLoc(loc),
		Snippet:  w.makeSnippet(loc),
	}
}

// objectNode returns a new node of the specified kind for the named Go entity obj.
// id must be the object's defining identifier.
func (w *indexWalker) objectNode(obj types.Object, kind enumpb.NodeEnum_Kind, id *ast.Ident) *kythepb.Node {
	if id.Pos() != obj.Pos() {
		panic("not a defining ident")
	}
	n := w.makeNode(w.objectTicket(obj), kind, id)
	n.Identifier = proto.String(id.Name)
	n.DisplayName = proto.String(w.displayName(obj))
	return n
}

// fillFileNode populates the fields of a Grok FILE node from source text.
func (w *indexWalker) fillFileNode(node *kythepb.Node, path string) {
	if node.Location == nil {
		node.Location = &kythepb.Location{Uri: node.Ticket}
	}
	node.Location.Span = &kythepb.Span{
		Type: kythepb.Span_FILE.Enum(),
	}
	source := w.Text[path]
	node.Content = &kythepb.NodeContent{
		SourceText: &kythepb.SourceText{
			EncodedText:  source,
			Encoding:     proto.String("UTF-8"),
			Md5HexString: proto.String(sourceHash(source)),
		},
	}
}

// makeUsage constructs a usage node for the specified (referring) identifier.
func (w *indexWalker) makeUsage(id *ast.Ident) *kythepb.Node {
	usage := w.makeNode(w.makeLocTicket(id, "USAGE"), enumpb.NodeEnum_USAGE, id)
	usage.Identifier = proto.String(id.Name)
	return usage
}

// makeSnippet constructs a snippet string surrounding the given source position.
// Returns a pointer to the snippet, or nil if none could be constructed.
func (w *indexWalker) makeSnippet(node ast.Node) *string {
	if node == nil {
		return nil
	}
	pos := w.Fset.Position(node.Pos())
	if !pos.IsValid() || w.Text[pos.Filename] == nil {
		return nil
	}
	data := w.Text[pos.Filename]
	lo := pos.Offset - pos.Column
	if lo < 0 {
		lo = 0
	}
	snip := bytes.TrimSpace(data[lo:])
	hi := bytes.IndexRune(snip, '\n')
	if hi < 0 {
		hi = len(snip)
	}
	snip = snip[:hi]

	// Make sure snippets don't get too long.  This happens sometimes with
	// generated code, and it blows up the size of the stored edge sets.
	const maxSnippetLen = 100
	for len(snip) > maxSnippetLen {
		// Try to chop off whitespace-delimited fields on the right to get
		// something in a reasonable range.  If that proves impossible, then
		// just focus a window around the point of interest.
		if i := bytes.LastIndexAny(snip, " \t"); i > pos.Column {
			snip = bytes.TrimSpace(snip[:i])
		} else {
			// The dimensions of the window are ad hoc; just something to
			// ensure the point of interest is somewhere inside it.
			min := pos.Column - maxSnippetLen/2
			if min < 0 {
				min = 0
			}
			max := pos.Column + maxSnippetLen/4
			if max > len(snip) {
				max = len(snip)
			}
			snip = snip[min:max]
		}
	}

	return proto.String(string(snip))
}

func (w *indexWalker) declareLocalVar(id *ast.Ident, obj types.Object) *kythepb.Node {
	_ = obj.(*types.Var) // assertion
	n := w.objectNode(obj, enumpb.NodeEnum_LOCAL, id)
	w.addDecl(n, notUnique)
	return n
}

// predeclaredNode emits the grok node for the predeclared object obj,
// if not already emitted.  ticket must be equal to objectTicket(obj).
func (w *indexWalker) predeclaredNode(ticket string, obj types.Object) {
	// Emit nodes for predeclared func/const/type objects on demand.
	// The context remembers which ones we've done before, so we'll
	// only emit them once per analysis.
	if w.emitted[ticket] {
		return
	}
	w.emitted[ticket] = true

	name := obj.Name()

	var kind enumpb.NodeEnum_Kind
	var docURI string
	var numeric *types.Basic
	switch obj := obj.(type) {
	case *types.TypeName:
		switch name {
		case "byte", "uint8", "uint16", "uint32", "uint64", "uint",
			"rune", "int8", "int16", "int32", "int64", "int":
			kind = enumpb.NodeEnum_INTEGER
		case "string":
			kind = enumpb.NodeEnum_STRING
		case "float32", "float64":
			kind = enumpb.NodeEnum_FLOAT
		case "bool":
			kind = enumpb.NodeEnum_BOOLEAN
		case "complex64", "complex128":
			kind = enumpb.NodeEnum_COMPLEX
		case "error":
			kind = enumpb.NodeEnum_INTERFACE
		}

		if T, ok := obj.Type().Underlying().(*types.Basic); ok {
			if T.Info()&types.IsNumeric != 0 {
				numeric = T
			}
		}
		docURI = makeGoURI(fmt.Sprintf("pkg/builtin#%s", name))

	case *types.Builtin:
		kind = enumpb.NodeEnum_FUNCTION
		docURI = makeGoURI(fmt.Sprintf("pkg/builtin#%s", name))

	case *types.Nil:
		kind = enumpb.NodeEnum_VALUE
		docURI = makeGoURI("ref/spec#The_zero_value")

	case *types.Const:
		kind = enumpb.NodeEnum_VALUE
		if name == "iota" {
			docURI = makeGoURI("ref/spec#Iota")
		} else {
			docURI = makeGoURI("ref/spec#Predeclared_identifiers")
		}

	case *types.Func:
		// Must be (error).Error().
		kind = enumpb.NodeEnum_FUNCTION
		docURI = makeGoURI("pkg/builtin#error")

	default:
		panic(obj)
	}

	node := w.makeNode(ticket, kind, nil)
	node.Identifier = proto.String(name)
	node.DisplayName = proto.String(name)
	node.Location = makeDocLoc(docURI)

	if numeric != nil {
		// Set the appropriate signedess modifiers.
		node.Modifiers = new(kythepb.Modifiers)
		if numeric.Info()&types.IsUnsigned != 0 {
			node.Modifiers.Unsigned = proto.Bool(true)
		} else {
			node.Modifiers.Signed = proto.Bool(true)
		}

		// This assumes a 64-bit platform; ideally we would take
		// this from the indexer configuration somehow.
		node.Dimension = append(node.Dimension, &kythepb.Dimension{
			Size: []int32{8 * int32((&types.StdSizes{8, 8}).Sizeof(numeric))},
		})
	}

	w.writeNode(node)
	w.addAlias(node, isUnique)

	log.Printf("Emitted node for predeclared object %s", obj)
}

// makeDocLoc constructs a Grok Location message to an external URI.
func makeDocLoc(uri string) *kythepb.Location {
	return &kythepb.Location{
		Uri: proto.String(uri),
		Span: &kythepb.Span{
			Type: kythepb.Span_EXTERNAL.Enum(),
		},
	}
}

// sourceHash computes the hexadecimal hash string for the specified source.
func sourceHash(source []byte) string {
	h := md5.New()
	h.Write(source)
	return hex.EncodeToString(h.Sum(nil))
}

// mapFields applies f to each identifier declared in fields.
func mapFields(fields *ast.FieldList, f func(*ast.Ident)) {
	if fields != nil {
		for _, field := range fields.List {
			for _, id := range field.Names {
				f(id)
			}
		}
	}
}

// makeGoURI assembles a Go site URI given the path and query components as a string.
func makeGoURI(tail string) string {
	return "http://godoc.corp.google.com/" + tail
}

func deref(T types.Type) types.Type {
	if U, ok := T.Underlying().(*types.Pointer); ok {
		return U.Elem()
	}
	return T
}
