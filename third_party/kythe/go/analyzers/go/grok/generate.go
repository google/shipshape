package grok

// This file defines a visitor on the Go AST that generates Grok index artifacts.

// TODO(fromberger): Blaze allows violation of the standard Go model of package
// structure.  To Go all the files in a given directory that say "package foo"
// are part of that package.  A BUILD file, however, may declare two distinct
// targets sharing some but not all of the same files declaring "package foo"
// -- with the result that a name like a.b.c.foo is not necessarily unique.
//
// And it's not enough that this is possible, but some insanity wolf actually
// took advantage of this to get different build behaviour for AppEngine vs.
// regular code in the core libraries.  This is why we can't have nice things.
//
// Once I get done drinking myself into a coma, I need to figure out a way to
// resolve this without completely breaking the nice naming properties.  In the
// meantime, there are going to be some bugs where tickets crash -- there are a
// couple of real winners in file/base/go.

// TODO(adonovan): in calls to addDecl, can we automatically derive the
// 'unique' flag from the object?

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"strconv"

	"third_party/kythe/go/util/grokuri"

	"code.google.com/p/go.tools/go/types"
	"code.google.com/p/goprotobuf/proto"

	enumpb "third_party/kythe/proto/enums_proto"
	kythepb "third_party/kythe/proto/kythe_proto"
)

const lang = "go"

// An indexWalker is an ast.Visitor that generates Grok index artifacts.
type indexWalker struct {
	*context
	stack []ast.Node // stack of enclosing nodes, last is currently visited node
}

// Visit implements ast.Visitor.
func (w *indexWalker) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		w.stack = w.stack[:len(w.stack)-1] // pop
		return w
	}

	w.stack = append(w.stack, n) // push

	switch n := n.(type) {
	case *ast.File:
		w.visitFile(n)
	case *ast.FuncDecl:
		w.visitFuncDecl(n)
	case *ast.ImportSpec:
		w.visitImportSpec(n)
	case *ast.ValueSpec:
		w.visitValueSpec(n)
	case *ast.TypeSpec:
		w.visitTypeSpec(n)
	case *ast.AssignStmt:
		w.visitAssignStmt(n)
	case *ast.LabeledStmt:
		w.visitLabeledStmt(n)
	case *ast.RangeStmt:
		w.visitRangeStmt(n)
	case *ast.FuncLit:
		w.visitFuncLit(n)
	case *ast.FuncType:
		w.visitFuncType(n)
	case *ast.StructType:
		w.visitStructType(n)
	case *ast.InterfaceType:
		w.visitInterfaceType(n)
	case *ast.Ident:
		w.visitIdent(n)
	}

	return w
}

// visitFile creates the Grok edges between a file and its package.
func (w *indexWalker) visitFile(file *ast.File) {
	// Create a file node.
	node := w.makeNode(w.fileTicket(file), enumpb.NodeEnum_FILE, file)
	if u, err := grokuri.Parse(node.GetLocation().GetUri()); err == nil {
		node.Identifier = proto.String(u.Filename())
		node.DisplayName = proto.String(u.CorpusPath())
	}
	w.fillFileNode(node, w.pathForNode(file))
	w.writeNode(node)

	// Add edges tying the file to its containing package.
	pt := w.packageTicket(w.Package)
	w.addEdges(pt, node.GetTicket(),
		enumpb.EdgeEnum_PACKAGE_CONTAINS, enumpb.EdgeEnum_BELONGS_TO_PACKAGE, noPos)

	// The "package x" decl is a usage of the package.
	pkgdecl := w.makeUsage(file.Name)
	w.writeNode(pkgdecl)
	w.addUsageEdges(pkgdecl)
	w.addEdges(pkgdecl.GetTicket(), pt,
		enumpb.EdgeEnum_REFERENCE, enumpb.EdgeEnum_REFERENCED_AT, noPos)
}

