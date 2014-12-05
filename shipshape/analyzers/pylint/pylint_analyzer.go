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

package pylint

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"code.google.com/p/goprotobuf/proto"

	notepb "shipshape/proto/note_proto"
	ctxpb "shipshape/proto/shipshape_context_proto"
	rangepb "shipshape/proto/textrange_proto"
)

const (
	usageError   = "exit status 32"
	modulePrefix = "************* Module"
)

var ()

// PyLintAnalyzer is a wrapper around the pylint command line tool.
// It will first request the local directory for each file it needs
// to run on.
// After running, it will convert findings to the original directory
// structure
type PyLintAnalyzer struct{}

func (PyLintAnalyzer) Category() string { return "PyLint" }

func (pya *PyLintAnalyzer) Analyze(ctx *ctxpb.ShipshapeContext) ([]*notepb.Note, error) {
	var notes []*notepb.Note
	// Call pylint on the files
	pythonFiles := extractPyFiles(ctx.FilePath)
	for _, pyFile := range pythonFiles {
		cmd := exec.Command("pylint",
			// TODO(ciera): get the python path
			//"--init-hook='import sys; sys.path.append(" + pythonpath + ")'",
			"--msg-template='{path}:::{line}:::{msg}'",
			"--reports=no",
			pyFile)
		buf, err := cmd.CombinedOutput()

		switch err := err.(type) {
		case nil:
			// no issues. Do nothing.
		case *exec.ExitError:
			// pylint uses error codes to signal if there was an
			// issue discovered. However, a fatal message means that
			// pylint couldn't even finish processing. There may be partial
			// results though.
			if err.Error() == usageError {
				return notes, err
			}

			var issues = strings.Split(string(buf), "\n")

			if len(issues) < 2 {
				return notes, errors.New("Output contains no issues")
			}
			// one issue per line
			// skip first line; just where config is at
			// skip the empty last line
			for _, issue := range issues[1 : len(issues)-1] {
				if strings.HasPrefix(issue, modulePrefix) {
					continue
				}

				parts := strings.Split(issue, ":::")

				if len(parts) != 3 {
					return notes, fmt.Errorf("Found ill-formated issue: %s", issue)
				}

				// convert into a base-10 32-bit int
				line, err := strconv.ParseInt(parts[1], 10, 32)
				if err != nil {
					return notes, err
				}

				notes = append(notes, &notepb.Note{
					Category:    proto.String(pya.Category()),
					Description: proto.String(strings.TrimSpace(parts[2])),
					Location: &notepb.Location{
						SourceContext: ctx.SourceContext,
						Path:          proto.String(parts[0]),
						Range: &rangepb.TextRange{
							StartLine: proto.Int32(int32(line)),
						},
					},
				})

			}
		case *exec.Error:
			return notes, err
		default:
			return notes, err
		}
	}
	return notes, nil
}

func extractPyFiles(paths []string) []string {
	pyFiles := []string{}
	for _, path := range paths {
		if filepath.Ext(path) == ".py" {
			pyFiles = append(pyFiles, path)
		}
	}
	return pyFiles
}
