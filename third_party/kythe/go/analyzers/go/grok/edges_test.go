package grok

import (
	"log"
	"strings"
	"testing"

	enumpb "third_party/kythe/proto/enums_proto"
)

// TODO(fromberger): References to assignment-declared variables do not
// always work yet, due to missing features in the Go resolver.  Once that
// is fixed, add a test for it.
// e.g.
// 	xx, err := 0, nil
// 	yy, err := 0, nil
// 	print(xx, yy, err)  // xx, err refer to first; yy to second.
// Needs a different assertion structure.

// TODO(adonovan): add a test that the package references marked below
// have edges to the correct package node:
//
// 	import "lib"
// 	import anotherlib "lib"
// 	var _ lib.T	    // <--here
// 	var _ anotherlib.T  // <--here
//
// 	{"lib.T", "USAGE", "lib", "PACKAGE", refKind},
// 	{"anotherlib.T", "lib", "USAGE", "PACKAGE", refKind},
//
// We can't currently test this in TestReferenceEdges because we don't
// have nodes for imported packages.  We need an assertion on tickets,
// not nodes.

const testTarget = "//dummy/test:target"
const testPath = "test.go"

// Verify that tree edges (DECLARES/DECLARED_BY) are populated.
func TestTreeEdges(t *testing.T) {
	unit := makeCompilation(testTarget, sourceInput(testPath, `
package foo
type T int
var V T
func (t T) G(z int) {}
type I interface {
  Foo()
}
type S struct {
  A, B int
  C    string
}`))

	indexer := NewIndexer()
	var sink testSink
	if err := indexer.Analyze(makeRequest(unit), unit, &sink); err != nil {
		t.Fatal(err)
	}
	sink.postprocess()

	const fKind = "DECLARES"
	const rKind = "DECLARED_BY"

	tests := []struct {
		startID, startKind, endID, endKind string
	}{
		// Top-level outline for the file.
		{testPath, "FILE", "T", "TYPE_ALIAS"},
		{testPath, "FILE", "V", "VARIABLE"},
		{testPath, "FILE", "I", "INTERFACE"},

		// Top-level outline for the package.
		{"foo", "PACKAGE", "T", "TYPE_ALIAS"},
		{"foo", "PACKAGE", "V", "VARIABLE"},
		{"foo", "PACKAGE", "I", "INTERFACE"},

		{"G", "METHOD", "t", "PARAMETER"},
		{"G", "METHOD", "z", "PARAMETER"},
		{"I", "INTERFACE", "Foo", "METHOD"},
		{"T", "TYPE_ALIAS", "G", "METHOD"},
		{"S", "STRUCT", "A", "FIELD"},
		{"S", "STRUCT", "B", "FIELD"},
		{"S", "STRUCT", "C", "FIELD"},
	}
	for _, test := range tests {
		fwd, rev := sink.edgePair(node.withIdent(test.startID).withKind(test.startKind),
			node.withIdent(test.endID).withKind(test.endKind), fKind, rKind)
		if fwd == nil {
			t.Errorf("Missing edge %q --%s--> %q", test.startID, fKind, test.endID)
		}
		if rev == nil {
			t.Errorf("Missing edge %q <--%s-- %q", test.startID, rKind, test.endID)
		}
	}

	// The file should not declare edges to the methods of an interface, only
	// explicit method definitions.
	fwd, rev := sink.edgePair(node.withIdent(testPath), node.withIdent("Foo"), fKind, rKind)
	if fwd != nil {
		t.Errorf("Unexpected forward edge: %+v", fwd)
	}
	if rev != nil {
		t.Errorf("Unexpected reverse edge: %+v", rev)
	}
}

