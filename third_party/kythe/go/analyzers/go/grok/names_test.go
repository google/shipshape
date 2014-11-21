package grok

// Unit tests for ticket, display name, and signature generation.

import (
	"testing"

	kythepb "third_party/kythe/proto/kythe_proto"

	"code.google.com/p/goprotobuf/proto"
)

func TestDisplayNames(t *testing.T) {
	const testPath = "dummy/test/target/foo.go"
	unit := makeCompilation(testTarget, sourceInput(testPath, `
package foo
type I interface {
  IMa(s string) (b bool)
  IMb() (c complex128)
}
type T int
func F() byte {
  v := byte(0)
  return v
}
func (t *T) TM() bool { return true }
var X = func(p float32) {
  go func() (r bool) { return false }()
}
`))

	indexer := NewIndexer()
	var sink testSink
	if err := indexer.Analyze(makeRequest(unit), unit, &sink); err != nil {
		t.Fatal(err)
	}

	N := func(tail string) string {
		return "google3/dummy/test/target." + tail
	}
	tests := []struct {
		ident, kind, dname string
	}{
		{"foo", "PACKAGE", "google3/dummy/test/target"},
		{"I", "INTERFACE", N("I")},
		{"IMa", "METHOD", N("I.IMa")},
		// {"s", "PARAMETER", N("I.IMa.s")}, // got: no node TODO(adonovan): maybe fix?
		{"IMb", "METHOD", N("I.IMb")},
		// {"c", "LOCAL", N("I.IMb.c")}, // got: no node TODO(adonovan): maybe fix?
		{"T", "TYPE_ALIAS", N("T")},
		{"F", "FUNCTION", N("F")},
		{"v", "LOCAL", N("(local).v")},
		{"TM", "METHOD", N("*T.TM")},
		{"t", "PARAMETER", N("(local).t")},
		{"X", "VARIABLE", N("X")},
		{"p", "PARAMETER", N("(local).p")},
		{"r", "LOCAL", N("(local).r")},
	}

	for _, test := range tests {
		node := sink.uniqueNode(node.withIdent(test.ident).withKind(test.kind))
		if node == nil {
			t.Errorf("Missing %s node for ident %q", test.kind, test.ident)
			continue
		}
		if got := node.GetDisplayName(); got != test.dname {
			t.Errorf("Node %q display name: got %q, want %q", node.GetTicket(), got, test.dname)
		}
	}
}

// Verify that distinct entities do not wind up with the same ticket, even if
// they share partial name information.
func TestTickets(t *testing.T) {
	unit := makeCompilation(testTarget,
		// The source code under test.  This code is intentionally obnoxious
		// about reusing names in different contexts, in order to push on the
		// cases where tickets (which depend on names) might collide.
		sourceInput(testPath, `
package foo
import p "lib"

var x, s int

type T int

func (t T) T() {}

func F(t T) {
  p := 0 // Like the package name
  _ = p
  for {
    var t T
    _ = t
  }

  go func(t T) (s T) { return t }(0)
}

var pT p.T

type S struct {
  T
  p T
  s bool
}

func (s S) S() (x T) { return s.p }`),

		// A fake library so that the source can import.
		fakeImport("lib.a", "lib\n\ttype @\"\".T int"),
	)

	indexer := NewIndexer()
	var sink testSink
	if err := indexer.Analyze(makeRequest(unit), unit, &sink); err != nil {
		t.Fatal(err)
	}

	tmap := make(map[string]*kythepb.Node)
	for _, node := range sink.allNodes(anyNode) {
		node.Content = nil // Discard content for comparisons.

		ticket := node.GetTicket()
		t.Logf("NK %-18s %q", node.GetKind().String(), ticket)
		prev := tmap[ticket]
		if prev == nil {
			tmap[ticket] = node
			continue
		}

		if !proto.Equal(node, prev) {
			t.Errorf("Ticket %q:\n\tgot:  %v\nwant: %v", ticket, node, prev)
		}
	}
}

// Verify that package nodes get created for the intrinsic packages "unsafe"
// and "C".
func TestIntrinsicPackages(t *testing.T) {
	unit := makeCompilation(testTarget, sourceInput(testPath, `
package foo
import "unsafe"
import "C"

func init() {
  C.doSomething()
  z := 1
  println(unsafe.Pointer(&z))
}`))

	indexer := NewIndexer()
	var sink testSink
	if err := indexer.Analyze(makeRequest(unit), unit, &sink); err != nil {
		t.Fatal(err)
	}
	sink.postprocess()

	for _, n := range sink.allNodes(node.withKind("PACKAGE")) {
		t.Logf("PACKAGE %+v", n)
	}

	for _, ticket := range []string{"go:google3:unsafe", "go:google3:C"} {
		node := sink.uniqueNode(node.withTicket(ticket).withKind("PACKAGE"))
		if node == nil {
			t.Errorf("Missing PACKAGE for %q", ticket)
		} else {
			t.Logf("Found intrinsic for %q: %+v", ticket, node)
		}
		if !node.GetModifiers().GetIsFigment() {
			t.Errorf("PACKAGE is missing is_figment modifier: %v", node)
		}
	}
}
