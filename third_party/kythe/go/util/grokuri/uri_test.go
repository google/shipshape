package grokuri

import (
	"testing"
)

func checkURI(t *testing.T, u *GrokURI, branch, host, label, rev, path, frag string) {
	if u.BranchName != branch {
		t.Errorf("BranchName: want %q, have %q", branch, u.BranchName)
	}
	if u.CorpusHost != host {
		t.Errorf("CorpusHost: want %q, have %q", host, u.CorpusHost)
	}
	if u.corpusLabel != label {
		t.Errorf("corpusLabel: want %q, have %q", label, u.corpusLabel)
	}
	if u.Revision != rev {
		t.Errorf("Revision: want %q, have %q", rev, u.Revision)
	}
	if u.Path != path {
		t.Errorf("Path: want %q, have %q", path, u.Path)
	}
	if u.Fragment != frag {
		t.Errorf("Fragment: want %q, have %q", frag, u.Fragment)
	}
}

func checkString(t *testing.T, u *GrokURI, want string) {
	if have := u.String(); have != want {
		t.Errorf("String: want %q, have %q", want, have)
	}
}

func TestBasicRules(t *testing.T) {
	u := New("corpus_label")
	checkURI(t, u, "", "", "corpus_label", "", "", "")
	checkString(t, u, "kythe://corpus_label")

	// For various combinations of fields, verify rendering the URI to a string.
	u.BranchName = "branch"
	checkURI(t, u, "branch", "", "corpus_label", "", "", "")
	checkString(t, u, "kythe://branch@corpus_label")

	u.CorpusHost = "google.com"
	checkURI(t, u, "branch", "google.com", "corpus_label", "", "", "")
	checkString(t, u, "kythe://branch@google.com;corpus_label")

	u.corpusLabel = "new_corpus"
	checkURI(t, u, "branch", "google.com", "new_corpus", "", "", "")
	checkString(t, u, "kythe://branch@google.com;new_corpus")

	u.Revision = "revision"
	checkURI(t, u, "branch", "google.com", "new_corpus", "revision", "", "")
	checkString(t, u, "kythe://branch@google.com;new_corpus$revision")

	u.Path = "some/path"
	checkURI(t, u, "branch", "google.com", "new_corpus", "revision", "some/path", "")
	checkString(t, u, "kythe://branch@google.com;new_corpus$revision/some/path")

	u.Fragment = "fragment"
	checkURI(t, u, "branch", "google.com", "new_corpus", "revision", "some/path", "fragment")
	checkString(t, u, "kythe://branch@google.com;new_corpus$revision/some/path#fragment")

	// Removing an existing field by setting it to "" should work.
	u.Revision = ""
	checkURI(t, u, "branch", "google.com", "new_corpus", "", "some/path", "fragment")
	checkString(t, u, "kythe://branch@google.com;new_corpus/some/path#fragment")

	// Parsing a Grok URI with an incorrect scheme should fail.
	if u, err := Parse("http://some.bad.thing"); err == nil {
		t.Errorf("parse succeeded with incorrect scheme: %+v", u)
	}
}

func TestMetadata(t *testing.T) {
	// Metadata should be correctly added, both with and without a value.
	u := New("google3")
	u.Path = "whatever"
	u.Metadata["solo"] = ""
	checkURI(t, u, "", "", "google3", "", "whatever", "")
	checkString(t, u, "kythe://google3/whatever?solo")

	u = New("google3")
	u.Metadata["alpha"] = "big hairy deal"
	checkURI(t, u, "", "", "google3", "", "", "")
	checkString(t, u, "kythe://google3?alpha=big%20hairy%20deal")

	// Multiple metadata values.  We can't compare strings, because map
	// order is undefined (the test might be flaky); so we'll re-parse
	u = New("google3")
	u.Metadata["going"] = "off the rails"
	u.Metadata["on a crazy"] = ""
	u.Metadata["train"] = "!"
	v, err := Parse(u.String())
	if err != nil {
		t.Fatalf("re-parse failed: %s", err)
	}
	check := func(key, expect string) {
		if s, ok := v.Metadata[key]; !ok {
			t.Errorf("missing key: %q", key)
		} else if s != expect {
			t.Errorf("metadata %q: want %q, have %q", key, expect, s)
		}
	}
	check("going", "off the rails")
	check("on a crazy", "")
	check("train", "!")
	if _, ok := v.Metadata["no such key"]; ok {
		t.Error("found unexpected key")
	}
}