// visitFuncDecl creates a node and edges for a function declaration and its parameters.
func (w *indexWalker) visitFuncDecl(decl *ast.FuncDecl) {
	// Don't reject blank idents, so we can
	// record tree edges from func _ to its children.

	obj, _ := w.TypeInfo.Defs[decl.Name].(*types.Func)
	if obj == nil {
		return // e.g. redefinition
	}
	fNode := w.objectNode(obj, enumpb.NodeEnum_FUNCTION, decl.Name)
	sig := obj.Type().(*types.Signature)

	// Concrete method declaration?
	if sig.Recv() != nil {
		fNode.Kind = enumpb.NodeEnum_METHOD.Enum()

		// Declare the receiver var, iff named.
		if names := decl.Recv.List[0].Names; names != nil {
			r := w.objectNode(sig.Recv(), enumpb.NodeEnum_PARAMETER, names[0])
			w.addDecl(r, notUnique)
			w.addTree(fNode.GetTicket(), r.GetTicket(), 0)
		}

		// Add the method beneath the declaring type in the tree.
		// (This condition is always true in well-typed programs.)
		if named, _ := deref(sig.Recv().Type()).(*types.Named); named != nil {
			w.addTree(w.objectTicket(named.Obj()), fNode.GetTicket(), noPos)
		}
	} else {
		w.fileTree(fNode.GetTicket())
	}
	w.addDecl(fNode, isUnique)
}

// visitFuncLit creates a FUNCTION node for a function literal.
func (w *indexWalker) visitFuncLit(decl *ast.FuncLit) {
	w.addDecl(w.makeNode(w.enclosingFuncTicket(), enumpb.NodeEnum_FUNCTION, decl.Type), notUnique)
}

func (w *indexWalker) visitValueSpec(spec *ast.ValueSpec) {
	kind := enumpb.NodeEnum_VARIABLE
	isLocal := w.isLocal()
	if isLocal {
		kind = enumpb.NodeEnum_LOCAL
	}

	for _, id := range spec.Names {
		if id.Name == "_" {
			continue
		}
		obj := w.TypeInfo.Defs[id]
		if obj == nil {
			continue // type error
		}
		vNode := w.objectNode(obj, kind, id)
		w.addDecl(vNode, !isLocal)
		if !isLocal {
			w.fileTree(vNode.GetTicket())
		}
	}
}

func (w *indexWalker) visitTypeSpec(spec *ast.TypeSpec) {
	if spec.Name.Name == "_" {
		return
	}

	kind := enumpb.NodeEnum_TYPE_ALIAS
	switch spec.Type.(type) {
	case *ast.StructType:
		kind = enumpb.NodeEnum_STRUCT
	case *ast.InterfaceType:
		kind = enumpb.NodeEnum_INTERFACE
	}

	obj, _ := w.TypeInfo.Defs[spec.Name].(*types.TypeName)
	if obj == nil {
		return // type error
	}
	vNode := w.objectNode(obj, kind, spec.Name)
	local := w.isLocal()
	w.addDecl(vNode, !local)
	if !local {
		w.fileTree(vNode.GetTicket())
	}

	// Add tree edges for all struct fields and interface types,
	// including those obtained via promotion/inheritance.
	switch T := obj.Type().Underlying().(type) {
	case *types.Struct:
		// Add tree for all fields, including promoted ones.
		for i, n := 0, T.NumFields(); i < n; i++ {
			w.addTree(vNode.GetTicket(), w.objectTicket(T.Field(i)), noPos)
		}

	case *types.Interface:
		// Add tree for all methods, including inherited ones.
		for i, n := 0, T.NumMethods(); i < n; i++ {
			w.addTree(vNode.GetTicket(), w.objectTicket(T.Method(i)), noPos)
		}
	}
}

func (w *indexWalker) visitStructType(decl *ast.StructType) {
	local := w.isLocal()

	// Add decls for all syntactic fields of all struct types
	// wherever they appear.
	for _, field := range decl.Fields.List {
		// TODO(adonovan): add decls for anonymous fields.
		// Also, consider that the decl could be a pointer
		// and/or a qualified identifier.

		for _, id := range field.Names {
			if id.Name == "_" {
				continue
			}
			obj := w.TypeInfo.Defs[id]
			if obj == nil {
				continue // type error
			}
			node := w.objectNode(obj, enumpb.NodeEnum_FIELD, id)
			w.addDecl(node, !local)
		}
	}
}

func (w *indexWalker) visitInterfaceType(decl *ast.InterfaceType) {
	local := w.isLocal()

	// Add decls for all syntactic methods of all interface types,
	// wherever they appear.
	for _, method := range decl.Methods.List {
		if method.Names == nil {
			continue // embedded interface
		}
		id := method.Names[0]
		obj := w.TypeInfo.Defs[id]
		if obj == nil {
			continue // type error
		}
		node := w.objectNode(obj, enumpb.NodeEnum_METHOD, id)
		w.addDecl(node, !local)
	}
}

