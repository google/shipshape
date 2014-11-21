package service

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"third_party/kythe/go/platform/analysis"

	"code.google.com/p/goprotobuf/proto"

	"third_party/kythe/go/rpc/client"
	"third_party/kythe/go/rpc/server"
	"third_party/kythe/go/rpc/stream"

	apb "third_party/kythe/proto/analysis_proto"
	spb "third_party/kythe/proto/storage_proto"
)

type File struct {
	path, digest string
	content      []byte
}

func (f File) proto() *apb.CompilationUnit_FileInput {
	return &apb.CompilationUnit_FileInput{
		Info: &apb.FileInfo{
			Path:   proto.String(f.path),
			Digest: proto.String(f.digest),
		},
	}
}

type fakeFDS struct {
	Files []File
}

var errFileNotFound = errors.New("file not found")

func (f fakeFDS) Get(ctx server.Context, info *apb.FileInfo, out chan<- *apb.FileData) error {
	for _, got := range f.Files {
		if got.path == info.GetPath() && got.digest == info.GetDigest() {
			out <- &apb.FileData{Info: info, Content: got.content}
			return nil
		}
	}
	fmt.Printf("file not found path:%v digest:%v\n", info.GetPath(), info.GetDigest())
	return errFileNotFound
}

func (f fakeFDS) mustFind(path string) File {
	for _, file := range f.Files {
		if file.path == path {
			return file
		}
	}
	panic("file not found")
}

var testFDS = fakeFDS{
	Files: []File{
		{"path/to/foo", "012345", []byte("alpha")},
		{"path/to/bar", "6789ab", []byte("bravo")},
		{"some/other/file", "cdef0123", []byte("charlie")},
		{"call_the_police", "456789a", []byte("delta")},
	},
}

type fakeAnalyzer struct {
	req    *apb.AnalysisRequest // Most recently handled analysis request
	inputs []File               // Required inputs fetched

	outputs []string // Outputs returned via the stream
	message string   // If nonempty, mark as incomplete with this message.
}

func (a *fakeAnalyzer) Analyze(req *apb.AnalysisRequest, f analysis.Fetcher, s analysis.Sink) error {
	// Record the compilation that was analyzed.
	a.req = req

	// Fetch all the required inputs.
	for _, r := range req.Compilation.RequiredInput {
		path, digest := r.GetInfo().GetPath(), r.GetInfo().GetDigest()
		data, err := f.Fetch(path, digest)
		if err != nil {
			return err
		}
		a.inputs = append(a.inputs, File{
			path:    path,
			digest:  digest,
			content: data,
		})
	}

	// Transmit all the expected outputs.
	for _, output := range a.outputs {
		if err := s.WriteBytes([]byte(output)); err != nil {
			return err
		}
	}

	if a.message != "" {
		return errors.New(a.message)
	}
	return nil
}

func newServer(cas *Service, fds fakeFDS) (server.Endpoint, error) {
	casServer := server.Service{Name: "CompilationAnalyzer"}
	if err := casServer.Register(cas); err != nil {
		return nil, err
	}

	fdsServer := server.Service{Name: "FileData"}
	if err := fdsServer.Register(fds); err != nil {
		return nil, err
	}

	return server.Endpoint{&casServer, &fdsServer}, nil
}

