// Package grokuri implements the Grok URI scheme.
//
// A Grok URI defines a directory, file, or part of a file to the Grok service
// and its indexers.  It is an application of the URI grammar specified in RFC
// 3986 and RFC 3987, defining the "grok" scheme.
//
// The general syntax of a Grok URI is:
//   grok://<corpus>/<path>?<metadata>#<fragment>
//
// The <corpus> field identifies the corpus where the file is stored.  This
// field is require for a valid Grok URI; all other fields are optional.
//
// The general syntax of the <corpus> field is:
//   <branch>@<hostname>;<label>$<revision>
//
// Of these fields, <label> is required; all other fields are optional.  This
// format allows a Grok URI to be parsed by a standard URI parser.
// Examples of valid corpus strings include:
//    google3               -- a <label> alone
//    release@google3       -- a <branch> and a <label>
//    chromium.org;chrome   -- a <hostname> and a <label>
//    google3$28659871      -- a <label> and a <revision> (here, a Piper CL).
//
// Usage examples:
// 1. Create a new Grok URI for the "google3" corpus, set its path and revision.
//      u := grokuri.New("google3")
//      u.Path = "devtools/grok/lang/go"
//      u.Revision = "2859871"
//    Setting a field to the empty string will "clear" it.
//
// 2. Parse a Grok URI from its string representation.
//    The corpus label is inferred from the URI string if possible:
//      u, err := grokuri.Parse("grok://google3/file/base/file.h")
//    Or, a default corpus label may be supplied:
//      u, err := grokuri.ParseDefault("file/base/file.h", "google3")
//
// 3. Convert a Grok URI to a string:
//      s := u.String()      --> "grok://google3/file/base/file.h"
//    Note that converting a Grok URI to a string may normalize some of the
//    provided fields; in particular, paths may be "cleaned" by splicing out
//    double slashes, "." and ".." notations, and trailing delimiters.
//
// 4. Extract a corpus-rooted path as a string:
//      s := u.CorpusPath()  --> "google3/file/base/file.h"
//
package grokuri

import (
	"errors"
	"net/url"
	"path"
	"regexp"
	"strings"
)

// TODO(mataevs) - This whole package is temporary. We will transition to a
// new URI for Kythe. When that will become available, we will replace this
// package (or rename all the elements from Grok to Kythe).

// SchemeLabel is the URI scheme used by Grok URIs.
const SchemeLabel = "kythe"

var (
	// Authority: [<host>;]<label>[$<revision>]
	authExpr = regexp.MustCompile(`(?:([^:;]+);)?([^\$]+)(?:\$(.*))?`)
)

// A GrokURI represents a parsed URI in the "grok" scheme.
type GrokURI struct {
	BranchName  string
	CorpusHost  string
	Revision    string
	Path        string
	Fragment    string
	Metadata    map[string]string
	corpusLabel string
}

// New returns a GrokURI with the specified corpus label.
// Returns nil if the provided label is "".
func New(label string) *GrokURI {
	if label == "" {
		return nil
	}
	return &GrokURI{
		corpusLabel: label,
		Metadata:    make(map[string]string),
	}
}

// PathURI returns a GrokURI for a corpus-relative path, where label specifies
// the corpus.  Returns nil if the provided label is "".
func PathURI(path, label string) *GrokURI {
	out := New(label)
	if out != nil {
		out.Path = path
	}
	return out
}

// Parse a Grok URI from its string representation.
// Equivalent to ParseDefault(raw, "").
func Parse(raw string) (*GrokURI, error) {
	return ParseDefault(raw, "")
}

// ParseDefault parses a Grok URI from its string representation.
// The corpus label is inferred from the string, if possible; otherwise, corpus is
// used as a default.  It is an error if no corpus label is found.
func ParseDefault(raw, corpus string) (*GrokURI, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}

	// If the scheme is specified at all, it must match ours.
	if parsed.Scheme != "" && parsed.Scheme != SchemeLabel {
		return nil, errors.New("invalid scheme: " + parsed.Scheme)
	}

	out := new(GrokURI)
	out.Metadata = make(map[string]string)
	out.Fragment = parsed.Fragment
	if parsed.User != nil {
		out.BranchName = parsed.User.String()
	}

	// A URI with a scheme but no authority may be opaque.
	if parsed.Path == "" && parsed.Opaque != "" {
		out.Path = strings.TrimLeft(parsed.Opaque, "/")
	} else {
		out.Path = strings.TrimLeft(parsed.Path, "/")
	}

	if err = out.parseAuthority(parsed.Host); err != nil {
		return nil, err
	}

	if err = out.parseMetadata(parsed.RawQuery); err != nil {
		return nil, err
	}

	// Verify that the corpus label is valid, using the default if needed.
	if out.corpusLabel == "" {
		if corpus == "" {
			return nil, errors.New("missing corpus label")
		}
		out.corpusLabel = corpus
	}
	return out, nil
}