func (w *indexWalker) visitImportSpec(spec *ast.ImportSpec) {
	ticket := w.makeLocTicket(spec, "IMPORT")

	var pkgname *types.PkgName
	var n *kythepb.Node
	if spec.Name != nil && spec.Name.Name != "_" {
		// e.g. import x "fmt"
		n = w.makeNode(ticket, enumpb.NodeEnum_IMPORT, spec.Name)
		n.Identifier = proto.String(spec.Name.Name)
		pkgname, _ = w.TypeInfo.Defs[spec.Name].(*types.PkgName)

	} else {
		// e.g. import "fmt"
		n = w.makeNode(ticket, enumpb.NodeEnum_IMPORT, spec.Path)
		s, _ := strconv.Unquote(spec.Path.Value)
		n.Identifier = proto.String(s)
		pkgname, _ = w.TypeInfo.Implicits[spec].(*types.PkgName)
	}

	if pkgname == nil {
		return // type error
	}
	pkg := pkgname.Pkg()
	if pkg == nil {
		return // type error
	}

	w.addUsageEdges(n)

	// The first time we see one of the "magic" intrinsic packages,
	// emit a package node for it -- these packages do not contain any
	// files, but this will permit other files to require them.
	switch pkg.Path() {
	case "unsafe":
		w.ensurePackageNode(pkg, makeGoURI("pkg/unsafe"))
	case "C":
		w.ensurePackageNode(pkg, makeGoURI("cmd/cgo"))
	}

	target := w.packageTicket(pkg)

	// The import site is a reference to the target package.
	w.addEdges(n.GetTicket(), target,
		enumpb.EdgeEnum_REFERENCE, enumpb.EdgeEnum_REFERENCED_AT, noPos)

	// The file requires the target package.
	w.addEdges(w.fileTicket(w.currentFile()), target,
		enumpb.EdgeEnum_REQUIRES, enumpb.EdgeEnum_REQUIRED_BY, noPos)
}

// visitAssignStmt creates a node for each variable declared using the short
// declaration syntax of declaring assignment, e.g. x, y, z := expr.
func (w *indexWalker) visitAssignStmt(stmt *ast.AssignStmt) {
	if stmt.Tok != token.DEFINE {
		return
	}
	for _, lhs := range stmt.Lhs {
		// In well-typed programs, lhs is always an Ident.
		if lhs, ok := lhs.(*ast.Ident); ok {
			if lhs.Name == "_" {
				continue
			}
			if obj := w.TypeInfo.Defs[lhs]; obj != nil {
				if obj.Pos() != lhs.Pos() {
					continue // already defined
				}
				w.declareLocalVar(lhs, obj)
			}
		}
	}
}

func (w *indexWalker) visitLabeledStmt(s *ast.LabeledStmt) {
	if s.Label.Name == "_" {
		return
	}
	obj := w.TypeInfo.Defs[s.Label].(*types.Label)
	if obj == nil {
		return // type error
	}
	n := w.objectNode(obj, enumpb.NodeEnum_LABEL, s.Label)
	w.addDecl(n, notUnique)
}

// visitRangeStmt creates a node for each variable declared using the short
// declaration syntax in a range statement, e.g., for x, y := range z { ... }.
func (w *indexWalker) visitRangeStmt(stmt *ast.RangeStmt) {
	if stmt.Tok != token.DEFINE {
		return
	}
	// In well-typed programs, Key and Value are always Idents.
	if key, ok := stmt.Key.(*ast.Ident); ok {
		if obj := w.TypeInfo.Defs[key]; obj != nil {
			if obj.Pos() == key.Pos() {
				w.declareLocalVar(key, obj)
			}
		}
	}
	if val, ok := stmt.Value.(*ast.Ident); ok {
		if obj := w.TypeInfo.Defs[val]; obj != nil {
			if obj.Pos() == val.Pos() {
				w.declareLocalVar(val, obj)
			}
		}
	}
}