// Verify that edges between a file and its contained declarations are populated.
func TestFileDeclarations(t *testing.T) {
	unit := makeCompilation(testTarget, sourceInput(testPath, `
package foo
func F(z int) bool {
  return z == 0
}
var V string
const C = 2.997e8
type S struct {
  B byte
}
`))

	indexer := NewIndexer()
	var sink testSink
	if err := indexer.Analyze(makeRequest(unit), unit, &sink); err != nil {
		t.Fatal(err)
	}
	sink.postprocess()

	const fKind = "CONTAINS_DECLARATION"
	const rKind = "DECLARATION_IN_FILE"

	tests := []struct {
		name, kind string
	}{
		{"F", "FUNCTION"},
		{"z", "PARAMETER"},
		{"V", "VARIABLE"},
		{"C", "VARIABLE"},
		{"S", "STRUCT"},
		{"B", "FIELD"},
	}

	// Confirm that there is a declaration of the appropriate type linked to
	// the file node.
	file := node.withIdent(testPath).withKind("FILE")
	for _, test := range tests {
		fwd, rev := sink.edgePair(file, node.withKind(test.kind).withIdent(test.name), fKind, rKind)
		if fwd == nil {
			t.Errorf("Missing edge %q --%s--> %q", testPath, fKind, test.name)
		}
		if rev == nil {
			t.Errorf("Missing edge %q <--%s-- %q", testPath, rKind, test.name)
		}
	}
}

// Verify that edges between a file and its contained usages are populated.
func TestFileUsages(t *testing.T) {
	const testSource = `package foo
func F(z int) bool {
  return z == 0 || F(z-1)
}
var V string
const C = 2.997e8
type S struct {
  B byte
}
func U() {
  s := S{'p'}
  println(V, C, s.B)
}`

	unit := makeCompilation(testTarget, sourceInput(testPath, testSource))
	indexer := NewIndexer()
	var sink testSink
	if err := indexer.Analyze(makeRequest(unit), unit, &sink); err != nil {
		t.Fatal(err)
	}
	sink.postprocess()

	const fKind = "CONTAINS_USAGE"
	const rKind = "USAGE_IN_FILE"

	// Confirm that there is a USAGE node at each of the given locations, and
	// that the file points to it.
	file := node.withIdent(testPath).withKind("FILE")
	for _, needle := range []string{"foo", "z == 0", "F(z-1)", "z-1", "S{'p'}", "V, C", "C, s", "s.B", "B)"} {
		pos := strings.Index(testSource, needle)
		if pos < 0 {
			t.Errorf("Broken test: missing needle %q", needle)
			continue
		}
		fwd, rev := sink.edgePair(file, node.withOffset(pos), fKind, rKind)
		if fwd == nil {
			t.Errorf("Missing edge %q --%s--> %q", testPath, fKind, needle)
		}
		if rev == nil {
			t.Errorf("Missing edge %q <--%s-- %q", testPath, rKind, needle)
		}
	}
}