func TestPathURI(t *testing.T) {
	// An empty corpus gives an error.
	u := PathURI("file/base/file.h", "")
	if u != nil {
		t.Errorf("expected nil, got %+v", u)
	}

	// A non-empty corpus gives a sensible result
	u = PathURI("", "corpus")
	checkString(t, u, "kythe://corpus")
	u = PathURI("file/base/file.h", "corpus")
	checkString(t, u, "kythe://corpus/file/base/file.h")
	u = PathURI("//victory/./in/.//our/time/", "blah")
	checkString(t, u, "kythe://blah/victory/in/our/time")
}

func TestRelativeURI(t *testing.T) {
	// Parsing a URI with no corpus and no default should fail.
	u, err := Parse("kythe:no/corpus/label")
	if err == nil {
		t.Errorf("expected error, got %+v", u)
	}

	// Parsing a URI with no corpus but a supplied default should give a valid
	// URI that includes the default.

	// Case 1: No scheme (fall back on default).
	u, err = ParseDefault("a/b/c", "google3")
	checkURI(t, u, "", "", "google3", "", "a/b/c", "")
	checkString(t, u, "kythe://google3/a/b/c")

	// Case 2: Scheme provided, must match "kythe".
	u, err = ParseDefault("kythe:a/b/c", "google3")
	checkURI(t, u, "", "", "google3", "", "a/b/c", "")
	checkString(t, u, "kythe://google3/a/b/c")

	u, err = ParseDefault("kythe:/a/b/c", "google3")
	checkURI(t, u, "", "", "google3", "", "a/b/c", "")
	checkString(t, u, "kythe://google3/a/b/c")

	u, err = ParseDefault("kythe:a/b/c#bad%20horse", "google3")
	checkURI(t, u, "", "", "google3", "", "a/b/c", "bad horse")
	checkString(t, u, "kythe://google3/a/b/c#bad%20horse")

	// Absolute paths should be trimmed to corpus-relative.
	u, err = ParseDefault("/absolut/pathka", "google3")
	checkURI(t, u, "", "", "google3", "", "absolut/pathka", "")
	checkString(t, u, "kythe://google3/absolut/pathka")
	u.Path = "/some/other/thing"
	checkString(t, u, "kythe://google3/some/other/thing")

	// Verify that paths are simplified before rendering to a string.
	u, err = ParseDefault("a/../b/./c//d/", "testcorpus")
	checkURI(t, u, "", "", "testcorpus", "", "a/../b/./c//d/", "")
	checkString(t, u, "kythe://testcorpus/b/c/d")
}

func TestParseRelative(t *testing.T) {
	// Verify that parsing relative to an existing URI works.
	u, _ := Parse("kythe://branch@google3$rev/a/b/c?foo#bar")
	checkURI(t, u, "branch", "", "google3", "rev", "a/b/c", "bar")

	v := u.ParseRelative("some/other/path")
	checkURI(t, v, "branch", "", "google3", "rev", "some/other/path", "bar")
	checkString(t, v, "kythe://branch@google3$rev/some/other/path#bar")

	// Changing the original URI doesn't affect the new one.
	u.BranchName = ""
	u.Fragment = ""
	checkURI(t, v, "branch", "", "google3", "rev", "some/other/path", "bar")
	checkString(t, v, "kythe://branch@google3$rev/some/other/path#bar")

	// ... but making a new URI from the changed URI works.
	v = u.ParseRelative("kythe://alternate/path/to/goodness?blah")
	checkURI(t, v, "", "", "alternate", "rev", "path/to/goodness", "")
	checkString(t, v, "kythe://alternate$rev/path/to/goodness?blah")
}

func TestCornerCases(t *testing.T) {
	// No corpus label, and an empty default.
	u, err := Parse("blank")
	if err == nil {
		t.Errorf("expected error, got %+v", u)
	}
}

