package service

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"shipshape/util/file"
	strset "shipshape/util/strings"
	"third_party/kythe/go/rpc/client"
	"third_party/kythe/go/rpc/server"

	"code.google.com/p/goprotobuf/proto"

	notepb "shipshape/proto/note_proto"
	contextpb "shipshape/proto/shipshape_context_proto"
	rpcpb "shipshape/proto/shipshape_rpc_proto"
)

const (
	// How long to wait for an analyzer service to become healthy.
	analyzerHealthTimeout = 30 * time.Second
	configFilename        = ".shipshape"
)

var (
	clients = make(map[string]*client.Client)
)

type ShipshapeDriver struct {
	AnalyzerLocations []string
	// catMap is a mapping from analyzer locations to the categories they have available.
	// The range of catMap is the same as AnalyzerLocations
	catMap map[string]strset.Set
}

type dispatcherCategories struct {
	dispatcher string
	categories strset.Set
}

// NewDriver creates a new driver with with the analyzers at the
// specified locations. This func makes no rpcs.
func NewDriver(analyzerLocations []string) *ShipshapeDriver {
	var addrs []string

	for _, addr := range analyzerLocations {
		addrs = append(addrs, strings.TrimPrefix(addr, "http://"))
	}
	return &ShipshapeDriver{AnalyzerLocations: addrs}
}

// NewTestDriver is only for testing. It creates a ShipshapeDriver
// with the address to categories map preset.
func NewTestDriver(addrToCats map[string]strset.Set) *ShipshapeDriver {
	var addrs []string
	var catMap = make(map[string]strset.Set)
	for addr, cats := range addrToCats {
		trimmed := strings.TrimPrefix(addr, "http://")
		addrs = append(addrs, trimmed)
		catMap[trimmed] = cats
	}
	return &ShipshapeDriver{AnalyzerLocations: addrs, catMap: catMap}
}

// Run runs the analyzers that this driver knows about on the provided ShipshapeRequest,
// taking configuration into account.
func (td ShipshapeDriver) Run(ctx server.Context, in *rpcpb.ShipshapeRequest, out chan<- *rpcpb.ShipshapeResponse) error {
	var ars []*rpcpb.AnalyzeResponse
	log.Printf("Received analysis request for event %v, categories %v, repo %v", *in.Event, in.TriggeredCategory, *in.ShipshapeContext.RepoRoot)
	// However we exit, send back the set of collected AnalyzeResponses
	defer func() {
		out <- &rpcpb.ShipshapeResponse{
			AnalyzeResponse: ars,
		}
	}()

	if in.ShipshapeContext.RepoRoot == nil {
		return fmt.Errorf("No repo root was set")
	}
	root := *in.ShipshapeContext.RepoRoot

	// cd into the root directory
	orgDir, restore, err := file.ChangeDir(root)
	if err != nil {
		log.Printf("Could not change into directory %s from base %s", root, orgDir)
		ars = append(ars, generateFailure("Driver setup", fmt.Sprint(err)))
		return err
	}
	defer func() {
		if err := restore(); err != nil {
			log.Printf("could not return back into %s from %s: %v", orgDir, root, err)
		}
	}()

	// load the config, if it exists
	cfg, err := loadConfig(configFilename, *in.Event)
	if err != nil {
		log.Print("error loading config")
		// TODO(collinwinter): attach the error to the config file.
		ars = append(ars, generateFailure("Driver setup", err.Error()))
		return err
	}

	// Use the triggered categories if specified
	var desiredCats strset.Set
	if len(in.TriggeredCategory) > 0 {
		desiredCats = strset.New(in.TriggeredCategory...)
	} else if cfg != nil {
		desiredCats = strset.New(cfg.categories...)
	} else {
		return fmt.Errorf("service needs to be called with triggered categories and/or a repo root with a valid %s file with the event %s", configFilename, *in.Event)
	}

	// Find out what categories we have available, and remove/warn on the missing ones
	td.catMap = td.getAllCategories()
	allCats := td.allCats()
	missingCats := strset.New().AddSet(desiredCats).RemoveSet(allCats)
	for missing := range missingCats {
		ars = append(ars, generateFailure(missing, fmt.Sprintf("The triggered category %q could not be found at the locations %v", missing, td.AnalyzerLocations)))
	}
	desiredCats = desiredCats.RemoveSet(missingCats)

	if len(desiredCats) == 0 {
		log.Printf("No categories configured to run, doing nothing")
		return nil
	}

	ignorePaths := []string{}
	if cfg != nil {
		ignorePaths = cfg.ignore
	}
	// Fill in the file_paths if they are empty in the context
	context := proto.Clone(in.ShipshapeContext).(*contextpb.ShipshapeContext)
	context.FilePath, err = retrieveAndFilterFiles(*context.RepoRoot, context.FilePath, ignorePaths)
	if err != nil {
		log.Print("Had problems accessing files: %v", err.Error())
		ars = append(ars, generateFailure("Driver setup", fmt.Sprint(err)))
		return err
	}
	if len(context.FilePath) == 0 {
		log.Printf("No files to run on, doing nothing")
		return nil
	}

	log.Print("Analyzing")
	ars = append(ars, td.callAllAnalyzers(desiredCats, context)...)
	log.Print("Analysis completed")
	return nil
}

