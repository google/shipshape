// Package keyvalue implements a generic GraphStore for anything that implements
// the DB interface.
package keyvalue

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"third_party/kythe/go/storage"

	"code.google.com/p/goprotobuf/proto"

	spb "third_party/kythe/proto/storage_proto"
)

// A Store implements the storage.GraphStore interface for a keyvalue DB
type Store struct{ DB }

// A DB is a sorted key-value store with read/write access. DBs must be Closed
// when no longer used to ensure resources are not leaked.
type DB interface {
	io.Closer

	// Reader returns an Iterator for a short read starting at a given key
	Reader([]byte) (Iterator, error)
	// Scanner returns an Iterator for full database read
	Scanner() (Iterator, error)
	// Writer return a new write-access object
	Writer() (Writer, error)
}

// Iterator provides sequential access to a DB. Iterators must be Closed when
// no longer used to ensure that resources are not leaked.
type Iterator interface {
	io.Closer

	// Next returns the currently positioned key-value entry and moves to the next
	// entry. If there is no key-value entry to return, an io.EOF error is
	// returned.
	Next() (key, val []byte, err error)
}

// Writer provides write access to a DB. Writes must be Closed when no longer
// used to ensure that resources are not leaked.
type Writer interface {
	io.Closer

	// Write writes a key-value entry to the DB. Writes may be batched until the
	// Writer is Closed.
	Write(key, val []byte) error
}

// Read implements a GraphStore's Read operation.
func (s *Store) Read(req *spb.ReadRequest, stream chan<- *spb.Entry) error {
	keyPrefix, err := KeyPrefix(req.Source, req.GetEdgeKind())
	if err != nil {
		return fmt.Errorf("invalid ReadRequest: %v", err)
	}
	iter, err := s.Reader(keyPrefix)
	if err != nil {
		return fmt.Errorf("db seek error: %v", err)
	}
	defer iter.Close()
	for {
		key, val, err := iter.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("db iteration error: %v", err)
		} else if !bytes.HasPrefix(key, keyPrefix) {
			break
		}

		entry, err := Entry(key, val)
		if err != nil {
			return fmt.Errorf("encoding error: %v", err)
		}
		stream <- entry
	}
	return nil
}

// Write implements a GraphStore's Write operation.
func (s *Store) Write(req *spb.WriteRequest) (err error) {
	wr, err := s.Writer()
	if err != nil {
		return fmt.Errorf("db writer error: %v", err)
	}
	defer func() {
		cErr := wr.Close()
		if err == nil && cErr != nil {
			err = fmt.Errorf("db writer close error: %v", cErr)
		}
	}()
	for _, update := range req.Update {
		if update.GetFactName() == "" {
			return fmt.Errorf("invalid WriteRequest: Update missing FactName")
		}
		updateKey, err := EncodeKey(req.Source, update.GetFactName(), update.GetEdgeKind(), update.Target)
		if err != nil {
			return fmt.Errorf("encoding error: %v", err)
		}
		if err := wr.Write(updateKey, update.FactValue); err != nil {
			return fmt.Errorf("db write error: %v", err)
		}
	}
	return nil
}

// Scan implements a GraphStore's Scan operation.
func (s *Store) Scan(req *spb.ScanRequest, stream chan<- *spb.Entry) error {
	iter, err := s.Scanner()
	if err != nil {
		return fmt.Errorf("db seek error: %v", err)
	}
	defer iter.Close()
	for {
		key, val, err := iter.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("db iteration error: %v", err)
		}
		entry, err := Entry(key, val)
		if err != nil {
			return fmt.Errorf("invalid key/value entry: %v", err)
		}

		if storage.EntryMatchesScan(req, entry) {
			stream <- entry
		}
	}
	return nil
}

const (
	// entryKeySep is used to separate the source, factName, edgeKind, and target of an
	// encoded Entry key
	entryKeySep = '\n'

	// vNameFieldSep is used to separate the fields of an encoded VName
	vNameFieldSep = "\000"
)

