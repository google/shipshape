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

// Package stream implements a Reader and Writer for simple streams of either
// length-delimited byte records (using kythe/rpc/delimited) or
// newline-delimited JSON values.
package stream

import (
	"encoding/json"
	"fmt"
	"io"

	"third_party/kythe/go/platform/delimited"
)

// Reader provides sequential access to a stream of RPC values.
//
// Usage:
//   rd := stream.NewReader(r, false)
//   for {
//     rec, err := rd.Next()
//     if err == io.EOF {
//       break
//     } else if err != nil {
//       log.Fatal(err)
//     }
//     doStuffWith(rec)
//   }
//
type Reader interface {
	// Next returns the next record from the input, or io.EOF if there are no
	// more records available.
	//
	// The slice returned is valid only until a subsequent call to Next.
	Next() ([]byte, error)
}

type jsonReader struct {
	decoder *json.Decoder
}

func (r *jsonReader) Next() ([]byte, error) {
	var data json.RawMessage
	err := r.decoder.Decode(&data)
	return data, err
}

// NewReader constructs a new stream Reader for the records in r. If decodeJSON
// is true, then the Reader is considered a stream of JSON arbitrary values;
// otherwise the Reader is considered a length-delimited stream of byte records.
func NewReader(r io.Reader, decodeJSON bool) Reader {
	if decodeJSON {
		return &jsonReader{json.NewDecoder(r)}
	}
	return delimited.NewReader(r)
}

type transformReader struct {
	rd Reader
	f  func([]byte) ([]byte, error)
}

func (r *transformReader) Next() ([]byte, error) {
	rec, err := r.rd.Next()
	if err != nil {
		return nil, err
	}
	return r.f(rec)
}

// Transform returns a new Reader that applies the given func to each stream
// value before returning it. Errors are passed through.
func Transform(rd Reader, f func([]byte) ([]byte, error)) Reader {
	return &transformReader{rd, f}
}

// A Writer outputs delimited records to an io.Writer.
//
// Basic usage:
//   wr := stream.NewWriter(w, false)
//   for record := range records {
//      if err := wr.Put(record); err != nil {
//        log.Fatal(err)
//      }
//   }
//
type Writer interface {
	// Put writes the specified record to the Writer's stream.
	Put(data []byte) error
}

type lineWriter struct {
	w io.Writer
}

func (w *lineWriter) Put(data []byte) error {
	_, err := fmt.Fprintln(w.w, string(data))
	return err
}

// NewWriter constructs a new stream Writer that writes records to w. If
// lineFormat is true, each record will be followed by a newline; otherwise each
// record is prefixed by a varint representing its length in bytes.
func NewWriter(w io.Writer, lineFormat bool) Writer {
	if lineFormat {
		return &lineWriter{w}
	}
	return delimited.NewWriter(w)
}
