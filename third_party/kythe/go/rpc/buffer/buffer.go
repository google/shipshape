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

// Package buffer implements a memory buffer with a limited capacity that can
// overflow into a file on disk.
//
// Example:
//   var r io.Reader = …
//   b := buffer.Buffer{Capacity: 10<<20}
//   defer b.Cleanup()
//   if _, err := io.Copy(b, r); err != nil {
//     log.Fatal(err)
//   }
//
//   var w io.Writer = …
//   if _, err = b.WriteTo(w); err != nil {
//     log.Fatal(err)
//   }
package buffer

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
)

// A Buffer is a readable and writable buffer that stores data in memory until
// it exceeds the given capacity in bytes, and thereafter switches to using a
// temporary file on disk.
type Buffer struct {
	// The maximum number of bytes that will be buffered in memory before
	// switching to a file.  If ≤ 0, all data will be stored on disk.
	Capacity int

	// The path of the temporary file that will be used to store overflow.  If
	// this field is "", a path is chosen by ioutil.TempFile when required; the
	// caller may read this field to find out the name that was chosen.
	Path string

	pos  int  // Read position.
	size int  // Number of unread bytes remaining in the buffer.
	disk bool // Whether the data has been flipped to disk.
	bits io.ReadWriteSeeker
}

// Read implements io.Reader.
func (b *Buffer) Read(data []byte) (int, error) {
	if b.bits == nil {
		return 0, io.EOF
	}
	if _, err := b.bits.Seek(int64(b.pos), 0 /* absolute */); err != nil {
		return 0, err
	}
	n, err := b.bits.Read(data)
	b.size -= n
	b.pos += n
	return n, err
}

// Write implements io.Writer.
func (b *Buffer) Write(data []byte) (int, error) {
	total := b.size + len(data)
	if b.bits == nil && total < b.Capacity {
		b.bits = memBuffer{new(bytes.Buffer)}
	}
	if total > b.Capacity {
		if err := b.flip(); err != nil {
			return 0, err
		}
	}
	if _, err := b.bits.Seek(0, 2 /* EOF */); err != nil {
		return 0, err
	}

	n, err := b.bits.Write(data)
	b.size += n
	return n, err
}

func (b *Buffer) flip() error {
	if b.bits != nil && b.disk {
		return nil // Already flipped, nothing to do.
	}

	old := b.bits // In case we need to copy.
	if b.Path != "" {
		f, err := os.OpenFile(b.Path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}
		b.bits = f
	} else if f, err := ioutil.TempFile("", ""); err == nil {
		b.Path = f.Name()
		b.bits = f
	} else {
		return err
	}

	if old != b.bits {
		if _, err := io.Copy(b.bits, old); err != nil {
			return err
		}
	}
	b.disk = true
	return nil
}

// Cleanup cleans up the overflow file created by the buffer, if it exists.
// The caller should either invoke this method, or take responsibility for
// cleaning up the file directly by reading the Path field.
func (b *Buffer) Cleanup() {
	if b.disk && b.Path != "" {
		os.Remove(b.Path) // It's OK if this fails.
	}
}

// Len returns the number of bytes that are stored in the buffer.
func (b *Buffer) Len() int { return b.size }

// memBuffer is a wrapper around *bytes.Buffer to let it satisfy io.Seeker.
type memBuffer struct{ *bytes.Buffer }

// Seek implements io.Seeker.
func (m memBuffer) Seek(pos int64, whence int) (int64, error) { return pos, nil }
