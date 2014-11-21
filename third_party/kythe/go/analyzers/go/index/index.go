// Package index implements the interfaces required to create a Grok semantic
// index for a Go compilation using the analyzer platform.
//
// Usage:
//   // Create an index from a compilation.
//   idx := index.New(compilation)
//
//   // Optionally, set library root and build system output directories.
//   // If you don't do this, they'll be inferred from the arguments of the
//   // compilation if possible.
//   idx.GoRoot = "path/where/go/gc/pkg/lives"
//   idx.BuildRoot = "path/where/objects_are/stored"
//
//   // Resolve packages and types.
//   if err := idx.Resolve(); err != nil {
//     log.Error(err)
//   }
//
//   // Handle errors from resolution if desired.
//   for _, err := range idx.Errors {
//     handleError(err)
//   }
package index

// TODO(adonovan): add a check that emitting an edge without both its
// nodes is an error, to catch the case where a node has the wrong ticket.

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/scanner"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"third_party/kythe/go/platform/analysis"

	"code.google.com/p/go.tools/go/gcimporter"
	"code.google.com/p/go.tools/go/loader"
	"code.google.com/p/go.tools/go/types"

	apb "third_party/kythe/proto/analysis_proto"
)

// An Index represents a type-resolved index for a single Go
// compilation unit, i.e. a package.
// After the index is constructed and resolved, the resolved ASTs can be found
// in the Files field, source text in the Text field, and errors in Errors.
type Index struct {
	Compilation *apb.CompilationUnit // The compilation under analysis.
	fetcher     analysis.Fetcher     // Where the required inputs can be found.

	Fset     *token.FileSet    // Position info for source files in compilation unit.
	Text     map[string][]byte // The text of all source files
	Files    []*ast.File       // The AST corresponding to Compilation[i].SourceFile
	Errors   []error           // Errors encountered while resolving.
	TypeInfo types.Info        // Type-checker facts for the ASTs of this package.
	Package  *types.Package    // Type-checker package

	AllPackages map[*types.Package]*loader.PackageInfo // All packages mentioned by unit

	GoRoot    string    // Directory where Go library packages are stored.
	osArch    string    // "pkg/os_arch" path segments; optional.
	BuildRoot []*string // Directory for build system outputs.
	PkgRoot   string    // The path of the package whose index this is.
	Corpus    string    // The corpus name

	pkgPath    map[string]string // Map from package name to file path.
	pathDigest map[string]string // Map from path to file digest.
}

// New returns a new Index for a compilation, using the specified fetcher to
// retrieve the necessary input files.
func New(unit *apb.CompilationUnit, f analysis.Fetcher) *Index {
	corpus := unit.GetVName().GetCorpus()
	log.Printf("Constructing index for corpus=%s, %q", corpus, unit.GetVName().GetSignature())
	goRoot, osArch, pkgRoot, buildRoot := findRoots(unit)
	if goRoot == "" {
		goRoot = build.Default.GOROOT
	}
	if osArch == "" {
		osArch = build.Default.GOOS + "_" + build.Default.GOARCH
	}
	idx := &Index{
		Compilation: unit,
		fetcher:     f,

		Fset:      token.NewFileSet(),
		Text:      make(map[string][]byte),
		GoRoot:    goRoot,
		osArch:    osArch,
		BuildRoot: buildRoot,
		PkgRoot:   pkgRoot,
		Corpus:    corpus,

		pkgPath:    make(map[string]string),
		pathDigest: make(map[string]string),
	}
	for _, in := range unit.RequiredInput {
		idx.pathDigest[in.GetInfo().GetPath()] = in.GetInfo().GetDigest()
	}
	return idx
}