func (u *GrokURI) parseAuthority(raw string) error {
	if raw == "" {
		return nil
	}
	matches := authExpr.FindStringSubmatch(raw)
	if matches == nil {
		return errors.New("invalid authority field")
	}

	u.CorpusHost = matches[1]
	u.corpusLabel = matches[2]
	u.Revision = matches[3]
	return nil
}

func (u *GrokURI) parseMetadata(raw string) error {
	query, err := url.ParseQuery(raw)
	if err != nil {
		return err
	}

	for k, v := range query {
		if len(v) > 0 {
			u.Metadata[k] = v[0]
		} else {
			u.Metadata[k] = ""
		}
	}

	return nil
}

func (u *GrokURI) String() string {
	// The url module does some slightly strange things with hostnames, so
	// we'll treat this as an opaque URI even if we don't have to
	opaque := u.unparseCorpus()
	if p := cleanPath(u.Path); p != "" {
		opaque += "/" + p
	}
	t := &url.URL{
		Scheme:   SchemeLabel,
		Opaque:   escape(opaque),
		Fragment: u.Fragment,
		RawQuery: u.unparseQuery(),
	}
	return t.String()
}

// CorpusPath returns the normalized path prepended by the corpus label.
func (u *GrokURI) CorpusPath() string {
	cpath := u.corpusLabel
	if p := cleanPath(u.Path); p != "" {
		cpath += "/" + p
	}
	return cpath
}

func escape(s string) string {
	// This is a hack to get around the fact that the url package doesn't
	// expose its escape function in a re-usable way.  url.QueryEscape will
	// replace ' ' with '+' which isn't what we want.
	u := &url.URL{Path: s}
	return u.String()
}

func cleanPath(raw string) string {
	p := strings.Trim(path.Clean(raw), "/")
	if p == "." {
		return ""
	}
	return p
}

func (u *GrokURI) unparseQuery() string {
	var ss []string
	for k, v := range u.Metadata {
		if v == "" {
			ss = append(ss, escape(k))
		} else {
			ss = append(ss, escape(k)+"="+escape(v))
		}
	}
	return strings.Join(ss, "&")
}

func (u *GrokURI) unparseCorpus() string {
	ss := []string{"//"}
	if u.BranchName != "" {
		ss = append(ss, u.BranchName+"@")
	}
	if u.CorpusHost != "" {
		ss = append(ss, u.CorpusHost+";")
	}
	ss = append(ss, u.corpusLabel)
	if u.Revision != "" {
		ss = append(ss, "$"+u.Revision)
	}
	return strings.Join(ss, "")
}

// ParseRelative creates a new Grok URI relative to this one.  The fields of
// this URI are used as defaults for missing fields in the result.  Metadata
// are not copied.
func (u *GrokURI) ParseRelative(raw string) *GrokURI {
	result, err := ParseDefault(raw, u.corpusLabel)
	if err != nil {
		return nil
	}
	if result.BranchName == "" {
		result.BranchName = u.BranchName
	}
	if result.CorpusHost == "" {
		result.CorpusHost = u.CorpusHost
	}
	if result.Revision == "" {
		result.Revision = u.Revision
	}
	if result.Path == "" {
		result.Path = u.Path
	}
	if result.Fragment == "" {
		result.Fragment = u.Fragment
	}
	return result
}

// MakePathRelative trims a path to be relative to the URI's corpus.
func (u *GrokURI) MakePathRelative(path string) string {
	return MakePathRelative(path, u.corpusLabel)
}

// CleanPath returns the normalized path of the URI, sans corpus.
func (u *GrokURI) CleanPath() string {
	return cleanPath(u.Path)
}

// Directory returns the directory component of the normalized path.
func (u *GrokURI) Directory() string {
	p := cleanPath(u.Path)
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[:i]
	}
	return p
}

// Filename returns the last slash-delimited component of the normalized path.
func (u *GrokURI) Filename() string {
	p := cleanPath(u.Path)
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}
