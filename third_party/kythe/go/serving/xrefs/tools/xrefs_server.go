// Binary xrefs_server is an HTTP frontend to an xrefs.Service based on a given
// LevelDB --graphstore.  The given graphstore will be first pre-processed to
// add reverse edges and store a static in-memory directory structure. The
// server will then expose the following methods:
//
//   GET /corpusRoots
//     Returns a JSON map from corpus to []root names (map[string][]string)
//   GET /dir/<path>?corpus=<corpus>&root=<root>[&recursive]
//     Returns a JSON object describing the given directory's contents
//     (sub-directories and files).  If the "recursive" arg is given, each
//     sub-directory will be recursively populated with its contents instead of
//     just providing the directory's name.  See shallowDirectory and
//     filetree.Directory for the JSON object's structure.
//   GET /file/<path>?corpus=<corpus>&root=<root>&language=<lang>&signature=<sig>
//     Returns the JSON equivalent of an xrefs.DecorationsReply for the
//     described file.  References and source-text will be supplied in the
//     reply.
//   GET /xrefs?ticket=<ticket>
//     Returns a JSON map from edgeKind to a set of anchor/file locations that
//     attach to the given node.
//   GET /<path>
//     Serves the file at the given local path, relative to --serving_path
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"third_party/kythe/go/serving/xrefs"
	"third_party/kythe/go/storage"
	"third_party/kythe/go/storage/filetree"
	"third_party/kythe/go/storage/leveldb"
	"third_party/kythe/go/util/httpencoding"
	"third_party/kythe/go/util/kytheuri"
	"third_party/kythe/go/util/schema"
	"third_party/kythe/go/util/stringset"

	spb "third_party/kythe/proto/storage_proto"
	xpb "third_party/kythe/proto/xref_proto"

	"code.google.com/p/goprotobuf/proto"
)

var (
	port        = flag.Int("port", 8080, "HTTP serving port")
	levelDBPath = flag.String("graphstore", "", "Path to LevelDB GraphStore")

	servingDir = flag.String("serving_path", "third_party/kythe/web/ui/resources/public", "Path to public serving directory")
)

var (
	tree *filetree.FileTree
	xs   xrefs.Service
)

func main() {
	flag.Parse()

	gs, err := leveldb.Open(*levelDBPath)
	if err != nil {
		log.Fatalf("Failed to load GraphStore: %v", err)
	}

	// Pre-process GraphStore
	if err := addReverseEdges(gs); err != nil {
		log.Fatalf("Failed to add reverse edges: %v", err)
	}
	tree, err = createFileTree(gs)
	if err != nil {
		log.Fatalf("Failed to create GraphStore file tree: %v", err)
	}
	xs = xrefs.NewGraphStoreService(gs)

	// Add HTTP handlers
	http.HandleFunc("/corpusRoots", corpusRootsHandler)
	http.HandleFunc("/dir/", dirHandler)
	http.HandleFunc("/file/", fileHandler)
	http.HandleFunc("/xrefs", xrefsHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(*servingDir, filepath.Clean(r.URL.Path)))
	})

	log.Printf("xrefs browser launching on port %d", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}

// corpusRootsHandler replies with a JSON map of corpus roots
func corpusRootsHandler(w http.ResponseWriter, r *http.Request) {
	corpusRoots := make(map[string][]string)
	for corpus, rootDirs := range tree.CorporaRoots {
		var roots []string
		for root := range rootDirs {
			roots = append(roots, root)
		}
		corpusRoots[corpus] = roots
	}
	writeJSON(w, r, corpusRoots)
}

// fileHandler parses a file VName from the Request URL's Path/Query and replies
// with a JSON object equivalent to a xpb.DecorationsReply with its SourceText
// and Reference fields populated
func fileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Only GET requests allowed", http.StatusMethodNotAllowed)
		return
	}
	path := trimPath(r.URL.Path, "/file/")
	if path == "/file" {
		http.Error(w, "Bad Request: no file path given", http.StatusBadRequest)
		return
	}

	args := r.URL.Query()
	fileVName := &spb.VName{
		Signature: firstOrNil(args["signature"]),
		Language:  firstOrNil(args["language"]),
		Corpus:    firstOrNil(args["corpus"]),
		Root:      firstOrNil(args["root"]),
		Path:      proto.String(path),
	}

	startTime := time.Now()
	reply, err := xs.Decorations(&xpb.DecorationsRequest{
		Location:   &xpb.Location{Ticket: proto.String(kytheuri.FromVName(fileVName).String())},
		SourceText: proto.Bool(true),
		References: proto.Bool(true),
	})
	if err != nil {
		code := http.StatusInternalServerError
		if strings.Contains(err.Error(), "file not found") {
			code = http.StatusNotFound
		}
		http.Error(w, err.Error(), code)
		return
	}
	log.Printf("Decorations [%v]", time.Since(startTime))

	if err := writeJSON(w, r, reply); err != nil {
		log.Printf("Error encoding file response: %v", err)
	}
}

