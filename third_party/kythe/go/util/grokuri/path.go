// Copyright 2012 Google Inc. All Rights Reserved.
// Author: fromberger@google.com (Michael Fromberger)

package grokuri

import (
	"strings"
)

// MakePathRelative converts an absolute path to a corpus-relative path by
// clipping it at the leftmost occurrence of corpusLabel, if any.  Relative
// paths are not changed.  The result may be an empty string.
//
// This is intended for use with paths that aren't relative to the corpus root,
// which should generally only occur with testing or repro inputs.
func MakePathRelative(path, corpusLabel string) string {
	if path == "" || !strings.HasPrefix(path, "/") {
		return path
	}

	var (
		needle   = "/" + corpusLabel + "/"
		labelPos = strings.Index(path, needle)
	)

	// Case 1: Found /corpusLabel/.  Snip the path after it.
	if labelPos >= 0 {
		return path[labelPos+len(needle):]
	}

	// Case 2: Path has the form .../corpusLabel.  Result is empty.
	if strings.HasSuffix(path, "/"+corpusLabel) {
		return ""
	}

	// Case 3: Use the whole path (sans leading "/" characters).
	return strings.TrimLeft(path, "/")
}
