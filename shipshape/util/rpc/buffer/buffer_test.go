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

package buffer

import (
	"bytes"
	"io"
	"testing"
)

func TestBuffer(t *testing.T) {
	const allInput = "foobarbazquux"
	const size = 6

	tests := []struct {
		word    string
		flipped bool
	}{
		{"foo", false},
		{"bar", false},
		{"baz", true},
		{"quux", true},
	}
	b := Buffer{Capacity: size}

	totalSize := 0
	for _, test := range tests {
		if got := b.Len(); got != totalSize {
			t.Errorf("Len(): got %d, want %d", got, totalSize)
		}

		n, err := b.Write([]byte(test.word))
		if err != nil {
			t.Errorf("Write %q: unexpected error: %v", test.word, err)
		}
		if n != len(test.word) {
			t.Errorf("Write %q length: got %d, want %d", test.word, n, len(test.word))
		}
		totalSize += n

		if b.disk != test.flipped {
			t.Errorf("Flipped: got %v, want %v", b.disk, test.flipped)
		}
	}

	if b.Path != "" {
		t.Logf("Buffer path is %q", b.Path)
	} else {
		t.Error("Buffer path is empty")
	}

	var check bytes.Buffer
	if n, err := io.Copy(&check, &b); err != nil {
		t.Fatalf("Copying data from buffer failed: %v", err)
	} else {
		t.Logf("Copied %d bytes from buffer", n)
	}

	if got := check.String(); got != allInput {
		t.Errorf("Buffer contents: got %q, want %q", got, allInput)
	}
}