// WaitForAnalyzers witll wait for all the given analyzers to become healthy
// That is, their service is up and ready to serve requests.
// Returns a mapping of which analyzers had which errors.
func WaitForAnalyzers(analyzerList []string) map[string]error {
	var wg sync.WaitGroup
	var health = make(map[string]error)

	for _, analyzerAddr := range analyzerList {
		wg.Add(1)
		go func(addr string) {
			httpClient := getHTTPClient(addr)
			err := httpClient.WaitUntilReady(analyzerHealthTimeout)
			health[addr] = err
			wg.Done()
		}(analyzerAddr)
	}
	wg.Wait()
	return health
}

// retrieveAndFilter files returns a list of files (initiated with files if that is non-empty,
// or from recursing on root if it is) and removes the ones in the ignore list.
func retrieveAndFilterFiles(root string, files []string, ignore []string) ([]string, error) {
	if len(files) == 0 {
		log.Printf("No files, getting some")
		var err error
		files, err = collectAllFiles(root)
		if err != nil {
			return nil, err
		}
	}

	return filterPaths(ignore, files), nil
}

// collectAllFiles returns a list of all files for the passed-in root
func collectAllFiles(root string) ([]string, error) {
	var paths []string
	walkpath := func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return nil
		}
		// Skip directories starting with "."
		dot := strings.HasPrefix(f.Name(), ".")
		if f.IsDir() && dot {
			return filepath.SkipDir
		}
		if !f.IsDir() && !dot {
			relpath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			paths = append(paths, relpath)
		}
		return nil
	}
	if err := filepath.Walk(root, walkpath); err != nil {
		return nil, err
	}
	return paths, nil
}

// filterPaths drops paths that fall under one of the given directories to ignore.
// All directory names are assumed to end with /.
func filterPaths(ignoreDirs []string, filePaths []string) []string {
	var keepPaths []string
nextFile:
	for _, file := range filePaths {
		for _, dir := range ignoreDirs {
			if strings.HasPrefix(file, dir) {
				continue nextFile
			}
		}
		keepPaths = append(keepPaths, file)
	}
	return keepPaths
}

// callAllAnalyzers loops through the analyzer services, determines whether analyze should be called
// on each, and then calls it with the appropriate set of files and categories.
// It takes the configuration and the original context, and returns a slice of AnalyzeResponses.
func (td ShipshapeDriver) callAllAnalyzers(desiredCats strset.Set, context *contextpb.ShipshapeContext) []*rpcpb.AnalyzeResponse {
	var ars []*rpcpb.AnalyzeResponse
	var chans []chan *rpcpb.AnalyzeResponse
	for analyzer, providedCats := range td.catMap {
		cats := providedCats.Intersect(desiredCats)

		log.Printf("Analyzer %s filtered to categories %v and files %v", analyzer, cats, context.FilePath)

		// If there are any categories to run on for this analyzer service,
		// go ahead and call analyze
		if len(cats) > 0 {
			c := make(chan *rpcpb.AnalyzeResponse)
			chans = append(chans, c)
			req := &rpcpb.AnalyzeRequest{
				ShipshapeContext: context,
				Category:         cats.ToSlice(),
			}
			go callAnalyze(analyzer, req, c)
		}
	}

	// Collect up all the responses where we actually called analyze
	for _, c := range chans {
		ar := <-c
		ars = append(ars, filterResults(context, ar))
	}
	return ars
}

