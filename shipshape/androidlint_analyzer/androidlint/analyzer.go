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

package androidlint

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	notepb "shipshape/proto/note_proto"
	ctxpb "shipshape/proto/shipshape_context_proto"
	rangepb "shipshape/proto/textrange_proto"

	"github.com/golang/protobuf/proto"
)

const (
	androidManifest = "AndroidManifest.xml"
	report          = "lint-report.xml"
)

var (
	exitStatus = "exit status 1"
)

// Analyzer is a wrapper around the Android lint tool. This analyzer looks for
// any files in the depot path that exist within an Android project. If any
// exist, it runs Android lint on the entire project.
type Analyzer struct{}

func (Analyzer) Category() string { return "AndroidLint" }

// Analyze runs android lint on all Android projects that are implicitly
// referenced by the list of files on the depot path. In case of an error, it
// returns partial results.
func (ala Analyzer) Analyze(ctx *ctxpb.ShipshapeContext) ([]*notepb.Note, error) {
	var notes []*notepb.Note

	// Get the list of Android Projects
	projects := getAndroidProjects(ctx.FilePath)

	for prj := range projects {
		tempReport, err := ioutil.TempFile("", report)
		if err != nil {
			return notes, fmt.Errorf("Could not create temp report file %s: %v", report, err)
		}
		defer os.Remove(tempReport.Name())

		// TODO(ciera): Add project options (--classpath) when we have build information.
		cmd := exec.Command("lint",
			"--showall",
			"--quiet",
			"--exitcode",
			"--xml", tempReport.Name(),
			prj)
		out, err := cmd.CombinedOutput()

		log.Printf("lint output is %s", out)

		switch err := err.(type) {
		case nil:
			// no issues. Do nothing.
		case *exec.ExitError:
			if err.Error() != exitStatus {
				return notes, fmt.Errorf("Unexpected error code from android lint: %v", err)
			}

			// Get the results from xml
			data, xmlErr := ioutil.ReadFile(tempReport.Name())
			if xmlErr != nil {
				return notes, fmt.Errorf("could not read %s : %v", tempReport.Name(), xmlErr)
			}

			var issues IssuesList
			xmlErr = xml.Unmarshal(data, &issues)
			if xmlErr != nil {
				return notes, fmt.Errorf("could not unmarshal XML from %s: %v", tempReport.Name(), xmlErr)
			}

			for _, issue := range issues.Issues {
				notes = append(notes, &notepb.Note{
					Category:    proto.String(ala.Category()),
					Subcategory: proto.String(issue.Subcategory),
					Description: proto.String(issue.Message),
					Location: &notepb.Location{
						SourceContext: ctx.SourceContext,
						Path:          proto.String(filepath.Join(prj, issue.Location.File)),
						Range: &rangepb.TextRange{
							StartLine:   proto.Int(issue.Location.Line),
							StartColumn: proto.Int(issue.Location.Column),
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

// Methods for determining where the android projects exist

func getAndroidProjects(paths []string) map[string]bool {
	pPaths := make(map[string]bool)
	for _, path := range paths {
		project, ok := getProject(filepath.Clean(path))
		if ok {
			pPaths[project] = true
		}
	}
	return pPaths
}

func getProject(path string) (string, bool) {
	fi, err := os.Stat(path)
	if err != nil {
		log.Printf("Could not find path %s: %v", path, err)
		return "", false
	}
	if fi.IsDir() {
		isAndroid := isAndroidRoot(path)
		if isAndroid || path == "." || path == string(filepath.Separator) {
			return path, isAndroid
		}
	}
	return getProject(filepath.Dir(path))
}

func isAndroidRoot(path string) bool {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Printf("had invalid path in android lint: %s %v", path, err)
		return false
	}

	for _, file := range files {
		if file.Name() == androidManifest {
			return true
		}
	}
	return false
}

// Types below are for extracting the messages
// out of the XML report

type IssuesList struct {
	Issues []Issue `xml:"issue"`
}

type Issue struct {
	Subcategory string   `xml:"id,attr"`
	Message     string   `xml:"message,attr"`
	Location    Location `xml:"location"`
}
type Location struct {
	File   string `xml:"file,attr"`
	Line   int    `xml:"line,attr"`
	Column int    `xml:"column,attr"`
}
