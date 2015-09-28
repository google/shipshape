package androidlint

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	testutil "shipshape/util/test"
)

func init() {
	// Use the path to Android Lint that is visible to Bazel.
	bin, err := filepath.Abs("shipshape/androidlint_analyzer/androidlint/lint")
	if err != nil {
		panic(err)
	}
	lintBin = bin

	linkedBin, err := filepath.EvalSymlinks(lintBin)
	if err != nil {
		panic(err);
	}
	if filepath.Base(linkedBin) == "false" {
		fmt.Println("Android Lint not installed.")
		os.Exit(1)
	}
}

func TestAnalyze(t *testing.T) {
	tests := []struct {
		dir        string
		files      []string
		numResults int
	}{
		{
			"shipshape/androidlint_analyzer/androidlint/testdata",
			[]string{"TicTacToeLib/res/values/strings.xml"},
			9,
		},
		{
			"shipshape/androidlint_analyzer/androidlint/testdata",
			[]string{"TicTacToeLib/res/values/strings.xml", "TicTacToeMain/res/values/strings.xml"},
			17,
		},
		{
			"shipshape/androidlint_analyzer/androidlint/testdata/TicTacToeMain",
			[]string{"res/values/strings.xml"},
			8,
		},
		{
			"shipshape/androidlint_analyzer/androidlint/testdata",
			[]string{"TicTacToeLib/src/com/example/android/tictactoe/library/GameView.java"},
			9,
		},
		{
			"shipshape/androidlint_analyzer/androidlint/testdata",
			[]string{"OtherProject/strings.xml"},
			0,
		},
	}

	var a Analyzer
	for _, test := range tests {
		ctx, err := testutil.CreateContext(test.dir, test.files)
		if err != nil {
			t.Fatalf("error from CreateContext: %v", err)
		}

		actualNotes, err := testutil.RunAnalyzer(ctx, a, t)
		if err != nil {
			t.Errorf("received an analysis failure: %v", err)
		}
		if len(actualNotes) != test.numResults {
			t.Errorf("Number of results: got %d, want %d", len(actualNotes), test.numResults)
		}
	}
}