// Verify that appropriate declarations have searchable names.
func TestSearchableNames(t *testing.T) {
	unit := makeCompilation(testTarget, sourceInput(testPath, `
package foo
type T int
type S struct { SF string }
func F(x int) bool {
  for _, p := range []int{} {
    _ = p
  }
  return false
}
func (S) M() float32 { return 0.0 }
func (*S) M2() float32 { return 0.0 }
`))

	indexer := NewIndexer()
	var sink testSink
	if err := indexer.Analyze(makeRequest(unit), unit, &sink); err != nil {
		t.Fatal(err)
	}
	sink.postprocess()

	const nameBase = "google3/dummy/test/target"
	const fKind = "HAS_DEFINITION"
	const rKind = "DEFINITION_OF"

	tests := []struct {
		id, kind, name string
	}{
		{"foo", "PACKAGE", nameBase},
		{"T", "TYPE_ALIAS", nameBase + ".T"},
		{"S", "STRUCT", nameBase + ".S"},
		{"SF", "FIELD", nameBase + ".S.SF"},
		{"F", "FUNCTION", nameBase + ".F"},
		{"x", "PARAMETER", nameBase + ".(local).x"},
		{"M", "METHOD", nameBase + ".S.M"},
	}

	// Log a summary of available searchable names.
	for _, n := range sink.allNodes(node.withKind("SEARCHABLE_NAME")) {
		t.Logf("SN %+v", n)
		if n.GetDisplayName() == "" {
			t.Errorf("Searchable name has empty display_name: %+v", n)
		}
	}

	for _, test := range tests {
		name := sink.someNode(node.withKind("SEARCHABLE_NAME").withName(test.name))
		if name == nil {
			t.Errorf("Missing searchable name for %q", test.name)
			continue
		}
		fwd, rev := sink.edgePair(node.isExactly(name), node.withKind(test.kind).withIdent(test.id), fKind, rKind)
		if fwd == nil {
			t.Errorf("Missing edge %q --%s--> %q", test.id, fKind, test.name)
		}
		if rev == nil {
			t.Errorf("Missing edge %q <--%s-- %q", test.id, rKind, test.name)
		}

		// Most nodes with searchable names should also be marked as definitions.
		// TODO(fromberger): We might want the package to be a definition too.
		if end := fwd.GetEndNode(); !end.GetModifiers().GetDefinition() {
			if end.GetKind() != enumpb.NodeEnum_PACKAGE {
				t.Errorf("Target node is not marked as a definition: %+v", end)
			}
		}
	}
}

// Verify that all nodes have the language field correctly set.
func TestNodesHaveLanguage(t *testing.T) {
	unit := makeCompilation(testTarget, sourceInput(testPath, `
// This comment has a node too.
package foo
func bar() string { return "blah" }
type baz int
func (b *baz) blort(z int) string {
  return bar()
}
`))

	indexer := NewIndexer()
	var sink testSink
	if err := indexer.Analyze(makeRequest(unit), unit, &sink); err != nil {
		t.Fatal(err)
	}

	if len(sink.nodes) == 0 {
		t.Error("No nodes generated for this compilation")
	}
	for _, node := range sink.nodes {
		if node.Language == nil {
			t.Errorf("Node missing language: %+v", node)
		} else if v, want := node.GetLanguage(), enumpb.Language_GO; v != want {
			t.Errorf("Node language: got %v, want %v", v, want)
		}
	}
}

// Verify that diagnostics get their language field set.
func TestDiagnosticsHaveLanguage(t *testing.T) {
	unit := makeCompilation(testTarget, sourceInput(testPath, `
package foo
import "missing"`))

	indexer := NewIndexer()
	var sink testSink
	indexer.Analyze(makeRequest(unit), unit, &sink)

	if len(sink.nodes) == 0 {
		t.Error("No nodes generated for this compilation")
	}
	for _, node := range sink.nodes {
		if node.Language == nil {
			t.Errorf("Node missing language: %+v", node)
		} else if v, want := node.GetLanguage(), enumpb.Language_GO; v != want {
			t.Errorf("Node language: got %v, want %v", v, want)
		}
	}
}

// Verify that call graph edges are populated between calling contexts and call
// targets.
func TestCallGraphEdges(t *testing.T) {
	unit := makeCompilation(testTarget, sourceInput(testPath, `
package foo
func F1() bool {
  return !F2() // call to F2
}
func F2() bool {
  return false
}
func R(x int) int {
  if x == 0 {
    return 1
  }
  return x * R(x - 1)
}
type T byte
func (t T) M() T {
  if F1() { // call to F1
    return t + 1
  } else if F2() { // call to F2
    return t + 2
  }
  return t
}`))

	indexer := NewIndexer()
	var sink testSink
	if err := indexer.Analyze(makeRequest(unit), unit, &sink); err != nil {
		t.Fatal(err)
	}
	sink.postprocess()

	const (
		fKind   = "FUNCTION"
		mKind   = "METHOD"
		callFwd = "CALLGRAPH_TO"
		callRev = "CALLGRAPH_FROM"
	)

	tests := []struct {
		src, sKind, tgt, tKind string
	}{
		{"F1", fKind, "F2", fKind},
		{"M", mKind, "F1", fKind},
		{"M", mKind, "F2", fKind},
	}
	for _, test := range tests {
		fwd, rev := sink.edgePair(node.withIdent(test.src).withKind(test.sKind),
			node.withIdent(test.tgt).withKind(test.tKind), callFwd, callRev)
		if fwd == nil {
			t.Errorf("Missing edge %q --%s--> %q", test.src, callFwd, test.tgt)
		}
		if rev == nil {
			t.Errorf("Missing edge %q <--%s-- %q", test.src, callRev, test.tgt)
		}
	}
}

