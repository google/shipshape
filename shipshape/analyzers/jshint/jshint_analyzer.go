/*
 * Copyright 2014 Google Inc. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package jshint

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"code.google.com/p/goprotobuf/proto"
	notepb "shipshape/proto/note_proto"
	ctxpb "shipshape/proto/shipshape_context_proto"
	rangepb "shipshape/proto/textrange_proto"
)

const (
	exitStatus = "exit status 2"
)

var (
	issueRE = regexp.MustCompile(`([^:]*): line ([0-9]+), col ([0-9]+), (.*)`)
)

// JSHintAnalyzer is a wrapper around a the jshint command line tool.
// This assumes it runs in a location where jshint is on the path.
// It can run on JS and HTML files. Right now, it just puts the file
// contents into stdin and uses no configuration
type JSHintAnalyzer struct{}

func (JSHintAnalyzer) Category() string { return "JSHint" }

func isJSHintFile(path string) bool {
	switch filepath.Ext(path) {
	// TODO(ciera): we can handle .html ONLY if we pull out the
	// <script> portions. Othewise jshint has a lot of trouble.
	case ".js":
		return true
	default:
		return false
	}
}

func (jsa *JSHintAnalyzer) Analyze(ctx *ctxpb.ShipshapeContext) ([]*notepb.Note, error) {
	var notes []*notepb.Note

	// Call jshint on each path
	for _, path := range ctx.FilePath {
		if !isJSHintFile(path) {
			continue
		}

		cmd := exec.Command("jshint", path)
		buf, err := cmd.CombinedOutput()

		switch err := err.(type) {
		case nil:
			// no issues. Do nothing.
		case *exec.ExitError:
			// jshint gives an exit status when there are issues reporter
			if err.Error() == exitStatus {
				// jshint gives one issue per line
				var issues = strings.Split(string(buf), "\n")
				if len(issues) <= 3 {
					// TODO(ciera): We should be able to keep going here
					// and try the next file. However, our API doesn't allow for
					// returning multiple errors. We need to reconsider the API
					return notes, errors.New("did not get correct output from jshint")
				}
				// ignore the last three lines, they are summary info
				for _, issue := range issues[:len(issues)-3] {
					parts := issueRE.FindStringSubmatch(issue)
					if len(parts) != 5 {
						return notes, fmt.Errorf("jshint gave incorrectly formatted issue: %q", issue)
					}

					// convert into a base-10 32-bit int
					line, err := strconv.ParseInt(parts[2], 10, 32)
					if err != nil {
						return notes, err
					}

					// convert into a base-10 32-bit int
					col, err := strconv.ParseInt(parts[3], 10, 32)
					if err != nil {
						return notes, err
					}

					notes = append(notes, &notepb.Note{
						Category:    proto.String(jsa.Category()),
						Description: proto.String(parts[4]),
						MoreInfo:    proto.String("http://www.jshint.com"),
						Location: &notepb.Location{
							SourceContext: ctx.SourceContext,
							Path:          proto.String(parts[1]),
							Range: &rangepb.TextRange{
								StartLine:   proto.Int32(int32(line)),
								StartColumn: proto.Int32(int32(col)),
							},
						},
					})
				}
			} else {
				return notes, err
			}
		case *exec.Error:
			return notes, err
		default:
			return notes, err
		}
	}
	return notes, nil
}
