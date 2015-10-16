package cli

import (
	"flag"
	"testing"

	rpcpb "github.com/google/shipshape/shipshape/proto/shipshape_rpc_proto"
	"github.com/google/shipshape/shipshape/util/defaults"
	"github.com/google/shipshape/shipshape/util/docker"
)

var (
	// There are two ways to specify test flags when using Bazel:
	// 1) In the BUILD file with an args stanza in the _test rule.
	// 2) On the command line using --test_arg (i.e. bazel test --test_arg=-shipshape_test_docker_tag=TAG ...).
	//
	// As of 9 Oct 2015, there are multiple Bazel targets that use --shipshape_test_docker_tag (:test_prod, :test_staging,
	// and :test_local) but there are no targets that set local Kythe.
	dockerTag  = flag.String("shipshape_test_docker_tag", "", "the docker tag for the images to use for testing")
	localKythe = flag.Bool("shipshape_test_local_kythe", false, "if true, don't pull the Kythe docker image")
)

func countFailures(resp rpcpb.ShipshapeResponse) int {
	failures := 0
	for _, analyzeResp := range resp.AnalyzeResponse {
		failures += len(analyzeResp.Failure)
	}
	return failures
}

func countNotes(resp rpcpb.ShipshapeResponse) int {
	notes := 0
	for _, analyzeResp := range resp.AnalyzeResponse {
		notes += len(analyzeResp.Note)
	}
	return notes
}

func countCategoryNotes(resp rpcpb.ShipshapeResponse, category string) int {
	notes := 0
	for _, analyzeResp := range resp.AnalyzeResponse {
		for _, note := range analyzeResp.Note {
			if *note.Category == category {
				notes += 1
			}
		}
	}
	return notes
}

func TestExternalAnalyzers(t *testing.T) {
	// Replaces part of the e2e test
	// Create a fake maven project with android failures

	// Run CLI using a .shipshape file
}

func TestBuiltInAnalyzersPreBuild(t *testing.T) {
	options := Options{
		File:                "shipshape/cli/testdata/workspace1",
		ThirdPartyAnalyzers: []string{},
		Build:               "",
		TriggerCats:         []string{"PostMessage", "JSHint", "go vet", "PyLint"},
		Dind:                false,
		Event:               defaults.DefaultEvent,
		Repo:                defaults.DefaultRepo,
		StayUp:              true,
		Tag:                 *dockerTag,
		LocalKythe:          *localKythe,
	}
	var allResponses rpcpb.ShipshapeResponse
	options.HandleResponse = func(shipshapeResp *rpcpb.ShipshapeResponse, _ string) error {
		allResponses.AnalyzeResponse = append(allResponses.AnalyzeResponse, shipshapeResp.AnalyzeResponse...)
		return nil
	}
	returnedNotesCount, err := New(options).Run()
	if err != nil {
		t.Fatal(err)
	}
	testName := "TestBuiltInAnalyzerPreBuild"

	if got, want := countFailures(allResponses), 0; got != want {
		t.Errorf("%v: Wrong number of failures; got %v, want %v (proto data: %v)", testName, got, want, allResponses)
	}
	if countedNotes := countNotes(allResponses); returnedNotesCount != countedNotes {
		t.Errorf("%v: Inconsistent note count: returned %v, counted %v (proto data: %v", testName, returnedNotesCount, countedNotes, allResponses)
	}
	if got, want := returnedNotesCount, 39; got != want {
		t.Errorf("%v: Wrong number of notes; got %v, want %v (proto data: %v)", testName, got, want, allResponses)
	}
	if got, want := countCategoryNotes(allResponses, "PostMessage"), 2; got != want {
		t.Errorf("%v: Wrong number of PostMessage notes; got %v, want %v (proto data: %v)", testName, got, want, allResponses)
	}
	if got, want := countCategoryNotes(allResponses, "JSHint"), 3; got != want {
		t.Errorf("%v: Wrong number of JSHint notes; got %v, want %v (proto data: %v)", testName, got, want, allResponses)
	}
	if got, want := countCategoryNotes(allResponses, "go vet"), 1; got != want {
		t.Errorf("%v: Wrong number of go vet notes; got %v, want %v (proto data: %v)", testName, got, want, allResponses)
	}
	if got, want := countCategoryNotes(allResponses, "PyLint"), 33; got != want {
		t.Errorf("%v: Wrong number of PyLint notes; got %v, want %v (proto data: %v)", testName, got, want, allResponses)
	}
}

// This is a regression test to ensure that when we run the exact same thing twice, it still works.
func TestTwoRunsExactlySame(t *testing.T) {
	TestBuiltInAnalyzersPreBuild(t)
	TestBuiltInAnalyzersPreBuild(t)
}

func TestBuiltInAnalyzersPostBuild(t *testing.T) {
	// Replaces part of the e2e test
	// Test with a kythe maven build
	// PostMessage and ErrorProne
}