// Resolve loads and parses the unit's source files and invokes the type
// checker, and returns an error if type-checking could not be accomplished.
// A nil result indicates type checking completed, possibly with type
// errors, which are recorded in Errors.
func (idx *Index) Resolve() error {
	// Derive Go package name from VName's signature.
	// TODO(mataevs) - check if the idx.Compilation.GetVName().GetSignature()
	// is the right place to get this info from.
	at := strings.TrimPrefix(idx.Compilation.GetVName().GetSignature(), "//")
	pkgname := idx.stripRootOSArch(at)
	if pkgname != at {
		// e.g. "//third_party/go/gc/encoding/json" -> "encoding/json"
		// (assumes the build label contains no ':')
		// This is a standard library package.
	} else {
		// e.g. "//base/go:log" -> "google3/base/go/log"
		// This is a user-defined package.
		pkgname = idx.Corpus
		if idx.Corpus != "" {
			pkgname = idx.Corpus + "/"
		}
		pkgname += strings.Replace(at, ":", "/", -1)
	}
	log.Printf("Creating Go package %q (corpus=%q, goroot=%q)", pkgname, idx.Corpus, idx.GoRoot)

	// Parse source files in the compilation unit.
	for _, filename := range idx.Compilation.SourceFile {
		log.Printf("Will fetch file %q", filename)
		src, err := idx.Fetch(filename, idx.pathDigest[filename])
		if err != nil {
			log.Panicf("Error loading %q: %s", filename, err)
			idx.addError(err)
			continue
		}
		idx.Text[filename] = src

		log.Printf("Parsing %q", filename)
		tree, err := parser.ParseFile(idx.Fset, filename, src, parser.ParseComments)
		if err != nil {
			log.Panicf("Error parsing %q: %s", filename, err)
			idx.addError(err)
			continue
		}
		idx.Files = append(idx.Files, tree)
	}
	if len(idx.Files) == 0 {
		return idx.addError(errors.New("empty package"))
	}

	// For each package expressed by one of the required inputs, record a file
	// where that package is defined.  If multiple files define a given package
	// we take the first one.  It's okay for a file to define no package, since
	// the required inputs include things like tool binaries.
	// Source files were handled above, and are ignored here.
	for _, input := range idx.Compilation.RequiredInput {
		filename := input.GetInfo().GetPath()

		// Determine the canonical package name corresponding to
		// the given input filename.
		//
		// This uses a simple filename-based heuristic to determine
		// whether a file should be in a package, and does not
		// read the contents of the file.
		if ext := filepath.Ext(filename); ext == ".a" || ext == ".gcgox" {
			// These two file types contain package type information.
			// .a files are compiled archives, including the type header and code.
			// .gcgox files are package type headers without code.
			importPath := strings.TrimSuffix(filename, ext)

			// If the file has other dotted gunk at the end, strip it off, since
			// that can't belong to a package name anyway.
			for strings.Index(filepath.Base(importPath), ".") > 0 {
				importPath = importPath[:strings.LastIndex(importPath, ".")]
			}

			// Convert file path to an import path.
			switch {
			case idx.GoRoot != "" && strings.HasPrefix(importPath, idx.GoRoot):
				// e.g. "third_party/go/gc/pkg/linux_amd64/encoding/json"
				//   -> "encoding/json"
				importPath = idx.stripRootOSArch(importPath)

			case idx.BuildRoot != nil:
				if ok, br := hasPrefix(importPath, idx.BuildRoot); ok {
					// e.g. "blaze-out/gcc*/bin/base/go/log" -> "google3/base/go/log"
					importPath = strings.TrimLeft(importPath[len(*br):], "/")

					// TODO(mataevs) - check the logic with the corpus. For Kythe,
					// adding the code below will just append another /kythe prefix
					// to all the packages, which is wrong.
					// if idx.Corpus != "" {
					// 	importPath = idx.Corpus + "/" + importPath
					// }
				}
			}

			if importPath != "" && idx.pkgPath[importPath] == "" {
				idx.pkgPath[importPath] = filename
				log.Printf("Map package %q to path %q", importPath, filename)
			}
		}
	}

	// Dependencies are loaded (via resolveImport) from
	// binary files in the compilation unit.
	conf := &loader.Config{
		Fset: idx.Fset,
		TypeChecker: types.Config{
			Packages:    make(map[string]*types.Package),
			Error:       func(err error) { idx.addError(err) },
			Import:      idx.resolveImport,
			FakeImportC: true, // TODO(adonovan): think about this
		},
		AllowErrors: true,
	}

	// Specify the initial package.
	conf.CreateFromFiles(pkgname, idx.Files...)

	iprog, err := conf.Load()
	if err != nil {
		// TODO(adonovan): fix the go/loader bug whereby a single
		// parse error causes Load() to fail without a Program.
		log.Printf("(*loader.Config).Load failed for %q: %s",
			idx.Compilation.GetVName().GetSignature(), err)
		return idx.addError(err)
	}
	info := iprog.Created[0]

	idx.TypeInfo = info.Info // copy
	idx.Package = info.Pkg
	idx.AllPackages = iprog.AllPackages

	return nil // success (perhaps partial)
}

func (idx *Index) stripRootOSArch(s string) string {
	s = strings.TrimPrefix(s, idx.GoRoot)
	s = strings.TrimLeft(s, "/")
	s = strings.TrimPrefix(s, idx.osArch)
	s = strings.TrimLeft(s, "/")
	return s
}