// filterResults removes any notes where the category is nil, the category is not specified for
// the file path by the configuration, or there is no location with a source context.
// The config category and internal failure category cannot be turned off.
func filterResults(context *contextpb.ShipshapeContext, response *rpcpb.AnalyzeResponse) *rpcpb.AnalyzeResponse {
	files := strset.New(context.FilePath...)
	var keep []*notepb.Note
	for _, note := range response.Note {
		if note.Category != nil {
			if note.Location != nil && note.Location.SourceContext != nil && (note.Location.Path == nil || files.Contains(*note.Location.Path)) {
				keep = append(keep, note)
			}
		}
	}

	return &rpcpb.AnalyzeResponse{
		Note:    keep,
		Failure: response.Failure,
	}
}

// allCats returns the entire set of categories for the driver, across all analyzers
func (td ShipshapeDriver) allCats() strset.Set {
	var catSet = strset.New()
	for _, cats := range td.catMap {
		catSet.AddSet(cats)
	}
	return catSet
}

// getAllCategories loops through the analyzers and gets the categories for each of them.
// It returns a mapping from the analyzer address to the set of categories it provides.
func (td ShipshapeDriver) getAllCategories() map[string]strset.Set {
	categories := make(map[string]strset.Set)
	var catChans []chan dispatcherCategories
	for _, analyzer := range td.AnalyzerLocations {
		c := make(chan dispatcherCategories)
		catChans = append(catChans, c)
		go callGetCategories(analyzer, c)
	}
	for _, c := range catChans {
		dispatcherCats := <-c
		categories[dispatcherCats.dispatcher] = dispatcherCats.categories
	}
	return categories
}

// callGetCategories requests the categories for the specified analyzer and puts them onto the
// channel provided. If anything goes wrong, it returns the empty set.
func callGetCategories(analyzer string, out chan<- dispatcherCategories) {
	httpClient := getHTTPClient(analyzer)
	var resp rpcpb.GetCategoryResponse
	var cats strset.Set
	err := httpClient.Call("/AnalyzerService/GetCategory", &rpcpb.GetCategoryRequest{}, &resp)
	if err != nil {
		log.Printf("Could not get categories from %s: %v", analyzer, err)
		cats = strset.New()
	} else {
		cats = strset.New(resp.Category...)
	}
	out <- dispatcherCategories{
		dispatcher: analyzer,
		categories: cats,
	}
}

// callAnalyze attempts to call analyze for the specified analyzer with the given request.
// If anything goes wrong, it puts an AnalysisFailure into the AnalyzeResponse.
func callAnalyze(analyzer string, req *rpcpb.AnalyzeRequest, out chan<- *rpcpb.AnalyzeResponse) {
	httpClient := getHTTPClient(analyzer)
	var resp rpcpb.AnalyzeResponse
	err := httpClient.Call("/AnalyzerService/Analyze", req, &resp)
	if err != nil {
		out <- &rpcpb.AnalyzeResponse{
			Failure: []*rpcpb.AnalysisFailure{
				&rpcpb.AnalysisFailure{
					FailureMessage: proto.String(fmt.Sprintf("Error from analyzer %s: %v", analyzer, err)),
				},
			},
		}
	} else {
		out <- &resp
	}
}

// generateFailure creates a response with an analysis failure containing the given
// category and message
func generateFailure(cat string, message string) *rpcpb.AnalyzeResponse {
	return &rpcpb.AnalyzeResponse{
		Failure: []*rpcpb.AnalysisFailure{
			&rpcpb.AnalysisFailure{
				Category:       proto.String(cat),
				FailureMessage: proto.String(message),
			},
		},
	}
}

// getHTTPClient provides a (cached) HTTPClient for the address specified.
func getHTTPClient(addr string) *client.Client {
	httpClient, exists := clients[addr]
	if !exists {
		clients[addr] = client.NewHTTPClient(addr)
		httpClient = clients[addr]
	}
	return httpClient
}
