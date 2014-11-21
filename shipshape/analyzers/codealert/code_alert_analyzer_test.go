package codealert

import (
	"testing"
)

func TestSamplePattern(t *testing.T) {
	var a CodeAlertAnalyzer
	notes := a.FindMatches("\nsome text\ndo not submit\nmore text\n")
	got := len(notes)
	if got != 1 {
		t.Errorf("Number of matches, got %v, want %q", got, 1)
	}
	const want = "CodeAlert"
	for _, note := range notes {
		got := note.GetCategory()
		if got != want {
			t.Errorf("Note category, got %v, want %v", got, want)
		}
	}
}
