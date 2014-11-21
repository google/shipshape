package grok

// This file provides methods to construct Grok tickets for Go AST nodes.

import (
	"fmt"
	"go/ast"
	"log"

	"third_party/kythe/go/util/grokuri"

	"code.google.com/p/go.tools/go/types"
	"code.google.com/p/goprotobuf/proto"
)

// objectTicket returns the ticket for the named Go entity obj.
//
// Example tickets:
//
// Built-ins:
//      nil                     "go:nil"
//      append                  "go:builtin-func:append"
//      true                    "go:builtin-const:true"
//
// File-level objects:
//      fmt                     "go:google3:base.go.log"
//
// Package-level objects:
//      Pi                      "go:math.Pi"
//      atomic.AddInt32         "go:sync/atomic.AddInt32"  TODO verify
//
// All function-local objects:
//      x                       "go:local:x@0x4c2082f9c70"
//      This includes:
//      - local func/var/type/cons instances
//      - all labels
//      - struct fields and interface methods without a fieldOwner
//        (NB: some of the latter may still be accessible outside the
//        package; see populateFieldOwners)
//
func (c *context) objectTicket(obj types.Object) string {
	t, ok := c.objTickets[obj]
	if !ok {
		t = c.objectTicketImpl(obj)
		log.Printf("objectTicket(%s) = %s\n", obj, t)
		c.objTickets[obj] = t
	}
	return t
}

func (c *context) objectTicketImpl(obj types.Object) string {
	// Since blank identifiers cannot be referenced, we skip node
	// generation for most blank objects, but not all: a blank
	// FuncDecl may be "referenced" by containment edges from its
	// internals, and may be referred to by attached comments.
	// So we treat blank idents like package-local identifiers.
	if obj.Name() == "_" {
		return c.langCorpus(fmt.Sprintf("blank:%s@%p", obj.Name(), obj))
	}

	switch obj := obj.(type) {
	case *types.Builtin:
		return lang + "builtin-func:" + obj.Name()

	case *types.Nil:
		return lang + ":nil"

	case *types.PkgName:
		// TODO(adonovan): test this on an unresolved import.
		return c.packageTicket(obj.Pkg())

	case *types.Const:
		if obj.Pkg() == nil {
			return lang + ":builtin-const:" + obj.Name()
		}

	case *types.Var:
		// Struct field?
		if obj.IsField() {
			if owner, ok := c.fieldOwner[obj]; ok {
				// Field of package-level named struct type.
				// Derive the name from the owner.
				return c.objectTicket(owner) + "." + obj.Name()
			}
			return c.langCorpus(fmt.Sprintf("field:%s@%p", obj.Name(), obj))
		}

	case *types.Func:
		// Method?
		if recv := obj.Type().(*types.Signature).Recv(); recv != nil {
			if owner, ok := c.fieldOwner[obj]; ok {
				// Method of package-level named interface type.
				// Derive the name from the owner.
				return c.objectTicket(owner) + "." + obj.Name()
			}

			// Could be:
			// - method of concrete named or *named type.
			// - method of unnamed interface type (even recursive)
			// - method of local named interface type
			// - method of local named concrete type with promoted methods
			return c.langCorpus(fmt.Sprintf("method:(%s).%s", recv.Type(), obj.Name()))
		}

	case *types.TypeName:
		if obj.Pkg() == nil {
			return lang + "builtin-type:" + obj.Name()
		}

	case *types.Label:
		return c.langCorpus(fmt.Sprintf("label:%s@%p", obj.Name(), obj))

	default:
		panic(fmt.Sprintf("unexpected object kind %T", obj))
	}

	// obj is a var/func/const/type/label belonging to some package.
	if obj.Pkg() == nil {
		panic(obj)
	}

	// Objects at package scope are given meaningful names.
	if obj.Parent() == obj.Pkg().Scope() {
		return c.langCorpus(fmt.Sprintf("%s:%s", obj.Pkg().Path(), obj.Name()))
	}

	// Objects local to some function can be given opaque names.
	return c.langCorpus(fmt.Sprintf("local:%s@%p", obj.Name(), obj))
}

func (c *context) langCorpus(s string) string {
	return fmt.Sprintf(lang+":%s:%s", c.Corpus, s)
}

func (c *context) fileTicket(f *ast.File) string {
	return c.makePathTicket(c.pathForNode(f))
}

// makePathTicket constructs a Grok URI ticket for the given path.  This method
// does not check whether the path exists.
func (c *context) makePathTicket(path string) string {
	uri, err := grokuri.ParseDefault(path, c.Corpus)
	if err != nil {
		log.Panicf("unable to construct file URI for %q: %s", path, err)
		return "@" + path
	}
	return uri.String()
}

// makeLocTicket constructs a position-based ticket for the given node.
func (c *context) makeLocTicket(node ast.Node, tag string) string {
	return c.langCorpus(fmt.Sprintf("%s:%s", tag, c.Fset.Position(node.Pos())))
}

// makeAliasTicket constructs an alias ticket for the given display name.
func (c *context) makeAliasTicket(alias string) string {
	return lang + ":ALIAS:" + alias
}

// packageTicket constructs a package ticket for the given package.
// e.g. "go:google3:google3/base/go/log"
func (c *context) packageTicket(pkg *types.Package) string {
	return c.langCorpus(pkg.Path())
}

// makeProtoTicket constructs a generic ticket for a proto message.
func (c *context) makeProtoTicket(tag string, msg proto.Message) string {
	key := "<unknown-proto>"
	if data, err := proto.Marshal(msg); err == nil {
		key = sourceHash(data)
	}
	return fmt.Sprintf(lang+":%s:%s", tag, key)
}

// pathForNode returns the file path containing the given node, as inferred
// from the fileset associated with the context.
func (c *context) pathForNode(node ast.Node) string {
	return c.Fset.File(node.Pos()).Name()
}
