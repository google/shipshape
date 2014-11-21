// Package grok implements a platform analyzer that generates Grok index
// artifacts for a Go compilation.
//
// Example usage:
//   const corpus = "google3"
//   indexer := grok.NewIndexer(corpus)
//   if err := indexer.Analyze(compilationUnit); err != nil
//     log.Fatalf("Analysis failed: %s", err)
//   }
//
// A grok.Indexer implements the analysis.Analyzer interface, permitting it to
// be passed to an analysis.Driver.  A single Indexer can be used to index an
// arbitrary number of distinct compilation units.
package grok

import (
	"errors"
	"go/scanner"
	"go/token"
	"log"

	"third_party/kythe/go/analyzers/go/index"
	"third_party/kythe/go/platform/analysis"

	"code.google.com/p/goprotobuf/proto"

	apb "third_party/kythe/proto/analysis_proto"
	enumpb "third_party/kythe/proto/enums_proto"
	kythepb "third_party/kythe/proto/kythe_proto"
)

// Indexer implements analysis.Analyzer to emit Grok index artifacts for Go
// compilations.
type Indexer struct {
	// A set of tickets of nodes that have been emitted by this indexer.  This
	// set is not necessarily complete -- it is used to avoid emitting multiple
	// copies of the nodes for predeclared identifiers, so only a selection of
	// tickets will be present.
	emitted map[string]bool
	// EmitEntries flags the conversion from IndexArtifact to Entry protobufs
	EmitEntries bool
}

// NewIndexer creates a new Indexer for the specified Go corpus.
func NewIndexer() Indexer {
	return Indexer{
		emitted: make(map[string]bool),
	}
}

// Analyze implements the analysis.Analyzer interface.
func (idx Indexer) Analyze(req *apb.AnalysisRequest, fetcher analysis.Fetcher, sink analysis.Sink) error {
	unit := req.GetCompilation()
	if unit.GoArguments == nil {
		log.Printf("Requested compilation is not a Go compilation: %+v", unit)
		return errors.New("not a Go compilation")
	}

	index := index.New(unit, fetcher)
	ctx := newContext(idx, index, sink, idx.EmitEntries)

	// If there's a corpus label in the request, use that in preference to the
	// indexer's preconfigured one, for this request.
	if c := unit.GetVName().GetCorpus(); c != "" {
		ctx.Corpus = c
	}

	signature := unit.GetVName().GetSignature()
	log.Printf("Begin analysis of %q", signature)
	err := index.Resolve()
	if err == nil {
		ctx.typeInfo()
		ctx.populateFieldOwners()
		ctx.walkAll(&indexWalker{context: ctx})
		ctx.walkAll(commentWalker{context: ctx})

		// Emit a package node and figment for the package, including links to all
		// its source files and its godoc page.  The figment will handle generating
		// the godoc URL for the package in this case.
		ctx.ensurePackageNode(ctx.Package, "")
	} else {
		// If resolution fails, emit file nodes so that the diagnostics will
		// have some thing to attach to, and the the index will have data about
		// what went wrong.
		log.Printf("Resolution for %q failed: %s", signature, err)
		w := indexWalker{context: ctx}
		for path := range ctx.Text {
			node := w.makeNode(w.makePathTicket(path), enumpb.NodeEnum_FILE, nil)
			w.fillFileNode(node, path)
			w.writeNode(node)
			log.Printf("Wrote stub node for %q", node.GetTicket())
		}
	}

	// Emit any relevant diagnostics discovered by the indexer.  If we can't
	// figure out which file to attach a diagnostic to, use the first one in
	// the index as a default.

	var defaultKey string
	ctx.Fset.Iterate(func(f *token.File) bool {
		defaultKey = ctx.makePathTicket(f.Name())
		return false
	})

	for _, err := range index.Errors {
		diag := &kythepb.Diagnostic{
			Type:    kythepb.Diagnostic_ERROR.Enum(),
			Message: proto.String(err.Error()),
		}
		target := defaultKey
		if t, ok := err.(*scanner.Error); ok {
			target = ctx.makePathTicket(t.Pos.Filename)
			diag.Range = ctx.spanFromPosition(t.Pos)
			diag.Message = proto.String(t.Msg)
		}
		ctx.addDiagnostic(target, diag)
	}

	log.Printf("Analysis complete: %q", signature)
	return err
}