// addError adds the specified error the the Index's slice of errors.  If err
// is a scanner.ErrorList, its component errors are added separately.
func (idx *Index) addError(err error) error {
	if s, ok := err.(scanner.ErrorList); ok {
		for _, e := range s {
			idx.Errors = append(idx.Errors, e)
		}
	} else {
		idx.Errors = append(idx.Errors, err)
	}
	return err
}

// resolveImport is used to implement types.Importer for an Index.  It is
// basically a clone of go/gcimporter.Import, except that it reads files from the
// Compilation instead of via os.Open, and resolves package names manually.
func (idx *Index) resolveImport(imports map[string]*types.Package, path string) (pkg *types.Package, err error) {
	log.Printf("resolveImport: path %q", path)
	defer func() {
		if err != nil {
			log.Printf("resolveImport(%s) failed: %s", path, err)
		} else {
			log.Printf("resolveImport(%s) = %v, scope %+v", path, pkg, pkg.Scope())
		}
	}()
	if path == "unsafe" {
		return types.Unsafe, nil
	}

	// If the package was already completely imported, use the previous value.
	// Continue on partial imports, however.
	if pkg := imports[path]; pkg != nil && pkg.Complete() {
		return pkg, nil
	}

	var filename string
	var buf *bufio.Reader
	var data []byte
	var ok bool
	// Find a file that purports to define the package.
	if filename, ok = idx.pkgPath[path]; ok {
		log.Printf("Resolved import path %q to %q", path, filename)

		data, err = idx.Fetch(filename, idx.pathDigest[filename])
		if err != nil {
			return nil, fmt.Errorf("unable to load %q: %s", filename, err)
		}
	} else {
		// Try mapping the package to a filename from the GORoot directory.
		// All standard packages should fall in this category.
		filename = strings.TrimRight(idx.GoRoot, "/") + "/pkg/" + idx.osArch + "/" + path + ".a"
		if _, err = os.Stat(filename); os.IsNotExist(err) {
			return nil, fmt.Errorf("unable to resolve package: %q", path)
		}
		log.Printf("Resolved import path %q to %q, fetching from Go root directory", path, filename)
		data, err = ioutil.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("unable to read %q: %s", filename, err)
		}
	}

	buf = bufio.NewReader(bytes.NewBuffer(data))
	if err = gcimporter.FindExportData(buf); err != nil {
		return nil, err
	}

	return gcimporter.ImportData(imports, filename, path, buf)
}

func hasPrefix(importPath string, buildRoot []*string) (bool, *string) {
	for _, br := range buildRoot {
		if strings.HasPrefix(importPath, *br) {
			return true, br
		}
	}
	return false, nil
}

// findRoots scans the arguments of a compilation and returns the directories
// for the Go library packages (goRoot) and build outputs (buildRoot).
// If either directory cannot be inferred, returns "" for that value.
func findRoots(unit *apb.CompilationUnit) (goRoot, osArch, pkgPath string, buildRoot []*string) {
	const (
		compilerTool = "--go-compiler"
		endToolMark  = ";"
	)

	var goOS, goArch, toolName string
	var fillNext *string

	for _, arg := range unit.Argument {
		switch arg {
		case compilerTool, endToolMark:
			toolName = arg
			continue
		case "--goroot":
			fillNext = &goRoot
			continue
		case "--goos":
			fillNext = &goOS
			continue
		case "--goarch":
			fillNext = &goArch
			continue
		case "-I":
			switch toolName {
			case "", compilerTool:
				br := new(string)
				buildRoot = append(buildRoot, br)
				fillNext = br
			}
			continue
		case "-p":
			fillNext = &pkgPath
			continue
		}
		if fillNext != nil {
			*fillNext = arg
			fillNext = nil
		}
	}

	target := strings.TrimLeft(unit.GetVName().GetSignature(), "/")
	if i := strings.LastIndex(target, ":"); i >= 0 {
		target = target[:i]
	}

	// Isolate the build root from the include directory by trimming at the
	// package name.
	for _, br := range buildRoot {
		if i := strings.Index(*br, target); i >= 0 {
			nbr := strings.TrimRight((*br)[:i], "/")
			buildRoot = append(buildRoot, &nbr)
		}
	}

	if goOS != "" && goArch != "" {
		osArch = fmt.Sprintf("pkg/%s_%s", goOS, goArch)
	}

	log.Printf("findRoots(targetBase=%q): goroot=%q, buildRoot=%v, osArch=%q, pkgPath=%q",
		target, goRoot, buildRoot, osArch, pkgPath)

	return
}

// Fetch implements analysis.Fetcher by delegating to the fetcher provided to
// the constructor.
func (idx *Index) Fetch(path, digest string) ([]byte, error) {
	return idx.fetcher.Fetch(path, digest)
}
