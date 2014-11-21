package grok

// This file defines the context type used for analyzing a single target.

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"path/filepath"

	"third_party/kythe/go/analyzers/go/index"
	"third_party/kythe/go/platform/analysis"
	"third_party/kythe/go/analyzers/go/conversion"
	"third_party/kythe/go/analyzers/go/conversion/entry"
	"third_party/kythe/go/util/grokuri"

	"code.google.com/p/go.tools/go/types"
	"code.google.com/p/go.tools/go/types/typeutil"
	"code.google.com/p/goprotobuf/proto"

	enumpb "third_party/kythe/proto/enums_proto"
	intpb "third_party/kythe/proto/internal_proto"
	kythepb "third_party/kythe/proto/kythe_proto"
)

type context struct {
	Indexer      // The indexer whose context this is
	*index.Index // The source index for the compilation unit.

	output     analysis.Sink                    // The sink used to write artifacts.
	converter  conversion.Converter             // Converter for Entry sink
	objTickets map[types.Object]string          // memoization of objectTicket()
	fieldOwner map[types.Object]*types.TypeName // immediately enclosing named type of struct field/interface method
}

func newContext(indexer Indexer, index *index.Index, sink analysis.Sink, emitEntries bool) *context {
	ctx := &context{
		Indexer: indexer,
		Index:   index,

		output:     sink,
		objTickets: make(map[types.Object]string),
		fieldOwner: make(map[types.Object]*types.TypeName),
	}
	if emitEntries {
		ctx.converter = &entry.Converter{index.Compilation.GetVName()}
	}
	return ctx
}

// walkAll applies the given ast.Visitor to each file in the index.
func (c *context) walkAll(v ast.Visitor) {
	for _, file := range c.Files {
		ast.Walk(v, file)
	}
}

// writeMessage sends the specified message to the output, if defined.  If an
// error occurs during writing, it is logged.
func (c *context) writeMessage(msg proto.Message) {
	if msg != nil {
		if err := analysis.WriteMessage(msg, c.output, c.converter); err != nil {
			log.Panicf(err.Error())
		}
	}
}

// makeLoc constructs a Grok Location message spanning the given Go AST node.
func (c *context) makeLoc(node ast.Node) *kythepb.Location {
	if node == nil {
		return nil
	}
	return c.makePosLoc(node.Pos(), node.End())
}

// makePosLoc constructs a Grok location message spanning the given range (if
// end is valid) or range (otherwise) of the file containing pos.
func (c *context) makePosLoc(pos, end token.Pos) *kythepb.Location {
	uri := grokuri.PathURI(c.Fset.Position(pos).Filename, c.Corpus)
	return &kythepb.Location{
		Uri:  proto.String(uri.String()),
		Span: c.makeSpan(pos, end),
	}
}

// makeSpan constructs a Grok Span message spanning the given positions.  The
// span may be a point, a range, or a file.
func (c *context) makeSpan(pos, end token.Pos) *kythepb.Span {
	kind := kythepb.Span_RANGE
	if !pos.IsValid() {
		kind = kythepb.Span_FILE
	} else if !end.IsValid() || pos == end {
		kind = kythepb.Span_POINT
	}
	return &kythepb.Span{
		Type:  kind.Enum(),
		Start: c.makePos(pos),
		End:   c.makePos(end),
	}
}

// spanFromPosition constructs a Grok point Span containing the given Position.
// Returns nil if the position is invalid.
func (c *context) spanFromPosition(pos token.Position) *kythepb.Span {
	if !pos.IsValid() {
		return nil
	}
	return &kythepb.Span{
		Type: kythepb.Span_POINT.Enum(),
		Start: &kythepb.Position{
			Offset: proto.Int(pos.Offset),
			Lineno: proto.Int(pos.Line),
			Charno: proto.Int(pos.Column - 1),
		},
	}
}

// makePos constructs a Grok Position message for the given source position.
// Returns nil if the position is invalid.
func (c *context) makePos(pos token.Pos) *kythepb.Position {
	p := c.Fset.Position(pos)
	if !p.IsValid() {
		return nil
	}
	return &kythepb.Position{
		Offset: proto.Int(p.Offset),
		Lineno: proto.Int(p.Line),
		Charno: proto.Int(p.Column - 1),
	}
}

