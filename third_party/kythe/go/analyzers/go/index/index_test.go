package index

import (
	"sort"
	"testing"

	"third_party/kythe/go/platform/local"
)

const (
	signature     = "//devtools/grok/unit:test"
	packagePath   = "google3/test/bog"
	includePath   = "build_root/blah/devtools/grok/unit/test/bog/stuff"
	goLibRoot     = "goroot_path"
	goOS          = "os"
	goArch        = "arch"
	testGoRoot    = goLibRoot
	testBuildRoot = "build_root/blah"
	testOSArch    = "pkg/os_arch"
	testPkgRoot   = "google3/test/bog"
)

var testArguments = []string{
	"path/to/compiler_tool", "misc0",
	"--go-compiler", "6g", "arg", "-I", includePath, ";",
	"zut", "-p", packagePath,
	"--someflag", "--otherflag", "value",
	"--goroot", goLibRoot,
	"misc1", "--goos", goOS,
	"misc2", "misc3", "--goarch", goArch,
	"misc4", "--quiet",
	"--packager", "pkgtool", "-I", "BOGUS", ";",
	"test/bog/file1.go", "test/bog/file2.go", // Source files
}

// The index tests use this hand-built dummy compilation that uses fixed files
// hard-coded into the test.
var testCompilation = local.NewCompilation()

func init() {
	testCompilation.SetSignature(signature)

	// The test compilation has two files in package foo, with a single
	// resolvable dependency on an archive package bar and an unresolvable
	// import of "does/not/resolve".
	testCompilation.AddData("test/bog/file1.go", []byte("package foo\n"))
	testCompilation.AddData("test/bog/file2.go", []byte(`package foo
import (
	"bar" // Resolves correctly
	"C"   // Resolves correctly
	"does/not/resolve"
)
var baz = bar.Random
`))

	testCompilation.SetSource("test/bog/file1.go")
	testCompilation.SetSource("test/bog/file2.go")

	// Package bar exports a single int variable named "Random".
	//
	// Note: The test data here simulates the required format of a gcgox
	// header. If the archive header format changes, the tests here will break.
	// That's what we want, though, since a change in that format forces a
	// change in the resolver as well.
	testCompilation.AddData("goroot_path/pkg/os_arch/bar.crappengine.gcgox",
		[]byte(`go object linux amd64 blah blah
import
$$
package bar
	var @"".Random int
$$
`))
	// A fake package archive in the build root (and NOT in the library root).
	testCompilation.AddData(testBuildRoot+"/baz.a", []byte(`go object linux amd64
import
$$
package baz
$$`))

	// Set up simulated build tool flags so we can test that inference of the
	// build and library roots works correctly.
	testCompilation.Proto.Argument = testArguments
}

func TestIndexProperties(t *testing.T) {
	idx := New(testCompilation.Proto, testCompilation)

	// The index should have correctly inferred the library and build roots.
	if v := idx.GoRoot; v != testGoRoot {
		t.Errorf("GoRoot: got %q, want %q", v, testGoRoot)
	}

	// Test the build roots
	var testBuildRootSlice = []string{testBuildRoot, includePath}
	sort.Strings(testBuildRootSlice)

	if len(idx.BuildRoot) != len(testBuildRootSlice) {
		t.Errorf("BuildRoot: got %d elements, want %d elements", len(idx.BuildRoot), len(testBuildRootSlice))
	}

	var vslice []string
	for _, broot := range idx.BuildRoot {
		vslice = append(vslice, *broot)
	}
	sort.Strings(vslice)

	for i := range vslice {
		if vslice[i] != testBuildRootSlice[i] {
			t.Errorf("BuildRoot: got %q, want %q", vslice[i], testBuildRootSlice[i])
		}
	}

	// Resolution should not return an error.  There should be
	// an error resulting from the unresolved import, but it should
	// not fail indexing entirely.
	err := idx.Resolve()
	if err != nil {
		t.Errorf("Resolve: got unexpected error: %s", err)
	}

	// There should be at least one error recorded.
	if len(idx.Errors) == 0 {
		t.Error("No errors recorded, want at least 1")
	}
	for i, err := range idx.Errors {
		t.Logf("[ok] Error #%d: %s", i+1, err)
	}

	// There should be source text and ASTs for the source files.
	for i, path := range idx.Compilation.SourceFile {
		if idx.Text[path] == nil {
			t.Errorf("Missing source for %q", path)
		}
		if idx.Files[i] == nil {
			t.Errorf("Missing AST for %q", path)
		}
	}

	// Package bar should have been resolved to the gcgox file.
	if path, ok := idx.pkgPath["bar"]; !ok {
		t.Errorf("Missing package mapping for %q", "bar")
	} else {
		t.Logf("Package %q found in file %q", "bar", path)
	}
}

func TestFindRoots(t *testing.T) {
	gRoot, osArch, pRoot, bRoots := findRoots(local.NewCompilation().Proto)
	if gRoot != "" {
		t.Errorf("GoRoot: got %q, want %q", gRoot, "")
	}
	if bRoots != nil {
		t.Errorf("BuildRoot: got %q, want %q", bRoots, "")
	}
	if osArch != "" {
		t.Errorf("osArch: got %q, want %q", osArch, "")
	}
	if pRoot != "" {
		t.Errorf("PkgRoot: got %q, want %q", pRoot, "")
	}

	gRoot, osArch, pRoot, bRoots = findRoots(testCompilation.Proto)
	if gRoot != testGoRoot {
		t.Errorf("GoRoot: got %q, want %q", gRoot, testGoRoot)
	}

	// Test the build roots.
	var testBuildRootSlice = []string{testBuildRoot, includePath}
	sort.Strings(testBuildRootSlice)

	if len(bRoots) != len(testBuildRootSlice) {
		t.Errorf("BuildRoot: got %d elements, want %d elements", len(bRoots), len(testBuildRootSlice))
	}

	var vslice []string
	for _, broot := range bRoots {
		vslice = append(vslice, *broot)
	}
	sort.Strings(vslice)

	for i := range vslice {
		if vslice[i] != testBuildRootSlice[i] {
			t.Errorf("BuildRoot: got %q, want %q", vslice[i], testBuildRootSlice[i])
		}
	}

	if osArch != testOSArch {
		t.Errorf("osArch: got %q, want %q", osArch, testOSArch)
	}
	if pRoot != testPkgRoot {
		t.Errorf("PkgRoot: got %q, want %q", pRoot, testPkgRoot)
	}
}