// visitFuncType creates nodes and edges for parameters and results of a
// function type.  Handles both anonymous and named functions.
func (w *indexWalker) visitFuncType(decl *ast.FuncType) {
	// FuncType can appear in three contexts.
	var sig *types.Signature
	switch parent := w.parent(1).(type) {
	case *ast.FuncDecl:
		// Named function/method declaration.
		if obj := w.TypeInfo.Defs[parent.Name]; obj != nil {
			sig = obj.Type().(*types.Signature)
		}

	case *ast.FuncLit:
		// Anonymous function declaration.
		sig, _ = w.TypeInfo.Types[parent].Type.(*types.Signature)

	default:
		// Arbitrary type expression, outside any declaration.
		// We don't need to addDecl the param/result names since
		// they can never be referenced, and we don't bother
		// with tree edges for type syntax outside a
		// declaration (just like struct/interface types).
		return
	}
	if sig == nil {
		return // type error
	}

	enclosing := w.enclosingFuncTicket()

	// Add decls and tree edges for all parameters and results.
	i := 0
	mapFields(decl.Params, func(id *ast.Ident) {
		obj := sig.Params().At(i)
		if obj != nil {
			node := w.objectNode(obj, enumpb.NodeEnum_PARAMETER, id)
			w.addDecl(node, notUnique)
			w.addTree(enclosing, node.GetTicket(), i+1)
		}
		i++
	})
	i = 0
	mapFields(decl.Results, func(id *ast.Ident) {
		obj := sig.Results().At(i)
		if obj != nil {
			n := w.declareLocalVar(id, obj)
			w.addTree(enclosing, n.GetTicket(), noPos)
		}
		i++
	})
}

// visitIdent handles referring identifiers.
// Declaring Idents are handled by the parent syntax.
func (w *indexWalker) visitIdent(id *ast.Ident) {
	obj := w.TypeInfo.Uses[id]
	if obj == nil {
		switch {
		case id.Name == "_":
			// Ignore blanks.
		case w.TypeInfo.Defs[id] != nil:
			// Defining idents are handled by their parent nodes.
		case len(w.stack) == 2:
			// "package foo" has no Uses entry; handled in visitFile.
		default:
			// unresolved reference
		}
		return
	}

	ticket := w.objectTicket(obj)

	// If this ident appears in call position (f()), or
	// beneath a selector in call position (x.f()),
	// its usage of f is as a callee.
	//
	// (This will not match if there are redundant parens.)
	var isCall bool
	if _, ok := obj.(*types.Func); ok {
		if call, ok := w.parent(1).(*ast.CallExpr); ok && call.Fun == id {
			isCall = true // f()
		} else if sel, ok := w.parent(1).(*ast.SelectorExpr); ok && sel.Sel == id {
			if call, ok := w.parent(2).(*ast.CallExpr); ok && call.Fun == sel {
				isCall = true // x.f()
			}
		}
	}

	log.Printf("visitIdent %T > %T > %s (%s) = %t",
		w.parent(2), w.parent(1), id.Name, obj, isCall)

	n := w.addUsage(ticket, id, isCall)

	// Is this a predeclared identifier?
	// (Beware: labels currently have no package.)
	if _, ok := obj.(*types.Label); !ok && obj.Pkg() == nil {
		w.predeclaredNode(ticket, obj)
	} else {
		w.addUsageEdges(n)
	}
}

// fileTree adds DECLARES/DECLARED_BY edges between the current file and the
// specified ticket, including the corresponding edges to the package.
func (w *indexWalker) fileTree(ticket string) {
	w.addTree(w.fileTicket(w.currentFile()), ticket, noPos)
	w.addTree(w.packageTicket(w.Package), ticket, noPos)
}

// isLocal reports whether the currently visited node is local to some function.
func (w *indexWalker) isLocal() bool {
	for _, n := range w.stack {
		switch n.(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			return true
		}
	}
	return false
}

// enclosingFuncTicket returns the ticket for the innermost enclosing
// function.  This is an objectTicket for a named function or an opaque
// ticket for an anonymous function.
// It returns "" if !w.isLocal() or there was a type error.
func (w *indexWalker) enclosingFuncTicket() string {
	var buf bytes.Buffer
	for _, n := range w.stack {
		switch n := n.(type) {
		case *ast.FuncDecl:
			obj := w.TypeInfo.Defs[n.Name]
			if obj == nil {
				return "" // type error
			}
			buf.WriteString(w.objectTicket(obj))
		case *ast.FuncLit:
			fmt.Fprintf(&buf, ".%p", n)
		}
	}
	return buf.String()
}

func (w *indexWalker) currentFile() *ast.File {
	return w.stack[0].(*ast.File)
}

// parent returns the nth parent of the currently visited node:
// 0=self, 1=parent, 2=grandparent, etc.
func (w *indexWalker) parent(n int) ast.Node {
	return w.stack[len(w.stack)-1-n]
}
