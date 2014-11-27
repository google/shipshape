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

package keyvalue

import (
	"testing"

	"code.google.com/p/goprotobuf/proto"

	spb "third_party/kythe/proto/storage_proto"
)

func vname(signature, corpus, root, path, language string) *spb.VName {
	return &spb.VName{
		Signature: &signature,
		Corpus:    &corpus,
		Root:      &root,
		Path:      &path,
		Language:  &language,
	}
}

func TestVNameEncoding(t *testing.T) {
	tests := []*spb.VName{
		nil,
		vname("sig", "corpus", "root", "path", "language"),
		vname("", "", "", "", ""),
		vname("", "kythe", "", "", "java"),
	}

	for _, v := range tests {
		rec, err := encodeVName(v)
		if err != nil {
			t.Errorf("encodeVName: unexpected error: %v", err)
		}

		res, err := decodeVName(rec)
		if err != nil {
			t.Errorf("decodeVName: unexpected error: %v", err)
		}

		if !proto.Equal(res, v) {
			t.Errorf("Decoded VName doesn't match original\n  orig: %+v\ndecoded: %+v", v, res)
		}
	}
}
