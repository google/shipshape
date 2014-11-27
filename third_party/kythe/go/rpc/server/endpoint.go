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

package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"third_party/kythe/go/rpc/protocol"
	"third_party/kythe/go/util/httpencoding"
)

// An Endpoint is a collection of services that implements http.Handler to
// dispatch KRPC requests to the appropriate service.  Each Endpoint handles
// the ServerInfo/List request as a special case.
type Endpoint []*Service

const (
	serverInfoService = "ServerInfo"
	listMethod        = "List"
)

// Resolve returns the *Method corresponding to the given service and method
// name.  Returns an error if either the service or the method was not found.
//
// Note that Resolve doesn't know about the server info meta-service; that is
// handled separately by the endpoint.
func (e Endpoint) Resolve(serviceName, methodName string) (*Method, error) {
	// Try to find the service.
	var service *Service
	for _, s := range e {
		if s.Name == serviceName {
			service = s
			break
		}
	}
	if service == nil {
		return nil, ErrNoSuchService
	}

	// Try to find the method.
	method := service.Method(methodName)
	if method == nil {
		return nil, ErrNoSuchMethod
	}

	return method, nil
}

// serviceList returns a JSON object summarizing the available services supported by this endpoint.
func (e Endpoint) serviceList() ([]byte, error) {
	return json.Marshal(append(e, &Service{
		Name: serverInfoService,
		Methods: []*Method{
			{Name: listMethod, Params: []string{}},
		},
	}))
}

// parseServiceMethod returns the ServiceName and MethodName of a service method URI string.
func parseServiceMethod(uri string) (service, method string, err error) {
	// The path must have the format "ServiceName/MethodName".
	parts := strings.SplitN(strings.Trim(uri, "/"), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		err = fmt.Errorf("misformatted service method URI: %q", uri)
		return
	}
	service, method = parts[0], parts[1]
	return
}

type responseWriter struct {
	req *protocol.Request
	en  *json.Encoder

	results  uint
	finished bool
}

func newResponseWriter(req *protocol.Request, en *json.Encoder) *responseWriter {
	return &responseWriter{req: req, en: en}
}

func (w *responseWriter) write(resp *protocol.Response) error {
	if w.results > 0 && w.req.Version != protocol.Version2Streaming {
		return errors.New("attempt to write multiple results in non-streaming protocol")
	} else if w.finished {
		return errors.New("attempt to write result after stream is complete")
	}
	resp.ID = w.req.ID
	resp.Version = w.req.Version
	if err := w.en.Encode(resp); err != nil {
		return fmt.Errorf("error encoding response: %v", err)
	}
	w.results++
	return nil
}

// pError writes a protocol error response
func (w *responseWriter) pError(err *protocol.Error) error {
	if err := w.write(&protocol.Response{Error: err}); err != nil {
		return err
	}
	w.finished = true
	return nil
}

// Error writes an error response
func (w *responseWriter) Error(code protocol.ErrorCode, msg string) error {
	return w.pError(&protocol.Error{
		Code:    code,
		Message: msg,
	})
}

// Result writes a result response
func (w *responseWriter) Result(result json.RawMessage) error {
	return w.write(&protocol.Response{Result: result})
}

// Success writes a streaming success response
func (w *responseWriter) Success() error {
	if err := w.write(&protocol.Response{Success: true}); err != nil {
		return err
	}
	w.finished = true
	return nil
}