// Verify that reference edges (references, calls) are generated.
func TestReferenceEdges(t *testing.T) {
	const testSource = `
package main
type T int
var V = T(25)
func F(t T, u bool) T {
  return t + 1
}
var s = struct{f func()}{f: func() {}}
func main() {
  var q = V // ref
  F(q, true)
  s.f()
}
func (T) m() {}
func init() {
foo:
	T.m(0)
	V.m()//V.m
	print(s.f)
	s.f()
	goto foo//goto
}
`

	unit := makeCompilation(testTarget,
		sourceInput(testPath, testSource),
	)
	indexer := NewIndexer()
	var sink testSink
	if err := indexer.Analyze(makeRequest(unit), unit, &sink); err != nil {
		t.Fatal(err)
	}
	sink.postprocess()

	const refKind = "REFERENCE"
	const callKind = "CALL"

	revMap := map[string]string{
		refKind:  "REFERENCED_AT",
		callKind: "CALLED_AT",
	}

	tests := []struct {
		needle, nKind, target, tKind, ref string
	}{
		{"main", "USAGE", "main", "PACKAGE", refKind},
		{"T(25)", "USAGE", "T", "TYPE_ALIAS", refKind},
		{"t + 1", "USAGE", "t", "PARAMETER", refKind},
		{"V // ref", "USAGE", "V", "VARIABLE", refKind},
		{"F(q, true)", "USAGE", "F", "FUNCTION", callKind},
		{"q, true", "USAGE", "q", "LOCAL", refKind},
		{"s.f()", "USAGE", "s", "VARIABLE", refKind},
		{"T.m(0)", "USAGE", "T", "TYPE_ALIAS", refKind},
		{"m(0)", "USAGE", "m", "METHOD", callKind},      // in T.m(0)
		{"m()//V.m", "USAGE", "m", "METHOD", callKind},  // in V.m()
		{"f)", "USAGE", "f", "FIELD", refKind},          // in print(s.f)
		{"f()", "USAGE", "f", "FIELD", refKind},         // in s.f()
		{"f()", "USAGE", "f", "FIELD", refKind},         // in s.f()
		{"f()", "USAGE", "f", "FIELD", refKind},         // in s.f()
		{"foo//goto", "USAGE", "foo", "LABEL", refKind}, // in goto foo
	}

	for _, test := range tests {
		pos := strings.Index(testSource, test.needle)
		if pos < 0 {
			t.Errorf("Broken test: missing needle %q", test.needle)
			continue
		}

		fKind, rKind := test.ref, revMap[test.ref]
		fwd, rev := sink.edgePair(node.withKind(test.nKind).withOffset(pos),
			node.withKind(test.tKind).withIdent(test.target), fKind, rKind)
		if fwd == nil {
			t.Errorf("Missing edge %q @ %d --%s--> %s %q", test.needle, pos, fKind, test.tKind, test.target)
		}
		if rev == nil {
			t.Errorf("Missing edge %q @ %d <--%s-- %s %q", test.needle, pos, rKind, test.tKind, test.target)
		}
	}
}