func TestChangingDirs(t *testing.T) {
	tests := []struct {
		name           string
		file           string
		expectedJSHint int
		expectedGovet  int
		expectedPyLint int
		expectRestart  bool
	}{
		{
			name:           "ChildDir",
			file:           "shipshape/cli/testdata/workspace2/subworkspace1",
			expectedJSHint: 0,
			expectedGovet:  1,
			expectedPyLint: 0,
			expectRestart:  true,
		},
		{
			name:           "SiblingDir",
			file:           "shipshape/cli/testdata/workspace2/subworkspace2",
			expectedJSHint: 0,
			expectedGovet:  0,
			expectedPyLint: 22,
			expectRestart:  true,
		},
		{
			name:           "ParentDir",
			file:           "shipshape/cli/testdata/workspace2",
			expectedJSHint: 3,
			expectedGovet:  1,
			expectedPyLint: 22,
			expectRestart:  true,
		},
		{
			name:           "File",
			file:           "shipshape/cli/testdata/workspace2/test.js",
			expectedJSHint: 3,
			expectedGovet:  0,
			expectedPyLint: 0,
			expectRestart:  false,
		},
		{
			name:           "ParentToChild",
			file:           "shipshape/cli/testdata/workspace2/subworkspace2",
			expectedJSHint: 0,
			expectedGovet:  0,
			expectedPyLint: 22,
			expectRestart:  false,
		},
		{
			name:           "ParentToOtherChild",
			file:           "shipshape/cli/testdata/workspace2/subworkspace1",
			expectedJSHint: 0,
			expectedGovet:  1,
			expectedPyLint: 0,
			expectRestart:  false,
		},
	}

	// Clean up the docker state
	container := "shipping_container"
	exists, err := docker.ContainerExists(container)
	if err != nil {
		t.Fatalf("Problem checking docker state; err: %v", err)
	}
	if exists {
		if result := docker.Stop(container, 0, true); result.Err != nil {
			t.Fatalf("Problem cleaning up the docker state; err: %v", result.Err)
		}
	}
	oldId := ""

	for _, test := range tests {
		options := Options{
			File:                test.file,
			ThirdPartyAnalyzers: []string{},
			Build:               "",
			TriggerCats:         []string{"PostMessage", "JSHint", "go vet", "PyLint"},
			Dind:                false,
			Event:               defaults.DefaultEvent,
			Repo:                defaults.DefaultRepo,
			StayUp:              true,
			Tag:                 *dockerTag,
			LocalKythe:          *localKythe,
		}
		var allResponses rpcpb.ShipshapeResponse
		options.HandleResponse = func(shipshapeResp *rpcpb.ShipshapeResponse, _ string) error {
			allResponses.AnalyzeResponse =
				append(allResponses.AnalyzeResponse, shipshapeResp.AnalyzeResponse...)
			return nil
		}
		testName := test.name
		if _, err := New(options).Run(); err != nil {
			t.Fatalf("%v: Failure on service call; err: %v", testName, err)
		}
		if got, want := countFailures(allResponses), 0; got != want {
			t.Errorf("%v: Wrong number of failures; got %v, want %v (proto data: %v)",
				testName, got, want, allResponses)
		}
		if got, want := countCategoryNotes(allResponses, "JSHint"), test.expectedJSHint; got != want {
			t.Errorf("%v: Wrong number of JSHint notes; got %v, want %v (proto data: %v)",
				testName, got, want, allResponses)
		}
		if got, want := countCategoryNotes(allResponses, "go vet"), test.expectedGovet; got != want {
			t.Errorf("%v: Wrong number of go vet notes; got %v, want %v (proto data: %v)",
				testName, got, want, allResponses)
		}
		if got, want := countCategoryNotes(allResponses, "PyLint"), test.expectedPyLint; got != want {
			t.Errorf("%v: Wrong number of PyLint notes; got %v, want %v (proto data: %v)",
				testName, got, want, allResponses)
		}
		newId, err := docker.ContainerId(container)
		if err != nil {
			t.Fatalf("%v: Could not get container id: %v", testName, err)
		}
		if got, want := newId != oldId, test.expectRestart; got != want {
			t.Errorf("%v: Incorrect restart status for container. Got %v, want %v", testName, got, want)
		}
		oldId = newId
	}
}

func TestStartService(t *testing.T) {
	tests := []struct {
		name          string
		file          string
		expectRestart bool
	}{
		{
			name:          "ChildDir",
			file:          "shipshape/cli/testdata/workspace2/subworkspace1",
			expectRestart: true,
		},
		{
			name:          "SiblingDir",
			file:          "shipshape/cli/testdata/workspace2/subworkspace2",
			expectRestart: true,
		},
		{
			name:          "ParentDir",
			file:          "shipshape/cli/testdata/workspace2",
			expectRestart: true,
		},
		{
			name:          "SameDir",
			file:          "shipshape/cli/testdata/workspace2",
			expectRestart: false,
		},
		{
			name:          "File",
			file:          "shipshape/cli/testdata/workspace2/test.js",
			expectRestart: false,
		},
		{
			name:          "ParentToChild",
			file:          "shipshape/cli/testdata/workspace2/subworkspace2",
			expectRestart: false,
		},
		{
			name:          "ParentToOtherChild",
			file:          "shipshape/cli/testdata/workspace2/subworkspace1",
			expectRestart: false,
		},
	}

	// Clean up the docker state
	container := "shipping_container"
	exists, err := docker.ContainerExists(container)
	if err != nil {
		t.Fatalf("Problem checking docker state; err: %v", err)
	}
	if exists {
		if result := docker.Stop(container, 0, true); result.Err != nil {
			t.Fatalf("Problem cleaning up the docker state; err: %v", result.Err)
		}
	}
	oldId := ""
	for _, test := range tests {
		options := Options{
			File:                test.file,
			ThirdPartyAnalyzers: []string{},
			Build:               "",
			TriggerCats:         []string{},
			Dind:                false,
			Event:               defaults.DefaultEvent,
			Repo:                defaults.DefaultRepo,
			StayUp:              true,
			Tag:                 *dockerTag,
			LocalKythe:          *localKythe,
		}
		if err := New(options).StartService(); err != nil {
			t.Fatalf("%v: Failure on service call; err: %v", test.name, err)
		}
		newId, err := docker.ContainerId(container)
		if err != nil {
			t.Fatalf("%v: Could not get container id: %v", test.name, err)
		}
		if got, want := newId != oldId, test.expectRestart; got != want {
			t.Errorf("%v: Incorrect restart status for container. Got %v, want %v", test.name, got, want)
		}
		oldId = newId

	}
}