func (e Endpoint) handleRequest(ctx Context, de *json.Decoder, en *json.Encoder) (err error) {
	var req protocol.Request

	wr := newResponseWriter(&req, en)
	defer func() {
		// On a panic, write error response
		if e := recover(); e != nil {
			switch e := e.(type) {
			case *protocol.Error:
				err = wr.pError(e)
			case error:
				err = wr.Error(protocol.ErrorInternal, e.Error())
			default:
				err = wr.Error(protocol.ErrorInternal, fmt.Sprintf("%+v", e))
			}
		}
	}()

	// Decode request
	if err := de.Decode(&req); err == io.EOF {
		return err
	} else if err != nil {
		if e := wr.Error(protocol.ErrorParsing, fmt.Sprintf("error decoding JSON: %v", err)); e != nil {
			return fmt.Errorf("failed to write error response for %v: %v", err, e)
		}
		return err
	}

	// Validate request
	if err := protocol.CheckID(req.ID); err != nil {
		return wr.Error(protocol.ErrorInvalidRequest, err.Error())
	} else if req.Version != protocol.Version2 && req.Version != protocol.Version2Streaming {
		return wr.Error(protocol.ErrorInvalidRequest, fmt.Sprintf("invalid protocol version: %q", req.Version))
	}

	serviceName, methodName, err := parseServiceMethod(req.Method)
	if err != nil {
		return wr.Error(protocol.ErrorMethodNotFound,
			fmt.Sprintf("Unknown method name format: %q", req.Method))
	}

	// Handle builtin /ServerInfo service
	if serviceName == serverInfoService {
		req.Version = protocol.Version2 // downgrade version for ServerInfo methods
		if methodName != listMethod {
			return wr.Error(protocol.ErrorMethodNotFound,
				fmt.Sprintf("%q method not found in builtin service %q", methodName, serverInfoService))
		}

		spec, err := e.serviceList()
		if err != nil {
			panic(err) // shouldn't happen, internal server error
		}

		return wr.Result(spec)
	}

	// Resolve method
	method, err := e.Resolve(serviceName, methodName)
	if err != nil {
		return wr.Error(protocol.ErrorMethodNotFound,
			fmt.Sprintf("Method not found: /%s/%s", serviceName, methodName))
	}

	if !method.Stream {
		// Ensure downgraded version if method returns a single result
		req.Version = protocol.Version2
	}

	// Construct method output handler
	var (
		results [][]byte // For Version2 array result
		out     func(result []byte)
		outErr  error
	)
	switch req.Version {
	case protocol.Version2:
		out = func(result []byte) {
			results = append(results, result)
		}
	case protocol.Version2Streaming:
		out = func(result []byte) {
			if outErr != nil {
				return
			}
			outErr = wr.Result(result)
		}
	default:
		panic(fmt.Errorf("Unhandled version outputs: %q", req.Version))
	}

	// Invoke method with params
	err = method.Invoke(ctx, req.Params, out)
	if outErr != nil {
		panic(outErr)
	}
	if err != nil {
		switch err := err.(type) {
		case *protocol.Error:
			return wr.pError(err)
		default:
			return wr.Error(protocol.ErrorApplication, err.Error())
		}
	}

	switch req.Version {
	case protocol.Version2:
		if method.Stream {
			// If the client requested V2, but the method is streaming, we must
			// collect the results into a JSON array.
			arry := append(append([]byte("["), bytes.Join(results, []byte(","))...), []byte("]")...)
			return wr.Result(arry)
		} else if len(results) == 1 {
			// If the client requested V2 and the method is non-streaming, we just
			// return the result.
			return wr.Result(results[0])
		} else {
			// This shouldn't happen due to the way method registration works; a
			// non-streaming method MUST return a single result.
			log.Fatalf("Found non-singleton result for non-stream method: %v", results)
		}
	case protocol.Version2Streaming:
		// We already wrote the results, just end the stream.
		return wr.Success()
	default:
		// This case should never happen if we've correctly handled all protocol
		// versions.
		panic(fmt.Errorf("Unexpected protocol version: %q", req.Version))
	}

	panic("unexpected end of request handler")
}

// ServePipes implements the rpc protocol over an input and output stream
func (e Endpoint) ServePipes(ctx Context, r io.Reader, w io.Writer) error {
	de := json.NewDecoder(r)
	en := json.NewEncoder(w)
	for {
		if err := e.handleRequest(ctx, de, en); err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
	}
}

// ServeHTTP implements the http.Handler interface.
func (e Endpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// All requests to a KRPC endpoint must use the POST method.
	if r.Method != "POST" {
		http.Error(w, "Method must be POST", http.StatusMethodNotAllowed)
		return
	}

	cw := httpencoding.CompressData(w, r)
	defer cw.Close()

	de := json.NewDecoder(r.Body)
	en := json.NewEncoder(cw)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := e.handleRequest(r.Header, de, en); err != nil {
		log.Printf("HTTP RPC Error: %v", err)
	}
}
