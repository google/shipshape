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