func TestAnalysisService(t *testing.T) {
	tests := []struct {
		outputs []string
		message string // Status message, if non-empty
		error   string // Error substring, if non-empty
		unit    *apb.CompilationUnit
	}{
		// A compilation with no required inputs.
		{outputs: []string{"four", "", "six"}, unit: &apb.CompilationUnit{
			VName: &spb.VName{Signature: proto.String("//test:no_inputs")},
		}},

		// A compilation with three required inputs, all matching known files.
		{outputs: []string{"one", "two", "three"}, unit: &apb.CompilationUnit{
			VName: &spb.VName{Signature: proto.String("//test:thingy")},
			RequiredInput: []*apb.CompilationUnit_FileInput{
				testFDS.mustFind("path/to/foo").proto(),
				testFDS.mustFind("path/to/bar").proto(),
				testFDS.mustFind("call_the_police").proto(),
			},
		}},

		// A compilation with two required inputs, one of which is unknown.
		{error: "file not found", unit: &apb.CompilationUnit{
			VName: &spb.VName{Signature: proto.String("//test:whatzit")},
			RequiredInput: []*apb.CompilationUnit_FileInput{
				testFDS.mustFind("some/other/file").proto(),
				{Info: &apb.FileInfo{Path: proto.String("no such file"), Digest: proto.String("whatever")}},
			},
		}},

		// A bogus required input that is missing its digest field.
		// This should result in an RPC error.
		{error: "file not found", unit: &apb.CompilationUnit{
			VName: &spb.VName{Signature: proto.String("//test:bogus_input")},
			RequiredInput: []*apb.CompilationUnit_FileInput{
				{Info: &apb.FileInfo{Path: proto.String("ok")}}, // Missing digest field.
			},
		}},
	}
	fa := new(fakeAnalyzer)
	endpoints, err := newServer(&Service{Analyzer: fa, FloatingClients: 1}, testFDS)
	if err != nil {
		t.Fatalf("Error while registering servers: %v", err)
	}

	// Startup the server
	addr := fmt.Sprintf(":7000")
	http.Handle("/", endpoints)
	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			t.Fatalf("Server startup failed: %v", err)
		}
	}()

	for _, test := range tests {
		// Set up the values needed by the fake.
		fa.req = nil
		fa.inputs = nil
		fa.outputs = test.outputs
		fa.message = test.message

		c := client.NewHTTPClient(addr)

		// Run the analysis via the fake.
		var outputs []string
		var result error
		stream := make(chan *apb.AnalysisOutput)
		req := &apb.AnalysisRequest{
			Compilation:     test.unit,
			FileDataService: proto.String(addr),
		}

		go remoteCall(c, "/CompilationAnalyzer/Analyze", req, stream, &result)
		for out := range stream {
			outputs = append(outputs, string(out.Value))
		}

		// In case of RPC errors, we won't bother to check the rest of the
		// results, since their values are not expected to make sense.
		// However, we do verify that we got a sensible error.
		if result != nil {
			if test.error == "" || !strings.Contains(result.Error(), test.error) {
				t.Errorf("Unexpected error from analysis: %+v; want %q", result, test.error)
			}
			continue
		}

		// We analyzed the correct compilation.
		if !proto.Equal(fa.req, req) {
			t.Errorf("Analyzis request: got %+v, want %+v", fa.req, req)
		}

		// We got all the expected outputs.
		if !reflect.DeepEqual(outputs, test.outputs) {
			t.Errorf("Analysis outputs: got %+v, want %+v", outputs, test.outputs)
		}

		// The reply status correctly reflects whether there was an analysis
		// error, and if there was no error, the correct inputs were consumed.
		if !checkInputs(fa.inputs, test.unit.RequiredInput) {
			t.Errorf("Analysis inputs: got %+v, want %+v", fa.inputs, test.unit.RequiredInput)
		}
	}
}

func checkInputs(got []File, want []*apb.CompilationUnit_FileInput) bool {
	for _, w := range want {
		found := false
		for _, g := range got {
			if g.path == w.GetInfo().GetPath() && g.digest == w.GetInfo().GetDigest() {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return len(got) == len(want)
}

// remoteCall calls the remote FileStore serviceMethod with the req argument.
func remoteCall(c *client.HTTPClient, serviceMethod string, req proto.Message, out chan<- *apb.AnalysisOutput, result *error) {
	defer close(out)

	resp, err := c.ProtoCall(serviceMethod, req)
	if err != nil {
		*result = err
		return
	}
	defer resp.Close()
	rd := stream.NewReader(resp, false)
	for {
		rec, err := rd.Next()
		if err == io.EOF {
			return
		} else if err != nil {
			*result = fmt.Errorf("remoteCompilationAnalyzerService: stream error: %v", err)
			return
		}

		analysisOutput := new(apb.AnalysisOutput)
		if err := proto.Unmarshal(rec, analysisOutput); err != nil {
			*result = fmt.Errorf("remoteCompilationAnalyzerService: Entry unmarshalling error: %v", err)
			return
		}

		out <- analysisOutput
	}
}