// dirHandler parses a corpus/root/path from the Request URL's Path/Query and
// replies with a JSON object describing the directories sub-directories and
// files
func dirHandler(w http.ResponseWriter, r *http.Request) {
	path := trimPath(r.URL.Path, "/dir")
	if path == "" {
		path = "/"
	}
	args := r.URL.Query()
	corpus, root := firstOrEmpty(args["corpus"]), firstOrEmpty(args["root"])

	dirTree := tree.LookupDir(corpus, root, path)

	// The response object to send as JSON. Could be either a
	// filetree.Directory or a shallowDirectory
	var resp interface{} = dirTree

	// If the "recursive" argument wasn't given, then prune the nested
	// directories and only return their names in a shallowDirectory
	if dirTree != nil && firstOrNil(args["recursive"]) == nil {
		dir := &shallowDirectory{
			Name:  dirTree.Name,
			Files: dirTree.Files,
		}
		resp = dir
		for subDir := range dirTree.Dirs {
			dir.Dirs = append(dir.Dirs, subDir)
		}
	}

	if err := writeJSON(w, r, resp); err != nil {
		log.Printf("Failed to encode directory: %v", err)
	}
}

type shallowDirectory struct {
	Name  string                  `json:"name"`
	Dirs  []string                `json:"dirs"`
	Files map[string][]*spb.VName `json:"files"`
}

var (
	anchorFilters  = []string{schema.NodeKindFact, "/kythe/loc/*"}
	revDefinesEdge = schema.MirrorEdge(schema.DefinesEdge)
)

// xrefsHandler returns a map from edge kind to set of anchorLocations for a
// given ticket.
func xrefsHandler(w http.ResponseWriter, r *http.Request) {
	// Graph path:
	//  node[ticket]
	//    ( --%revEdge-> || --forwardEdge-> relatedNode --%defines-> )
	//    []anchor --rev(childof)-> file

	startTime := time.Now()
	ticket := firstOrEmpty(r.URL.Query()["ticket"])
	if ticket == "" {
		http.Error(w, "Bad Request: missing target parameter", http.StatusBadRequest)
		return
	}

	// node[ticket] --*-> []anchor
	anchorEdges, anchorNodes, err := edgesMaps(xs.Edges(&xpb.EdgesRequest{
		Ticket: []string{ticket},
		Filter: anchorFilters,
	}))
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad anchor edges request: %v", err), http.StatusInternalServerError)
		return
	} else if len(anchorEdges) == 0 {
		http.Error(w, fmt.Sprintf("No reference edges found for %q", ticket), http.StatusNotFound)
		return
	}

	// Preliminary response map w/o File tickets populated
	anchorLocs := make(map[string][]*anchorLocation)

	anchorTargetSet := stringset.New()
	relatedNodeSet := stringset.New()
	relatedNodeEdgeKinds := make(map[string][]string) // ticket -> []edgeKind
	for kind, targets := range anchorEdges[ticket] {
		for _, target := range targets {
			if schema.EdgeDirection(kind) == schema.Reverse {
				// --%revEdge-> anchor
				loc := nodeAnchorLocation(anchorNodes[target])
				if loc == nil {
					continue
				}
				anchorTargetSet.Add(target)
				anchorLocs[kind] = append(anchorLocs[kind], loc)
			} else {
				// --forwardEdge-> relatedNode
				relatedNodeSet.Add(target)
				relatedNodeEdgeKinds[target] = append(relatedNodeEdgeKinds[target], kind)
			}
		}
	}

	if len(relatedNodeSet) > 0 {
		// relatedNode --%defines-> anchor
		relatedAnchorEdges, relatedAnchorNodes, err := edgesMaps(xs.Edges(&xpb.EdgesRequest{
			Ticket: relatedNodeSet.Slice(),
			Kind:   []string{revDefinesEdge},
			Filter: anchorFilters,
		}))
		if err != nil {
			http.Error(w, fmt.Sprintf("Bad inter anchor edges request: %v", err), http.StatusInternalServerError)
			return
		}

		for interNode, edgeKinds := range relatedNodeEdgeKinds {
			for _, target := range relatedAnchorEdges[interNode][revDefinesEdge] {
				node := relatedAnchorNodes[target]
				if nodeKind(node) == schema.AnchorKind {
					loc := nodeAnchorLocation(node)
					if loc == nil {
						continue
					}
					anchorTargetSet.Add(target)
					for _, kind := range edgeKinds {
						anchorLocs[kind] = append(anchorLocs[kind], loc)
					}
				}
			}
		}
	}

	// []anchor -> file
	fileEdges, fileNodes, err := edgesMaps(xs.Edges(&xpb.EdgesRequest{
		Ticket: anchorTargetSet.Slice(),
		Kind:   []string{schema.ChildOfEdge},
		Filter: []string{schema.NodeKindFact},
	}))
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad files edges request: %v", err), http.StatusInternalServerError)
		return
	}

	// Response map to send as JSON (filtered from anchorLocs for only anchors w/ known files)
	refs := make(map[string][]*anchorLocation)

	// Find files for each of anchorLocs and filter those without known files
	var totalRefs int
	for kind, locs := range anchorLocs {
		var fileLocs []*anchorLocation
		for _, loc := range locs {
			file := stringset.New()
			for _, targets := range fileEdges[loc.Anchor] {
				for _, target := range targets {
					if nodeKind(fileNodes[target]) == schema.FileKind {
						file.Add(target)
					}
				}
			}
			if len(file) != 1 {
				log.Printf("Not one file found for anchor %q: %v", loc.Anchor, file.Slice())
				continue
			}
			ticket := file.Slice()[0]
			vname, err := kytheuri.ToVName(ticket)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to parse file VName %q: %v", ticket, err), http.StatusInternalServerError)
				return
			}
			loc.File = vname
			fileLocs = append(fileLocs, loc)
		}
		if len(fileLocs) != 0 {
			totalRefs += len(fileLocs)
			refs[kind] = fileLocs
		}
	}

	log.Printf("XRefs [%v]\t%d", time.Since(startTime), totalRefs)
	if err := writeJSON(w, r, refs); err != nil {
		log.Println(err)
	}
}

