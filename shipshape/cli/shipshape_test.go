package cli

import (
	"testing"
)

// TODO(ciera): Make a testdata directory that contains test files
// for all these tests.

func TestExternalAnalyzers(t *testing.T) {
	// Replaces part of the e2e test
	// Create a fake maven project with android failures

	// Run CLI using a .shipshape file
}

func TestBuiltInAnalyzersPreBuild(t *testing.T) {
	// Replaces part of the e2e test
	// Test PostMessage and Go, with no kythe build

}

func TestBuiltInAnalyzersPostBuild(t *testing.T) {
	// Replaces part of the e2e test
	// Test with a kythe maven build
	// PostMessage and ErrorProne
}

func TestStreamsMode(t *testing.T) {
	// Test whether it works in streams mode
	// Before creating this, ensure that streams mode
	// is actually still something we need to support.
}

func TestChangingDirectories(t *testing.T) {
	// Replaces the changedir test
	// Make sure to test changing down, changing up, running on the same directory, running on a single file in the same directory, and changing to a sibling
}

func dumpLogs() {

}

func checkOutput(category string, numResults int) {

}