// Verify that usages in functions have a USAGE_CONTEXT edge to the function.
func TestUsageContext(t *testing.T) {
	const testSource = `
package foo
var V int
func F() { println(V, true) }
func G(i int) {
  j := func() int { return i + V + 0 }()
  println(j, false)
}`
	unit := makeCompilation(testTarget, sourceInput(testPath, testSource))
	indexer := NewIndexer()
	var sink testSink
	if err := indexer.Analyze(makeRequest(unit), unit, &sink); err != nil {
		t.Fatal(err)
	}
	sink.postprocess()

	const fKind = "USAGE_CONTEXT"
	const rKind = "ENCLOSED_USAGE"

	tests := []struct {
		needle, target string
	}{
		{"V, true", "F"},
		{"i + V", ""}, // "" => anon func within G (hacky)
		{"V + 0", ""}, // "" => anon func within G (hacky)
		{"j, false", "G"},
		{"false", "G"},
	}
	for _, test := range tests {
		pos := strings.Index(testSource, test.needle)
		if pos < 0 {
			t.Errorf("Broken test: missing needle %q", test.needle)
			continue
		}
		fwd, rev := sink.edgePair(node.withKind("USAGE").withOffset(pos),
			node.withKind("FUNCTION").withIdent(test.target), fKind, rKind)
		if fwd == nil {
			t.Errorf("Missing edge %q @ %d --%s--> %q", test.needle, pos, fKind, test.target)
		}
		if rev == nil {
			t.Errorf("Missing edge %q @ %d <--%s-- %q", test.needle, pos, rKind, test.target)
		}
	}
}

// Verify that package nodes and figments are created and linked correctly.
func TestPackageStructure(t *testing.T) {
	// Set up a compilation with a fake library package, so we can ensure that
	// library packages get nodes as well.
	unit := makeCompilation(testTarget, sourceInput(testPath, "package foo\n"),
		fakeImport("lib/x.a", "x"), fakeImport("nonlib/y.gcgox", "y"))
	unit.Proto.Argument = []string{"--goroot", "lib"}

	indexer := NewIndexer()
	var sink testSink
	if err := indexer.Analyze(makeRequest(unit), unit, &sink); err != nil {
		t.Fatal(err)
	}
	sink.postprocess()

	const fKind = "PACKAGE_CONTAINS"
	const rKind = "BELONGS_TO_PACKAGE"

	fwd, rev := sink.edgePair(node.withKind("PACKAGE"), node.withKind("FILE"), fKind, rKind)
	if fwd == nil {
		t.Errorf("Missing edge %q --%s--> %q", testPath, fKind, "package foo")
	}
	if rev == nil {
		t.Errorf("Missing edge %q <--%s-- %q", testPath, rKind, "package foo")
	}

	if pkg := sink.someNode(node.withKind("PACKAGE").withIdent("y")); pkg != nil {
		t.Errorf("Unexpected package %+v", pkg)
	}

	// There should be a figment for the package.
	pkg, file := fwd.StartNode, fwd.EndNode
	figment := sink.uniqueNode(node.withKind("FILE").withName("google3/dummy/test/target"))
	if figment == nil {
		t.Fatalf("Missing figment node for %s", testTarget)
	}

	// The package's location should be "in" the figment.  The line number
	// should match what is in the template.
	if got, want := pkg.GetLocation().GetUri(), figment.GetTicket(); got != want {
		t.Errorf("Package location URI: got %q, want %q", got, want)
	}
	if got, want := pkg.Location.Span.Start.GetLineno(), int32(5); got != want {
		t.Errorf("Package location line: got %d, want %d", got, want)
	}

	// The figment should reference the source.  The line number should match
	// what is in the template.
	fwd, rev = sink.edgePair(node.withKind("USAGE").withURI(figment.Location.GetUri()).withLine(6),
		node.isExactly(file), "REFERENCE", "REFERENCED_AT")
	if fwd == nil {
		t.Errorf("Missing edge %q --REFERENCE--> %q", figment.GetTicket(), testPath)
	}
	if rev == nil {
		t.Errorf("Missing edge %q <--REFERENCED_AT-- %q", figment.GetTicket(), testPath)
	}
}