func TestMakePathRelative(t *testing.T) {
	check := func(path, corpus, want string) {
		if have := MakePathRelative(path, corpus); have != want {
			t.Errorf("path %q relative to %q: want %q, have %q", path, corpus, want, have)
		}
	}
	// Empty and relative paths are unchanged.
	check("", "whatever", "")
	check("a/b/c", "whatever", "a/b/c")

	// No occurrence of the corpus label.
	check("/a/b/c", "whatever", "a/b/c")
	check("///a/b/c", "whatever", "a/b/c")

	// Corpus label occurs first.
	check("/initial/b/c", "initial", "b/c")

	// Corpus label occurs in the middle.
	check("/a/medial/b/c", "medial", "b/c")

	// Corpus label occurs last, with or without trailing "/".
	check("/a/b/final/", "final", "")
	check("/a/b/final", "final", "")

	// Corpus label without bracketing "/" isn't clipped.
	check("/a/not_corpus/corpus/ok", "corpus", "ok")
	check("/a/corpus/not_corpus/ok", "corpus", "not_corpus/ok")
	check("/a/corpus/ok/corpus", "corpus", "ok/corpus")
	check("/a/ok/not_corpus", "corpus", "a/ok/not_corpus")

	// As above, using an existing URI as basis.
	u := New("snip")
	mcheck := func(path, want string) {
		if have := u.MakePathRelative(path); have != want {
			t.Errorf("path %q relative to %q: want %q, have %q",
				path, u.corpusLabel, want, have)
		}
	}
	mcheck("", "")
	mcheck("a/b/c", "a/b/c")

	// No occurrence of the corpus label.
	mcheck("/a/b/c", "a/b/c")
	mcheck("///a/b/c", "a/b/c")

	// Corpus label occurs first.
	mcheck("/snip/b/c", "b/c")

	// Corpus label occurs in the middle.
	mcheck("/a/snip/b/c", "b/c")

	// Corpus label occurs last, with or without trailing "/".
	mcheck("/a/b/snip/", "")
	mcheck("/a/b/snip", "")

	// Corpus label without bracketing "/" isn't clipped.
	mcheck("/a/not_snip/snip/ok", "ok")
	mcheck("/a/snip/not_snip/ok", "not_snip/ok")
	mcheck("/a/snip/ok/snip", "ok/snip")
	mcheck("/a/ok/not_snip", "a/ok/not_snip")
}

func TestCorpusPath(t *testing.T) {
	u := New("google3")
	check := func(u *GrokURI, want string) {
		if have := u.CorpusPath(); have != want {
			t.Errorf("corpus path for %q: want %q, have %q", u.String(), want, have)
		}
	}
	check(u, "google3")

	v := u.ParseRelative("file/base/file.h")
	check(v, "google3/file/base/file.h")

	v = u.ParseRelative("/a/b/c")
	check(v, "google3/a/b/c")

	v, err := Parse("kythe://branch@host;corpus$rev/some/path?query#frag")
	if err != nil {
		t.Errorf("unexpected parse error: %v", err)
	} else {
		check(v, "corpus/some/path")
	}
}

func TestDirectoryAndFilename(t *testing.T) {
	tests := []struct {
		uri, dir, file string
	}{
		{"kythe://google3/a/b/c/d.txt", "a/b/c", "d.txt"},
		{"kythe://corpus/a/b/c/", "a/b", "c"},
		{"kythe://corpus", "", ""},
		{"kythe://corpus/", "", ""},
		{"kythe://corpus/alpha///bravo/../charlie/delta", "alpha/charlie", "delta"},
	}
	for _, test := range tests {
		u, err := Parse(test.uri)
		if err != nil {
			t.Errorf("Invalid URI %q (broken test): %s", test.uri, err)
			continue
		}
		if actual := u.Directory(); actual != test.dir {
			t.Errorf("PathDirectory(%q): want %q, have %q", test.uri, test.dir, actual)
		}
		if actual := u.Filename(); actual != test.file {
			t.Errorf("PathFilename(%q): want %q, have %q", test.uri, test.file, actual)
		}
	}
}

func TestCleanPath(t *testing.T) {
	tests := []struct {
		uri, path string
	}{
		{"kythe://corpus/a/b/c/d.txt", "a/b/c/d.txt"},
		{"kythe://corpus/a/b/c/", "a/b/c"},
		{"kythe://corpus", ""},
		{"kythe://corpus/", ""},
		{"kythe://corpus/alpha///bravo/../charlie/delta", "alpha/charlie/delta"},
		{"//x/a/././b///c//", "a/b/c"},
		{"//xx/a//b/.././c", "a/c"},
	}
	for _, test := range tests {
		u, err := Parse(test.uri)
		if err != nil {
			t.Errorf("Invalid URI %q (broken test): %s", test.uri, err)
			continue
		}
		if actual := u.CleanPath(); actual != test.path {
			t.Errorf("CleanPath(%q): want %q, have %q", test.uri, test.path, actual)
		}
	}
}