// makeDiagnostic constructs a node to represent a diagnostic.
func (c *context) makeDiagnostic(diag *kythepb.Diagnostic) *kythepb.Node {
	return &kythepb.Node{
		Kind:     enumpb.NodeEnum_DIAGNOSTIC.Enum(),
		Ticket:   proto.String(c.makeProtoTicket("DIAG", diag)),
		Snippet:  diag.Message,
		Language: enumpb.Language_GO.Enum(),
		Content: &kythepb.NodeContent{
			Diagnostic: diag,
		},
	}
}

// writeNode writes the specified node to the output sink.  It handles
// separating node content from the node, if present.
func (c *context) writeNode(node *kythepb.Node) {
	// Detach and write node content separately.
	nc := node.Content
	node.Content = nil
	c.writeMessage(&intpb.IndexArtifact{
		Node: node,
	})
	if nc != nil {
		nc.Ticket = node.Ticket
		c.writeMessage(&intpb.IndexArtifact{
			NodeContent: nc,
		})
	}
	node.Content = nc
}

// writeEdge writes the specified partial edge to the output sink.
func (c *context) writeEdge(edge *intpb.PartialEdge) {
	c.writeMessage(&intpb.IndexArtifact{
		PartialEdge: edge,
	})
}

// noPos is used to signify an edge with no position value set.
const noPos = -1

// addEdge adds a singe edge source --kind--> target.
// If pos >= 0, the position of the edge is set to that value.
func (c *context) addEdge(source, target string, kind enumpb.EdgeEnum_Kind, pos int) {
	pe := &intpb.PartialEdge{
		Kind:        kind.Enum(),
		StartTicket: &source,
		EndTicket:   &target,
	}
	if pos >= 0 {
		pe.Position = proto.Int(pos)
	}
	c.writeEdge(pe)
}

// addEdges adds the edges source --fKind--> target --rKind--> source.
// If pos >= 0, the positions of the two edges are set to that value.
func (c *context) addEdges(source, target string, fKind, rKind enumpb.EdgeEnum_Kind, pos int) {
	c.addEdge(source, target, fKind, pos)
	c.addEdge(target, source, rKind, pos)
}

// addDiagnostic writes the given diagnostic and edges to associate it with the
// given ticket.
func (c *context) addDiagnostic(target string, diag *kythepb.Diagnostic) {
	node := c.makeDiagnostic(diag)
	node.Location = &kythepb.Location{
		Uri: proto.String(target),
	}
	if diag.Range != nil {
		node.Location.Span = diag.Range
	} else {
		node.Location.Span = &kythepb.Span{
			Type: kythepb.Span_FILE.Enum(),
		}
	}
	c.writeNode(node)
	c.addEdges(target, node.GetTicket(),
		enumpb.EdgeEnum_HAS_DIAGNOSTIC, enumpb.EdgeEnum_DIAGNOSTIC_OF, noPos)
}

// addAlias writes a searchable name for the specified node to the output sink,
// along with the appropriate edges.  Unique indicates whether the searchable
// name is expected to have a unique referent.  If node has no searchable name,
// no alias is added.
func (c *context) addAlias(node *kythepb.Node, unique bool) {
	if node.GetDisplayName() == "" {
		return
	}
	alias := &kythepb.Node{
		Kind:        enumpb.NodeEnum_SEARCHABLE_NAME.Enum(),
		Ticket:      proto.String(c.makeAliasTicket(node.GetDisplayName())),
		Identifier:  node.Identifier,
		DisplayName: node.DisplayName,
		Language:    enumpb.Language_GO.Enum(),
		Modifiers: &kythepb.Modifiers{
			Discrete: proto.Bool(unique),
		},
	}
	c.writeNode(alias)
	c.addEdges(alias.GetTicket(), node.GetTicket(),
		enumpb.EdgeEnum_HAS_DEFINITION, enumpb.EdgeEnum_DEFINITION_OF, noPos)
}