// This regression test ensures the indexer doesn't fail or crash on
// some (mostly ill-formed) problematic inputs.
func TestDoesntCrash(t *testing.T) {
	for i, input := range []string{
		// A function redefinition has no types.Func object.
		// The inner FuncLit may attempt to create edges to the outer.
		`package foo; func f() {}; func f() { go func() {}() }`,
		// Unresolved import.
		`package foo; import "bar"; var _ bar.T`,
	} {
		unit := makeCompilation(testTarget, sourceInput(testPath, input))
		if err := NewIndexer().Analyze(makeRequest(unit), unit, new(testSink)); err != nil {
			t.Errorf("#%d: Analyze(%s) failed: %s", i, input, err)
		}
	}
}

// Verify that edges are added for:
// - T INHERITS M for each method M of each type T
// - T IMPLEMENTS U for all pairs of types
//   s.t. T (or *T) is assignable to U
// - TM IMPLEMENTS UM for each concrete method TM
//   and abstract method UM s.t. T IMPLEMENTS U.
func TestMethodSetsAndImplements(t *testing.T) {
	unit := makeCompilation(testTarget, sourceInput(testPath, `
package foo

type I interface { f() }
type J interface { f(); g(); }

type T int
func (T) f() {}
func (*T) g() {}

type S struct { *T }

func f() {
	type U struct { T }
}

`))

	indexer := NewIndexer()
	var sink testSink
	if err := indexer.Analyze(makeRequest(unit), unit, &sink); err != nil {
		t.Fatal(err)
	}
	sink.postprocess()

	tests := []struct {
		kind, start, end string
	}{
		{"IMPLEMENTS", "target.J", "target.I"},
		{"IMPLEMENTS", "target.S", "target.I"},
		{"IMPLEMENTS", "target.S", "target.J"},
		{"IMPLEMENTS", "target.T", "target.I"},
		{"IMPLEMENTS", "target.T", "target.J"},
		{"INHERITS", "target.I", "target.I.f"},
		{"INHERITS", "target.J", "target.J.f"},
		{"INHERITS", "target.J", "target.J.g"},
		{"INHERITS", "target.S", "target.*T.g"},
		{"INHERITS", "target.S", "target.T.f"},
		{"INHERITS", "target.T", "target.*T.g"},
		{"INHERITS", "target.T", "target.T.f"},
		{"IMPLEMENTS", "target.*T.g", "target.J.g"},
		{"IMPLEMENTS", "target.*T.g", "target.J.g"},
		{"IMPLEMENTS", "target.T.f", "target.I.f"},
		{"IMPLEMENTS", "target.T.f", "target.I.f"}, // start should really be S.f
		{"IMPLEMENTS", "target.T.f", "target.J.f"},
		{"IMPLEMENTS", "target.T.f", "target.J.f"}, // ditto
		{"IMPLEMENTS", "target.(local).U", "target.I"},
		{"IMPLEMENTS", "target.(local).U", "target.J"},
	}
	abbrev := func(ticket string) string {
		return strings.Replace(ticket, "google3/dummy/test/target", "target", -1)
	}

	if true {
		for _, e := range sink.edges {
			start := abbrev(e.StartNode.GetDisplayName())
			end := abbrev(e.EndNode.GetDisplayName())
			log.Printf("EDGE %s %s %s", e.GetKind(), start, end)
		}
	}

next:
	for _, test := range tests {
		for _, e := range sink.edges {
			if e.GetKind().String() == test.kind {
				kind := e.GetKind().String()
				start := abbrev(e.StartNode.GetDisplayName())
				end := abbrev(e.EndNode.GetDisplayName())
				if test.kind == kind &&
					test.start == start &&
					test.end == end {
					continue next
				}
			}
		}
		t.Errorf("Missing edge %q --%s--> %q", test.start, test.kind, test.end)
	}
}