// EncodeKey returns a canonical encoding of an Entry (minus its value).
func EncodeKey(source *spb.VName, factName string, edgeKind string, target *spb.VName) ([]byte, error) {
	if source == nil {
		return nil, fmt.Errorf("missing source VName for key encoding")
	} else if (edgeKind == "" || target == nil) && (edgeKind != "" || target != nil) {
		return nil, fmt.Errorf("edgeKind and target Ticket must be both non-empty or empty")
	} else if strings.IndexRune(edgeKind, entryKeySep) != -1 {
		return nil, fmt.Errorf("edgeKind contains key separator")
	} else if strings.IndexRune(factName, entryKeySep) != -1 {
		return nil, fmt.Errorf("factName contains key separator")
	}

	keySuffix := bytes.Join([][]byte{
		nil, []byte(edgeKind), []byte(factName), nil,
	}, []byte{entryKeySep})

	srcEncoding, err := encodeVName(source)
	if err != nil {
		return nil, fmt.Errorf("error encoding source VName: %v", err)
	} else if bytes.IndexRune(srcEncoding, entryKeySep) != -1 {
		return nil, fmt.Errorf("source VName contains key separator %v", source)
	}
	targetEncoding, err := encodeVName(target)
	if err != nil {
		return nil, fmt.Errorf("error encoding target VName: %v", err)
	} else if bytes.IndexRune(targetEncoding, entryKeySep) != -1 {
		return nil, fmt.Errorf("target VName contains key separator")
	}

	return append(append(srcEncoding, keySuffix...), targetEncoding...), nil
}

// KeyPrefix returns a prefix to every encoded key for the given source VName and exact
// edgeKind. If edgeKind is "*", the prefix will match any edgeKind.
func KeyPrefix(source *spb.VName, edgeKind string) ([]byte, error) {
	if source == nil {
		return nil, fmt.Errorf("missing source VName")
	}
	srcEncoding, err := encodeVName(source)
	if err != nil {
		return nil, fmt.Errorf("error encoding source VName: %v", err)
	}

	prefix := append(srcEncoding, entryKeySep)
	if edgeKind == "*" {
		return prefix, nil
	}

	return append(prefix, append([]byte(edgeKind), entryKeySep)...), nil
}

// Entry decodes the key (assuming it was encoded by EncodeKey) into an Entry
// and populates its value field.
func Entry(key []byte, val []byte) (*spb.Entry, error) {
	keyParts := bytes.Split(key, []byte{entryKeySep})
	if len(keyParts) != 4 {
		return nil, fmt.Errorf("invalid key[%d]: %q", len(keyParts), string(key))
	}

	srcVName, err := decodeVName(keyParts[0])
	if err != nil {
		return nil, fmt.Errorf("error decoding source VName: %v", err)
	}
	targetVName, err := decodeVName(keyParts[3])
	if err != nil {
		return nil, fmt.Errorf("error decoding target VName: %v", err)
	}

	return &spb.Entry{
		Source:    srcVName,
		FactName:  proto.String(string(keyParts[2])),
		EdgeKind:  proto.String(string(keyParts[1])),
		Target:    targetVName,
		FactValue: val,
	}, nil
}

// encodeVName returns a canonical byte array for the given VName. Returns nil if given nil.
func encodeVName(v *spb.VName) ([]byte, error) {
	if v == nil {
		return nil, nil
	} else if strings.Index(v.GetSignature(), vNameFieldSep) != -1 ||
		strings.Index(v.GetCorpus(), vNameFieldSep) != -1 ||
		strings.Index(v.GetRoot(), vNameFieldSep) != -1 ||
		strings.Index(v.GetPath(), vNameFieldSep) != -1 ||
		strings.Index(v.GetLanguage(), vNameFieldSep) != -1 {
		return nil, fmt.Errorf("VName contains invalid rune: %q", vNameFieldSep)
	}
	return []byte(strings.Join([]string{
		v.GetSignature(),
		v.GetCorpus(),
		v.GetRoot(),
		v.GetPath(),
		v.GetLanguage(),
	}, vNameFieldSep)), nil
}

// decodeVName returns the VName coded in the given byte array. Returns nil, if len(data) == 0.
func decodeVName(data []byte) (*spb.VName, error) {
	if len(data) == 0 {
		return nil, nil
	}
	parts := strings.Split(string(data), vNameFieldSep)
	if len(parts) != 5 {
		return nil, fmt.Errorf("invalid VName encoding: %q", string(data))
	}
	return &spb.VName{
		Signature: &parts[0],
		Corpus:    &parts[1],
		Root:      &parts[2],
		Path:      &parts[3],
		Language:  &parts[4],
	}, nil
}