// ensurePackageNode constructs and emits a package node for the specified
// import path, if it has not been emitted.  If uri != "", it is used to make
// an external location for the node.
func (c *context) ensurePackageNode(pkg *types.Package, uri string) {
	ticket := c.packageTicket(pkg)
	if !c.emitted[ticket] {
		c.emitted[ticket] = true
		pNode := &kythepb.Node{
			Ticket:      proto.String(ticket),
			Kind:        enumpb.NodeEnum_PACKAGE.Enum(),
			Identifier:  proto.String(pkg.Name()),
			DisplayName: proto.String(pkg.Path()),
			Language:    enumpb.Language_GO.Enum(),
			Modifiers: &kythepb.Modifiers{
				IsFigment: proto.Bool(true),
			},
		}
		if uri == "" {
			pNode.Location = c.addPackageFigment()
		} else {
			pNode.Location = makeDocLoc(uri)
		}
		c.writeNode(pNode)
		c.addAlias(pNode, isUnique)
		log.Printf("Emit package node for %q", pkg.Path())
	}
}

// addPackageFigment creates a figment (FILE) node to represent the package
// being indexed, and adds references to each of the package's source files.
// Returns the location to attribute to the package.
func (c *context) addPackageFigment() *kythepb.Location {
	figURI := grokuri.New(c.Corpus)
	figURI.Path = filepath.Join("GENERATED/figments/go/pkg", c.Package.Path())

	var buf bytes.Buffer

	span := func(pos, end, line, indent int) *kythepb.Span {
		return &kythepb.Span{
			Type: kythepb.Span_RANGE.Enum(),
			Start: &kythepb.Position{
				Offset: proto.Int(pos),
				Lineno: proto.Int(line),
				Charno: proto.Int(indent),
			},
			End: &kythepb.Position{
				Offset: proto.Int(end),
				Lineno: proto.Int(line),
				Charno: proto.Int(indent + (end - pos)),
			},
		}
	}

	fmt.Fprintf(&buf, `// This file is a figment of Grok's imagination.
// Package: %q
// Docs:    %s
//
package %s
`, c.Package.Path(), makeGoURI("pkg/"+c.Package.Path()), c.Package.Name())

	const (
		// These constants must be updated if the template header changes to
		// ensure the location span is correct.

		pkgLine   = 5 // 1-based line number of the fake package clause
		pkgIndent = 8 // len("package ")
	)

	end := buf.Len() - 1
	pos := end - len(c.Package.Name())

	// Emit references from the figment to each of the package's source files.
	for i, file := range c.Files {
		target := c.fileTicket(file)
		uri, err := grokuri.Parse(target)
		if err != nil {
			panic("invalid file URI")
		}
		pos := buf.Len()
		fmt.Fprintln(&buf, uri.CorpusPath())
		end := buf.Len() - 1
		ticket := c.langCorpus(fmt.Sprintf("USAGE:%s:%d-%d", figURI.CorpusPath(), pos, end))
		c.writeNode(&kythepb.Node{
			Ticket:   proto.String(ticket),
			Kind:     enumpb.NodeEnum_USAGE.Enum(),
			Language: enumpb.Language_GO.Enum(),
			Location: &kythepb.Location{
				Uri:  proto.String(figURI.String()),
				Span: span(pos, end, pkgLine+i+1, 0),
			},
			Snippet: proto.String(string(buf.Bytes()[pos : end+1])),
		})
		c.addEdges(ticket, target, enumpb.EdgeEnum_REFERENCE, enumpb.EdgeEnum_REFERENCED_AT, noPos)
		c.addEdges(ticket, figURI.String(),
			enumpb.EdgeEnum_USAGE_IN_FILE, enumpb.EdgeEnum_CONTAINS_USAGE, noPos)
	}

	// Emit the figment itself.
	c.writeNode(&kythepb.Node{
		Ticket:      proto.String(figURI.String()),
		Kind:        enumpb.NodeEnum_FILE.Enum(),
		Identifier:  proto.String(c.Package.Name()),
		DisplayName: proto.String(c.Package.Path()),
		Language:    enumpb.Language_GO.Enum(),
		Location: &kythepb.Location{
			Uri:  proto.String(figURI.String()),
			Span: &kythepb.Span{Type: kythepb.Span_FILE.Enum()},
		},
		Modifiers: &kythepb.Modifiers{
			IsFigment: proto.Bool(true),
		},
		Content: &kythepb.NodeContent{
			SourceText: &kythepb.SourceText{
				EncodedText: buf.Bytes(),
			},
		},
	})
	c.addEdges(figURI.String(), c.packageTicket(c.Package),
		enumpb.EdgeEnum_DECLARES, enumpb.EdgeEnum_DECLARED_BY, noPos)
	c.addEdges(figURI.String(), c.packageTicket(c.Package),
		enumpb.EdgeEnum_CONTAINS_DECLARATION, enumpb.EdgeEnum_DECLARATION_IN_FILE, noPos)
	return &kythepb.Location{
		Uri:  proto.String(figURI.String()),
		Span: span(pos, end, pkgLine, pkgIndent),
	}
}

