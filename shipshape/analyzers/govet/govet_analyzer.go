// Package govet implements a Shipshape analyzer that runs go vet over all Go
// files in the given ShipshapeContext.
package govet

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
	exitStatus = "exit status 1"
)

var (
	issueRE = regexp.MustCompile(`([^:]*):([0-9]+): (.*)`)
)

// GoVetAnalyzer is a wrapper around the go vet command line tool.
// This assumes it runs in a location where go is on the path.
type GoVetAnalyzer struct{}

func (GoVetAnalyzer) Category() string { return "go vet" }

func isGoFile(path string) bool {
	return filepath.Ext(path) == ".go"
}

func (gva *GoVetAnalyzer) analyzeOneFile(ctx *ctxpb.ShipshapeContext, path string) ([]*notepb.Note, error) {
	var notes []*notepb.Note
	cmd := exec.Command("go", "vet", path)
	buf, err := cmd.CombinedOutput()

	switch err := err.(type) {
	case nil:
		// No issues reported, do nothing.
	case *exec.ExitError:
		// go vet exits with an error when there are findings to report.
		if err.Error() != exitStatus {
			return notes, err
		}

		// go vet gives one issue per line, with the penultimate line indicating
		// the exit code and the last line being empty.
		var issues = strings.Split(string(buf), "\n")
		if len(issues) < 3 {
			// TODO(ciera): We should be able to keep going here
			// and try the next file. However, our API doesn't allow for
			// returning multiple errors. We need to reconsider the API.
			return notes, errors.New("did not get correct output from `go vet`")
		}
		for _, issue := range issues[:len(issues)-2] {
			parts := issueRE.FindStringSubmatch(issue)
			if len(parts) != 4 {
				return notes, fmt.Errorf("`go vet` gave incorrectly formatted issue: %q", issue)
			}

			filename := parts[1]
			description := parts[3]

			// Convert the line number into a base-10 32-bit int.
			line, err := strconv.ParseInt(parts[2], 10, 32)
			if err != nil {
				return notes, err
			}

			notes = append(notes, &notepb.Note{
				// TODO(collinwinter): we should synthesize subcategories here.
				Category:    proto.String(gva.Category()),
				Description: proto.String(description),
				Location: &notepb.Location{
					SourceContext: ctx.SourceContext,
					Path:          proto.String(filename),
					Range: &rangepb.TextRange{
						StartLine: proto.Int32(int32(line)),
					},
				},
			})
		}
	default:
		return notes, err
	}
	return notes, nil
}

func (gva *GoVetAnalyzer) Analyze(ctx *ctxpb.ShipshapeContext) ([]*notepb.Note, error) {
	var notes []*notepb.Note

	// Call go vet on each go file individually. go vet requires that all files
	// given be in the same directory, and this is an easy way of achieving that.
	for _, path := range ctx.FilePath {
		if !isGoFile(path) {
			continue
		}

		ourNotes, err := gva.analyzeOneFile(ctx, path)
		// TODO(collinwinter): figure out whether analyzers should return an
		// error XOR notes and impose that everywhere.
		notes = append(notes, ourNotes...)
		if err != nil {
			return notes, err
		}
	}
	return notes, nil
}
