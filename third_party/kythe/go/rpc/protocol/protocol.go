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

// Package protocol defines the JSON K-RPC protocol types and constants.
package protocol

import (
	"encoding/json"
	"fmt"
)

// ErrorCode is a number indicating a type of RPC error
type ErrorCode int

// JSON-RPC defined error codes
const (
	// ErrorParsing is an error code meaning invalid JSON was received by the
	// server.  An error occurred on the server while parsing the JSON text.
	ErrorParsing ErrorCode = -32700

	// ErrorInvalidRequest is an error code meaning the JSON sent is not a valid
	// Request object.
	ErrorInvalidRequest = -32600

	// ErrorMethodNotFound is an error code meaning the method does not exist or
	// is not available.
	ErrorMethodNotFound = -32601

	// ErrorInvalidParams is an error code meaning invalid method parameter(s).
	ErrorInvalidParams = -32602

	// ErrorInternal is an error code meaning an internal JSON-RPC error occurred.
	ErrorInternal = -32603

	// ErrorApplication is the default error code for application errors
	ErrorApplication = 0
)

// JSON-RPC version strings
const (
	Version2          = "2.0"
	Version2Streaming = "2.0 streaming"
)

// Request is a JSON-RPC call to a server
type Request struct {
	// Version of the JSON-RPC protocol.
	Version string `json:"jsonrpc"`

	// ID established by the client. If it is not included, the request is assumed
	// to be a notification.
	ID json.RawMessage `json:"id,omitempty"`

	// Method name to be invoked. Method names that begin with the word rpc
	// followed by a period character (U+002E or ASCII 46) are reserved for
	// rpc-internal methods and extensions and MUST NOT be used for anything else.
	Method string `json:"method"`

	// Params is a JSON encoded object or array that holds the parameter values to
	// be used during the invocation of the method. This member MAY be omitted.
	Params json.RawMessage `json:"params,omitempty"`
}

// CheckID returns an error if the given JSON is not a valid ID
func CheckID(id json.RawMessage) error {
	var num json.Number
	err := json.Unmarshal(id, &num)
	if err == nil {
		return nil
	}
	var str string
	err = json.Unmarshal(id, &str)
	if err == nil {
		return nil
	}
	return fmt.Errorf("invalid id: %q", string(id))
}

// Response is a result to a JSON-RPC request.
type Response struct {
	// Version of the JSON-RPC protocol.
	Version string `json:"jsonrpc"`

	// ID established in a corresponding request. If there was an error in
	// detecting the id in the request, it must be left empty.
	ID json.RawMessage `json:"id,omitempty"`

	// Result is a value determined by the server. Result is mutually exclusive
	// with Error.
	Result json.RawMessage `json:"result,omitempty"`

	// Error resulting from a corresponding request. For a Version2Streaming
	// response, this field denotes the immediate end of the stream.
	Error *Error `json:"error,omitempty"`

	Success bool `json:"success,omitempty"`
}

// Error is a descriptor object for an RPC error
type Error struct {
	// Code is a number that indicates the error type that occurred.
	Code ErrorCode `json:"code"`

	// Message is a string providing a short description of the error. The message
	// SHOULD be limited to a concise single sentence.
	Message string `json:"message"`

	// Data is an error value defined by the server (e.g. detailed error
	// information, nested errors etc.).
	Data json.RawMessage `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	return e.Message
}