// displayName returns the displayed name of the specified named Go
// object, which is used in the outline display and the search index.
// It need not be unique.
func (c *context) displayName(obj types.Object) string {
	if obj.Pkg() == nil {
		return obj.Name() // universal object (e.g. true, int, append)
	}

	displayName := obj.Pkg().Path() + "."

	// Which scope?
	switch {
	case obj.Pkg().Scope() == obj.Parent():
		// package scope

	case obj.Parent() == nil:
		// field or method

		switch obj := obj.(type) {
		case *types.Var:
			// field
			if owner, ok := c.fieldOwner[obj]; ok {
				displayName += owner.Name() + "."
			} else {
				displayName += "(unnamed struct)."
			}

		case *types.Func:
			// method
			recv := obj.Type().(*types.Signature).Recv()
			if _, ok := recv.Type().Underlying().(*types.Interface); ok {
				if owner, ok := c.fieldOwner[obj]; ok {
					displayName += owner.Name() + "."
				} else {
					displayName += "(unnamed interface)."
				}
			} else {
				rt := recv.Type()
				if ptr, ok := rt.(*types.Pointer); ok {
					displayName += "*"
					rt = ptr.Elem()
				}
				displayName += rt.(*types.Named).Obj().Name() + "."
			}

		default:
			panic(obj) // unexpected object with no parent scope
		}

	default:
		// file or local scope

		// It's fine to have duplicate names for local objects
		// since they are marked notUnique, but local objects' names
		// must not conflict with package-level objects's names,
		// which are marked isUnique.
		displayName += "(local)."
	}

	return displayName + obj.Name()
}

// populateFieldOwners populates the fieldOwners mapping from fields of
// package-level named struct types to the owning named struct type, and
// from methods of package-level named interface types to the owning
// named interface type.
//
// This relation is required when we construct tickets and display names
// for these fields/methods, since they may be referenced from another
// package, thus they need a deterministic name, but the object does
// expose its "owner"; indeed, it may have several.
//
// Caveats:
//
// (1) This mapping is deterministic but not necessarily the best one
// according to the original syntax, to which, in general, we do not have
// access.  In these two examples, the type checker considers field X as
// belonging equally to types T and U, even though according the syntax,
// it belongs primarily to T in the first example and U in the second:
//
//      type T struct {X int}
//      type U T
//
//      type T U
//      type U struct {X int}
//
// (2) This pass is not exhaustive: there remain objects that may be
// referenced from outside the package but for which we can't easily
// come up with good names.  Here are some examples:
//
//      // package p
//      var V1, V2 struct {X int} = ...
//      func F() struct{X int} {...}
//      type T struct {
//              Y struct { X int }
//      }
//
//      // main
//      p.V2.X = 1
//      print(p.F().X)
//      new(p.T).Y[0].X
//
// Also note that there may be arbitrary pointer, struct, chan, map,
// array, and slice type constructors between the type of the exported
// package member (V2, F or T) and the type of its X subelement.
//
// For now, we simply ignore such names.  They should be rare in readable code.
//
func (c *context) populateFieldOwners() {
	// The (undefined) order among packages doesn't matter.
	for pkg := range c.AllPackages {
		// Within a package we use the arbitrary but
		// deterministic name-based ordering.
		scope := pkg.Scope()
		for _, name := range scope.Names() {
			if obj, ok := scope.Lookup(name).(*types.TypeName); ok {
				switch T := obj.Type().Underlying().(type) {
				case *types.Struct:
					for i := 0; i < T.NumFields(); i++ {
						f := T.Field(i)
						if f.Pkg() != pkg {
							continue // wrong package
						}
						// Save the first-encountered owner.
						if _, ok := c.fieldOwner[f]; !ok {
							c.fieldOwner[f] = obj
						}
					}

				case *types.Interface:
					for i := 0; i < T.NumMethods(); i++ {
						m := T.Method(i)
						if m.Pkg() != pkg {
							continue // wrong package
						}
						// Save the first-encountered owner.
						if _, ok := c.fieldOwner[m]; !ok {
							c.fieldOwner[m] = obj
						}
					}
				}
			}
		}
	}
}