type anchorLocation struct {
	Anchor string     `json:"anchor"`
	File   *spb.VName `json:"file"`
	Start  int        `json:"start"`
	End    int        `json:"end"`
}

func addReverseEdges(gs storage.GraphStore) error {
	log.Println("Adding reverse edges")
	entries := make(chan *spb.Entry)
	var (
		wg           sync.WaitGroup
		totalEntries int
		addedEdges   int
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for entry := range entries {
			kind := entry.GetEdgeKind()
			if kind != "" && schema.EdgeDirection(kind) == schema.Forward {
				if err := gs.Write(&spb.WriteRequest{
					Source: entry.Target,
					Update: []*spb.WriteRequest_Update{{
						Target:    entry.Source,
						EdgeKind:  proto.String(schema.MirrorEdge(kind)),
						FactName:  entry.FactName,
						FactValue: entry.FactValue,
					}},
				}); err != nil {
					log.Fatalf("Failed to write reverse edge: %v", err)
				}
				addedEdges++
			}
			totalEntries++
		}
	}()
	startTime := time.Now()
	err := gs.Scan(&spb.ScanRequest{}, entries)
	close(entries)
	if err != nil {
		return err
	}
	wg.Wait()
	log.Printf("Wrote %d reverse edges in store of %d entries: %v", addedEdges, totalEntries, time.Since(startTime))
	return nil
}

func createFileTree(gs storage.GraphStore) (*filetree.FileTree, error) {
	log.Println("Creating GraphStore file tree")
	startTime := time.Now()
	defer func() {
		log.Printf("Tree populated in %v", time.Since(startTime))
	}()
	t := filetree.NewTree()
	return t, t.Populate(gs)
}

func writeJSON(w http.ResponseWriter, r *http.Request, v interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	cw := httpencoding.CompressData(w, r)
	defer cw.Close()
	return json.NewEncoder(cw).Encode(v)
}

func firstOrEmpty(strs []string) string {
	if str := firstOrNil(strs); str != nil {
		return *str
	}
	return ""
}

func firstOrNil(strs []string) *string {
	if len(strs) == 0 {
		return nil
	}
	return &strs[0]
}

func trimPath(path, prefix string) string {
	return strings.TrimPrefix(filepath.Clean(path), prefix)
}

// edgesMaps post-processes an EdgesReply into a ticket->edgeKind->[]targets map
// and a nodes map.
func edgesMaps(r *xpb.EdgesReply, err error) (map[string]map[string][]string, map[string]*xpb.NodeInfo, error) {
	if err != nil {
		return nil, nil, err
	}

	edges := make(map[string]map[string][]string)
	for _, s := range r.EdgeSet {
		g := make(map[string][]string)
		for _, group := range s.Group {
			g[group.GetKind()] = group.TargetTicket
		}
		edges[s.GetSourceTicket()] = g
	}
	nodes := make(map[string]*xpb.NodeInfo)
	for _, n := range r.Node {
		nodes[n.GetTicket()] = n
	}
	return edges, nodes, nil
}

// nodeKind returns the schema.NodeKindFact value of the given node, or if not
// found, ""
func nodeKind(n *xpb.NodeInfo) string {
	for _, f := range n.GetFact() {
		if f.GetName() == schema.NodeKindFact {
			return string(f.Value)
		}
	}
	return ""
}

// nodeAnchorLocation returns an equivalent anchorLocation for the given node.
// Returns nil if the given node isn't a valid anchor
func nodeAnchorLocation(anchor *xpb.NodeInfo) *anchorLocation {
	if nodeKind(anchor) != schema.AnchorKind {
		return nil
	}
	var start, end int
	for _, f := range anchor.Fact {
		var err error
		switch f.GetName() {
		case schema.AnchorStartFact:
			start, err = strconv.Atoi(string(f.Value))
		case schema.AnchorEndFact:
			end, err = strconv.Atoi(string(f.Value))
		}
		if err != nil {
			log.Printf("Failed to parse %q: %v", string(f.Value), err)
		}
	}
	return &anchorLocation{
		Anchor: anchor.GetTicket(),
		Start:  start,
		End:    end,
	}
}