// typeInfo emits edges for the method set of each type and the set of
// interfaces it satisfies.  It is wholly independent of the syntax.
func (c *context) typeInfo() {
	// Find all named types mentioned in this unit.
	var allNamed []*types.TypeName
	for _, info := range c.AllPackages {
		for _, obj := range info.Defs {
			if obj, ok := obj.(*types.TypeName); ok {
				allNamed = append(allNamed, obj)
			}
		}
	}

	// Record the method set for each named type defined in this package.
	//
	// Since there's nothing in the CodeSearch UI representing *T
	// that users can click on, we use IntuitiveMethodSet and
	// ignore the pointerness of the receiver type.
	var msets types.MethodSetCache
	for _, xobj := range allNamed {
		if xobj.Pkg() != c.Package {
			continue
		}

		x := xobj.Type()

		xmset := msets.MethodSet(x)
		if xmset.Len() == 0 {
			continue // no methods
		}

		// For all named types x with methods, defined in this package...

		// Add an "x INHERITS xm" edge for each of its methods xm.
		// (We do this for all methods, whether declared, inherited, or abstract.)
		xticket := c.objectTicket(xobj)
		for _, xm := range typeutil.IntuitiveMethodSet(x, &msets) {
			c.addEdges(xticket, c.objectTicket(xm.Obj()),
				enumpb.EdgeEnum_INHERITS, enumpb.EdgeEnum_INHERITED_BY, noPos)
		}

		if isInterface(x) {
			// Add "y IMPLEMENTS x" edges for each interface x that
			// y or *y is assignable to.
			//
			// This is quadratic, but probably ok since we're only
			// looking at all the types within a single compilation unit.
			for _, yobj := range allNamed {
				if xobj == yobj {
					continue
				}

				y := yobj.Type()

				// Is y (or *y) assignable to interface x?
				// We treat both facts the same.
				var ymset *types.MethodSet
				if types.AssignableTo(y, x) {
					ymset = msets.MethodSet(y)
				} else if !isInterface(y) {
					py := types.NewPointer(y)
					if types.AssignableTo(py, x) {
						ymset = msets.MethodSet(py)
					}
				}
				if ymset == nil {
					continue // not assignable
				}

				// Add "y IMPLEMENTS x" edge.
				yticket := c.objectTicket(yobj)
				c.addEdges(yticket, xticket,
					enumpb.EdgeEnum_IMPLEMENTS, enumpb.EdgeEnum_IMPLEMENTED_BY, noPos)

				if !isInterface(y) {
					// Add "ym IMPLEMENTS xm" edge for each
					// concrete method ym and corresponding
					// abstract method xm.
					//
					// May result in duplicate edges.  This input:
					//      type I interface {f()}
					//      type T int
					//      func (T) f() {}
					//      type U struct {T}
					// produces these edges:
					//      ym                    xm
					//      (T).f --IMPLEMENTS--> (I).f
					//      (U).f --IMPLEMENTS--> (I).f
					// but the underlying Obj() of xm and ym is (T).f
					// in both cases.  I'm not sure if this is a problem.
					for i, n := 0, xmset.Len(); i < n; i++ {
						xm := xmset.At(i)
						ym := ymset.Lookup(xm.Obj().Pkg(), xm.Obj().Name())
						if ym == nil {
							panic(xm)
						}
						xmticket := c.objectTicket(xm.Obj())
						ymticket := c.objectTicket(ym.Obj())
						c.addEdges(ymticket, xmticket,
							enumpb.EdgeEnum_IMPLEMENTS, enumpb.EdgeEnum_IMPLEMENTED_BY, noPos)
					}
				}
			}
		}
	}
}

func isInterface(T types.Type) bool {
	_, ok := T.Underlying().(*types.Interface)
	return ok
}
